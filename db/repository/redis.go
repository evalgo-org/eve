package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisRepository implements CacheRepository using Redis/Valkey/DragonflyDB
type RedisRepository struct {
	client *redis.Client
}

// NewRedisRepository creates a new Redis-based cache repository
func NewRedisRepository(url string) (*RedisRepository, error) {
	opts, err := redis.ParseURL(url)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Redis URL: %w", err)
	}

	client := redis.NewClient(opts)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &RedisRepository{
		client: client,
	}, nil
}

// Lock operations

func (r *RedisRepository) AcquireLock(ctx context.Context, actionID string, ttl time.Duration) (bool, error) {
	key := "lock:" + actionID
	lockData := map[string]interface{}{
		"actionID": actionID,
		"lockedAt": time.Now().Format(time.RFC3339),
		"ttl":      ttl.String(),
	}

	data, err := json.Marshal(lockData)
	if err != nil {
		return false, err
	}

	// SET key value NX EX ttl_seconds
	// NX = only set if not exists
	result, err := r.client.SetNX(ctx, key, data, ttl).Result()
	if err != nil {
		return false, err
	}

	return result, nil
}

func (r *RedisRepository) ReleaseLock(ctx context.Context, actionID string) error {
	key := "lock:" + actionID
	return r.client.Del(ctx, key).Err()
}

func (r *RedisRepository) IsLocked(ctx context.Context, actionID string) (bool, error) {
	key := "lock:" + actionID
	exists, err := r.client.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}
	return exists > 0, nil
}

// Cache operations

func (r *RedisRepository) SetCache(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	cacheKey := "cache:" + key
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal value: %w", err)
	}

	return r.client.Set(ctx, cacheKey, data, ttl).Err()
}

func (r *RedisRepository) GetCache(ctx context.Context, key string, value interface{}) error {
	cacheKey := "cache:" + key
	data, err := r.client.Get(ctx, cacheKey).Bytes()
	if err == redis.Nil {
		return fmt.Errorf("cache miss: key not found")
	}
	if err != nil {
		return err
	}

	return json.Unmarshal(data, value)
}

func (r *RedisRepository) DeleteCache(ctx context.Context, key string) error {
	cacheKey := "cache:" + key
	return r.client.Del(ctx, cacheKey).Err()
}

// Pub/Sub operations

func (r *RedisRepository) Publish(ctx context.Context, channel string, message interface{}) error {
	data, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	return r.client.Publish(ctx, channel, data).Err()
}

func (r *RedisRepository) Subscribe(ctx context.Context, channel string) (<-chan interface{}, error) {
	pubsub := r.client.Subscribe(ctx, channel)

	// Wait for confirmation
	_, err := pubsub.Receive(ctx)
	if err != nil {
		return nil, err
	}

	// Create output channel
	out := make(chan interface{})

	// Start goroutine to forward messages
	go func() {
		defer close(out)
		defer pubsub.Close()

		ch := pubsub.Channel()
		for {
			select {
			case msg := <-ch:
				if msg == nil {
					return
				}
				var data interface{}
				if err := json.Unmarshal([]byte(msg.Payload), &data); err == nil {
					out <- data
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	return out, nil
}

// Counter operations

func (r *RedisRepository) Increment(ctx context.Context, key string) (int64, error) {
	counterKey := "counter:" + key
	return r.client.Incr(ctx, counterKey).Result()
}

func (r *RedisRepository) Decrement(ctx context.Context, key string) (int64, error) {
	counterKey := "counter:" + key
	return r.client.Decr(ctx, counterKey).Result()
}

// Close closes the Redis connection
func (r *RedisRepository) Close() error {
	return r.client.Close()
}
