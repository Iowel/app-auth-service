package cache

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/Iowel/app-auth-service/internal/domain"
	"github.com/redis/go-redis/v9"
)

type redisCache struct {
	host    string
	db      int
	expires time.Duration
}

func NewRedisCache(host string, db int, exp time.Duration) IPostCache {
	return &redisCache{
		host:    host,
		db:      db,
		expires: exp,
	}
}

func (cache *redisCache) getClient() *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:     cache.host,
		Password: "",
		DB:       cache.db,
	})
}

func (cache *redisCache) Set(key string, value *domain.UserCache) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// создаем клиента
	client := cache.getClient()

	// парсим полученное сообщение в json
	json, err := json.Marshal(value)
	if err != nil {
		log.Printf("failed to marshal user: %v", err)
		return
	}

	// устанавливаем ключ
	client.Set(ctx, key, string(json), cache.expires*time.Second)
}

func (cache *redisCache) Get(key string) *domain.UserCache {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// создаем клиента
	client := cache.getClient()

	// получаем значение
	val, err := client.Get(ctx, key).Result()
	if err != nil {
		return nil
	}

	// присваиваем полученное значение
	user := domain.UserCache{}

	// распаковываем json сохраненный в кэше редис
	err = json.Unmarshal([]byte(val), &user)
	if err != nil {
		log.Printf("failed to unmarshall user: %v", err)
	}

	return &user
}

func (cache *redisCache) GetAll() []*domain.UserCache {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// создаем клиента
	client := cache.getClient()

	var point uint64
	var users []*domain.UserCache

	for {
		keys, nextPoint, err := client.Scan(ctx, point, "user:*", 100).Result()
		if err != nil {
			log.Printf("failed to scan keys from redis: %v", err)
			break
		}

		for _, key := range keys {
			val, err := client.Get(ctx, key).Result()
			if err != nil {
				log.Printf("failed to get value for key %s: %v", key, err)
				continue
			}

			var u domain.UserCache
			if err := json.Unmarshal([]byte(val), &u); err != nil {
				log.Printf("failed to unmarshal user from key %s: %v", key, err)
				continue
			}

			users = append(users, &u)
		}

		if nextPoint == 0 {
			break
		}
		point = nextPoint
	}

	return users

}

func (cache *redisCache) Delete(key string) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	client := cache.getClient()

	_, err := client.Del(ctx, key).Result()
	if err != nil {
		log.Printf("failed to delete key %s: %v", key, err)
	}
}
