package storage

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"strconv"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/rampredd/url-shortener/app"
	"github.com/rampredd/url-shortener/base62"
)

type metricResponse struct {
	Url   string `json:"url"`
	Score int    `json:"score"`
}

const (
	metricsKey = "url_metric"
)

// Check if id is already saved in DB
func isUsed(id uint64) bool {
	exists, err := app.RedisDB.Exists(context.Background(), prepareShortKey(id)).Result()
	if err != nil {
		return false
	}
	if exists > 0 {
		return true
	}
	return false
}

// Make key to store short URL in DB
func prepareShortKey(id uint64) string {
	return fmt.Sprintf("short:%s", strconv.FormatUint(id, 10))
}

// Make key to store long URL in DB
func prepareLongKey(url string) string {
	return fmt.Sprintf("long:%s", url)
}

// Save short URL in DB
func Save(url string) (string, error) {
	longKey := prepareLongKey(url)
	key, err := app.RedisDB.Get(context.Background(), longKey).Result()

	if err != redis.Nil && err != nil {
		return "", err
	}

	if key != "" {
		return key, nil
	}

	var id uint64
	var expires time.Time = time.Now().Add(time.Hour * 24)
	for used := true; used; used = isUsed(id) {
		id = rand.Uint64()
	}

	key = prepareShortKey(id)
	shortLink := make(map[string]interface{})
	shortLink["url"] = url

	// Save short link in DB
	_, err = app.RedisDB.HMSet(context.Background(), key, shortLink).Result()
	if err != nil {
		return "", err
	}

	// Set expiration to short link
	err = app.RedisDB.ExpireAt(context.Background(), key, expires).Err()
	if err != nil {
		return "", err
	}

	member := redis.Z{Member: url}

	// Add long URL and it's visit count in DB
	_, err = app.RedisDB.ZAdd(context.Background(), metricsKey, &member).Result()
	if err != nil {
		return "", err
	}

	shortUrl := base62.Encode(id)

	// save in DB with longurl as key
	app.RedisDB.Set(context.Background(), longKey, shortUrl, time.Hour*24)

	return shortUrl, nil
}

func Load(code string) (string, error) {
	decodedId, err := base62.Decode(code)
	if err != nil {
		return "", err
	}

	key := prepareShortKey(decodedId)

	// Get long URL from DB
	urlString, err := app.RedisDB.HGet(context.Background(), key, "url").Result()
	if err != nil {
		return "", err
	} else if len(urlString) == 0 {
		return "", errors.New("No Link")
	}

	// Increment visit count of long URL
	_, err = app.RedisDB.ZIncrBy(context.Background(), metricsKey, float64(1), urlString).Result()
	if err != nil {
		log.Printf("Error in incrementing %s", err.Error())
		return "", err
	}

	return urlString, nil
}

func LoadInfo() ([]metricResponse, error) {
	sortRule := redis.ZRangeBy{
		Min:    "0",
		Max:    "999999999",
		Offset: 0,
		Count:  3,
	}

	// Fetch long URL and its visit count from DB
	values, err := app.RedisDB.ZRevRangeByScoreWithScores(context.Background(), metricsKey, &sortRule).Result()
	if err != nil {
		return nil, err
	} else if len(values) == 0 {
		return nil, errors.New("No metrics available now")
	}

	response := make([]metricResponse, 0)

	// Parse DB response and compose into API response
	for _, v := range values {
		res := metricResponse{Url: v.Member.(string), Score: int(v.Score)}
		response = append(response, res)
	}
	return response, nil
}
