package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"url-shortener/internal/domain"
	"url-shortener/internal/metrics"

	"github.com/redis/go-redis/v9"
)

// Cache provides caching operations using Redis
// This implements the CACHE-ASIDE PATTERN:
// 1. Check cache first
// 2. If miss, get from database
// 3. Store in cache for next time
type Cache struct {
	client *redis.Client
	ttl    time.Duration
}

// NewCache creates a new Redis cache
func NewCache(client *redis.Client, ttl time.Duration) *Cache {
	return &Cache{
		client: client,
		ttl:    ttl,
	}
}

// GetURL retrieves a URL from cache
// Returns nil if not found (cache miss)
func (c *Cache) GetURL(ctx context.Context, shortCode string) (*domain.URL, error) {
	start := time.Now()
	defer func() {
		metrics.CacheOperationDuration.WithLabelValues("get").Observe(time.Since(start).Seconds())
	}()

	// Key naming convention: "url:{shortCode}"
	key := fmt.Sprintf("url:%s", shortCode)

	// Get from Redis
	data, err := c.client.Get(ctx, key).Result()
	if err == redis.Nil {
		// Cache miss - not an error, just not found
		metrics.RecordCacheMiss()
		return nil, nil
	}
	if err != nil {
		// Actual error (connection issue, etc.)
		return nil, fmt.Errorf("redis get error: %w", err)
	}

	// Cache hit!
	metrics.RecordCacheHit()

	// Deserialize JSON to domain.URL
	var url domain.URL
	if err := json.Unmarshal([]byte(data), &url); err != nil {
		return nil, fmt.Errorf("failed to unmarshal cached URL: %w", err)
	}

	return &url, nil
}

// SetURL stores a URL in cache
func (c *Cache) SetURL(ctx context.Context, shortCode string, url *domain.URL) error {
	key := fmt.Sprintf("url:%s", shortCode)

	// Serialize URL to JSON
	data, err := json.Marshal(url)
	if err != nil {
		return fmt.Errorf("failed to marshal URL: %w", err)
	}

	// Store in Redis with TTL
	// TTL ensures cache doesn't grow indefinitely and stale data is removed
	err = c.client.Set(ctx, key, data, c.ttl).Err()
	if err != nil {
		return fmt.Errorf("redis set error: %w", err)
	}

	return nil
}

// DeleteURL removes a URL from cache
// Used when URL is updated or deleted
func (c *Cache) DeleteURL(ctx context.Context, shortCode string) error {
	key := fmt.Sprintf("url:%s", shortCode)

	err := c.client.Del(ctx, key).Err()
	if err != nil {
		return fmt.Errorf("redis delete error: %w", err)
	}

	return nil
}

// Exists checks if a key exists in cache
func (c *Cache) Exists(ctx context.Context, shortCode string) (bool, error) {
	key := fmt.Sprintf("url:%s", shortCode)

	count, err := c.client.Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("redis exists error: %w", err)
	}

	return count > 0, nil
}

// Clear removes all cached URLs
// Useful for testing or cache invalidation
func (c *Cache) Clear(ctx context.Context) error {
	// Use SCAN to find all url:* keys
	iter := c.client.Scan(ctx, 0, "url:*", 0).Iterator()

	keys := []string{}
	for iter.Next(ctx) {
		keys = append(keys, iter.Val())
	}

	if err := iter.Err(); err != nil {
		return fmt.Errorf("redis scan error: %w", err)
	}

	if len(keys) > 0 {
		err := c.client.Del(ctx, keys...).Err()
		if err != nil {
			return fmt.Errorf("redis delete error: %w", err)
		}
	}

	return nil
}

// GetStats returns cache statistics
func (c *Cache) GetStats(ctx context.Context) (map[string]interface{}, error) {
	info, err := c.client.Info(ctx, "stats").Result()
	if err != nil {
		return nil, fmt.Errorf("redis info error: %w", err)
	}

	// Count cached URLs
	count := 0
	iter := c.client.Scan(ctx, 0, "url:*", 0).Iterator()
	for iter.Next(ctx) {
		count++
	}

	return map[string]interface{}{
		"cached_urls": count,
		"info":        info,
	}, nil
}

// InitRedis creates a new Redis client
func InitRedis(addr, password string, db int) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,

		// Connection pool settings
		PoolSize:     10,              // Maximum number of socket connections
		MinIdleConns: 2,               // Minimum number of idle connections
		MaxRetries:   3,               // Maximum number of retries
		DialTimeout:  5 * time.Second, // Timeout for establishing connection
		ReadTimeout:  3 * time.Second, // Timeout for socket reads
		WriteTimeout: 3 * time.Second, // Timeout for socket writes
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return client, nil
}
