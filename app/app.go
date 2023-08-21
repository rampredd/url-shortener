package app

import (
	"encoding/json"
	"log"
	"os"
	"sync"

	"github.com/go-redis/redis/v8"
	"github.com/go-redsync/redsync/v4"
)

type RedisConfig struct {
	Addresses []string `json:"cluster"`
	Password  string   `json:"password"`
}

type AppConfig struct {
	CorsOrigins []string    `json:"cors_origins"`
	ListenHost  string      `json:"listen_host"`
	Port        string      `json:"port"`
	RedisCfg    RedisConfig `json:"redis"`
}

var (
	Config      *AppConfig
	RedisDB     redis.UniversalClient
	WgTerminate sync.WaitGroup
	RedisSync   *redsync.Redsync
)

func LoadConfig(location string) error {
	log.Printf("Loading config file %s", location)
	var config *AppConfig
	f, e := os.Open(location)
	if e != nil {
		return e
	}
	defer f.Close()

	if e := json.NewDecoder(f).Decode(&config); e != nil {
		return e
	}

	Config = config

	return nil
}
