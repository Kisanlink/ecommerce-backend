package database

import (
	"context"
	"fmt"
	"time"

	"github.com/Kisanlink/ecommerce-backend/internal/config"
	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"
)

// RedisManager manages Redis connections
type RedisManager struct {
	client *redis.Client
	logger *zap.Logger
	ctx    context.Context
}

// NewRedisManager creates a new Redis manager
func NewRedisManager(logger *zap.Logger) (*RedisManager, error) {
	redisConfig := config.GetRedisEnvConfig()

	rdb := redis.NewClient(&redis.Options{
		Addr:         fmt.Sprintf("%s:%s", redisConfig.Host, redisConfig.Port),
		Password:     redisConfig.Password,
		DB:           redisConfig.DB,
		PoolSize:     redisConfig.PoolSize,
		DialTimeout:  time.Duration(redisConfig.DialTimeoutSec) * time.Second,
		ReadTimeout:  time.Duration(redisConfig.ReadTimeoutSec) * time.Second,
		WriteTimeout: time.Duration(redisConfig.WriteTimeoutSec) * time.Second,
		PoolTimeout:  time.Duration(redisConfig.PoolTimeoutSec) * time.Second,
		IdleTimeout:  time.Duration(redisConfig.IdleTimeoutMin) * time.Minute,
		MaxConnAge:   time.Duration(redisConfig.MaxConnAgeMin) * time.Minute,
	})

	ctx := context.Background()

	// Test connection
	_, err := rdb.Ping(ctx).Result()
	if err != nil {
		logger.Error("Failed to connect to Redis", zap.Error(err))
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	logger.Info("Successfully connected to Redis",
		zap.String("host", redisConfig.Host),
		zap.String("port", redisConfig.Port),
		zap.Int("db", redisConfig.DB))

	return &RedisManager{
		client: rdb,
		logger: logger,
		ctx:    ctx,
	}, nil
}

// GetClient returns the Redis client
func (rm *RedisManager) GetClient() *redis.Client {
	return rm.client
}

// GetContext returns the context
func (rm *RedisManager) GetContext() context.Context {
	return rm.ctx
}

// Close closes the Redis connection
func (rm *RedisManager) Close() error {
	if rm.client != nil {
		return rm.client.Close()
	}
	return nil
}

// Set stores a value in Redis with expiration
func (rm *RedisManager) Set(key string, value interface{}, expiration time.Duration) error {
	return rm.client.Set(rm.ctx, key, value, expiration).Err()
}

// Get retrieves a value from Redis
func (rm *RedisManager) Get(key string) (string, error) {
	return rm.client.Get(rm.ctx, key).Result()
}

// Del deletes a key from Redis
func (rm *RedisManager) Del(key string) error {
	return rm.client.Del(rm.ctx, key).Err()
}

// Exists checks if a key exists in Redis
func (rm *RedisManager) Exists(key string) (bool, error) {
	count, err := rm.client.Exists(rm.ctx, key).Result()
	return count > 0, err
}

// SetNX sets a key only if it doesn't exist
func (rm *RedisManager) SetNX(key string, value interface{}, expiration time.Duration) (bool, error) {
	return rm.client.SetNX(rm.ctx, key, value, expiration).Result()
}

// HSet sets a field in a hash
func (rm *RedisManager) HSet(key string, field string, value interface{}) error {
	return rm.client.HSet(rm.ctx, key, field, value).Err()
}

// HGet gets a field from a hash
func (rm *RedisManager) HGet(key string, field string) (string, error) {
	return rm.client.HGet(rm.ctx, key, field).Result()
}

// HGetAll gets all fields from a hash
func (rm *RedisManager) HGetAll(key string) (map[string]string, error) {
	return rm.client.HGetAll(rm.ctx, key).Result()
}

// HDel deletes fields from a hash
func (rm *RedisManager) HDel(key string, fields ...string) error {
	return rm.client.HDel(rm.ctx, key, fields...).Err()
}

// Expire sets expiration on a key
func (rm *RedisManager) Expire(key string, expiration time.Duration) error {
	return rm.client.Expire(rm.ctx, key, expiration).Err()
}

// TTL gets time to live of a key
func (rm *RedisManager) TTL(key string) (time.Duration, error) {
	return rm.client.TTL(rm.ctx, key).Result()
}
