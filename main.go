package main

import (
	"context"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/go-redsync/redsync/v4"
	"github.com/go-redsync/redsync/v4/redis/goredis/v8"

	"flag"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/rampredd/url-shortener/api"
	"github.com/rampredd/url-shortener/app"
	"github.com/rs/cors"
)

func initRedisDb(cfg *app.RedisConfig) (redis.UniversalClient, error) {
	rdb := redis.NewUniversalClient(&redis.UniversalOptions{
		Addrs:    cfg.Addresses,
		Password: cfg.Password,
	})
	return rdb, nil
}

func main() {
	configPtr := flag.String("config", "config.json", "Config file name")
	if err := app.LoadConfig(*configPtr); err != nil {
		log.Fatal(err)
	}

	var err error
	// Redis DB connection
	if app.RedisDB, err = initRedisDb(&app.Config.RedisCfg); err != nil {
		log.Fatalf("Create Redis connection error: %s", err)
	}
	app.RedisSync = redsync.New(goredis.NewPool(app.RedisDB))

	//Register context cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	//List of APIs
	r := mux.NewRouter()
	r.HandleFunc("/shorten-url", api.ShortenUrl).Methods("POST")
	r.HandleFunc("/metrics", api.Metrics).Methods("GET")
	r.HandleFunc("/short-url/{shortlink}", api.Redirect).Methods("GET")

	c := cors.New(cors.Options{
		AllowedOrigins:   app.Config.CorsOrigins,
		AllowedMethods:   []string{"GET", "POST"},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: true,
		ExposedHeaders:   []string{"X-Total-Count"},
		Debug:            false,
	})

	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc,
		syscall.SIGINT,
		syscall.SIGTERM,
	)

	//Wait for signals
	go func() {
		var s os.Signal
		for {
			select {
			case s = <-sigc:
				switch s {
				case syscall.SIGINT, syscall.SIGTERM:
					cancel()
					return
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	//Register service to shutdown http server gracefully
	//http handler will call apiGracefulShutdown
	apiGracefulShutdown := func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			select {
			case <-ctx.Done():
				w.WriteHeader(503)
				w.Write([]byte("Service Unavailable"))
				return
			default:
			}
			app.WgTerminate.Add(1)
			defer func() {
				app.WgTerminate.Done()
			}()
			h.ServeHTTP(w, r.WithContext(ctx))
		})
	}

	var httpServers []*http.Server

	srv := &http.Server{
		Addr:    net.JoinHostPort(app.Config.ListenHost, app.Config.Port),
		Handler: c.Handler(apiGracefulShutdown(r)),
	}
	httpServers = append(httpServers, srv)

	log.Printf("Listening %s:%s ", app.Config.ListenHost, app.Config.Port)

	//Start http server and listen
	go func(srv *http.Server, addr string) {
		if err := srv.ListenAndServe(); err != nil {
			if !errors.Is(err, http.ErrServerClosed) {
				log.Print(err, "Cannot start web server")
				cancel()
			}
		}
	}(srv, srv.Addr)

	//All is done
	<-ctx.Done()

	if len(httpServers) > 0 {
		ctxSrvShutdown, cancelCtxSrvShutdown := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancelCtxSrvShutdown()
		for _, srv := range httpServers {
			srv.Shutdown(ctxSrvShutdown)
		}
	}

	log.Println("exiting")
}
