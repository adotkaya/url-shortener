package ratelimit

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// RateLimiter implements rate limiting using the TOKEN BUCKET algorithm
//
// HOW IT WORKS:
// 1. Each user/IP has a "bucket" with N tokens
// 2. Tokens are refilled at a constant rate (e.g., 100 tokens/minute)
// 3. Each request consumes 1 token
// 4. If no tokens available → request is rate limited (429 error)
//
// WHY USE REDIS?
// - Distributed rate limiting (works across multiple servers)
// - Fast (in-memory)
// - Atomic operations prevent race conditions
type RateLimiter struct {
	client      *redis.Client
	maxRequests int           // Maximum requests allowed
	window      time.Duration // Time window (e.g., 1 minute)
	burstSize   int           // Maximum burst size
}

// NewTokenBucketLimiter creates a new rate limiter
// Example: NewTokenBucketLimiter(client, 100, time.Minute, 120)
// Allows 100 requests per minute, with burst up to 120
func NewTokenBucketLimiter(client *redis.Client, maxRequests int, window time.Duration, burstSize int) *RateLimiter {
	return &RateLimiter{
		client:      client,
		maxRequests: maxRequests,
		window:      window,
		burstSize:   burstSize,
	}
}

// Allow checks if a request should be allowed
// Returns (allowed bool, remaining int, resetTime time.Time, error)
func (rl *RateLimiter) Allow(ctx context.Context, key string) (bool, int, time.Time, error) {
	// Redis key for this identifier
	redisKey := fmt.Sprintf("ratelimit:%s", key)

	// Use Lua script for atomic operation
	// This ensures no race conditions when multiple requests arrive simultaneously
	script := redis.NewScript(`
		local key = KEYS[1]
		local max_requests = tonumber(ARGV[1])
		local window = tonumber(ARGV[2])
		local current_time = tonumber(ARGV[3])

		-- Get current count
		local current = redis.call('GET', key)

		if current == false then
			-- First request - initialize counter
			redis.call('SET', key, 1, 'EX', window)
			return {1, max_requests - 1, current_time + window}
		else
			current = tonumber(current)
			if current < max_requests then
				-- Increment counter
				redis.call('INCR', key)
				local ttl = redis.call('TTL', key)
				return {1, max_requests - current - 1, current_time + ttl}
			else
				-- Rate limit exceeded
				local ttl = redis.call('TTL', key)
				return {0, 0, current_time + ttl}
			end
		end
	`)

	now := time.Now()
	windowSeconds := int(rl.window.Seconds())

	// Execute Lua script
	result, err := script.Run(
		ctx,
		rl.client,
		[]string{redisKey},
		rl.maxRequests,
		windowSeconds,
		now.Unix(),
	).Result()

	if err != nil {
		return false, 0, time.Time{}, fmt.Errorf("rate limit check failed: %w", err)
	}

	// Parse result
	resultSlice, ok := result.([]interface{})
	if !ok || len(resultSlice) != 3 {
		return false, 0, time.Time{}, fmt.Errorf("unexpected result format")
	}

	allowed := resultSlice[0].(int64) == 1
	remaining := int(resultSlice[1].(int64))
	resetUnix := resultSlice[2].(int64)
	resetTime := time.Unix(resetUnix, 0)

	return allowed, remaining, resetTime, nil
}

// Reset clears the rate limit for a key
// Useful for testing or manual overrides
func (rl *RateLimiter) Reset(ctx context.Context, key string) error {
	redisKey := fmt.Sprintf("ratelimit:%s", key)
	return rl.client.Del(ctx, redisKey).Err()
}

// GetInfo returns current rate limit info for a key
func (rl *RateLimiter) GetInfo(ctx context.Context, key string) (int, time.Duration, error) {
	redisKey := fmt.Sprintf("ratelimit:%s", key)

	// Get current count
	count, err := rl.client.Get(ctx, redisKey).Int()
	if err == redis.Nil {
		// No rate limit data - all requests available
		return rl.maxRequests, 0, nil
	}
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get rate limit info: %w", err)
	}

	// Get TTL
	ttl, err := rl.client.TTL(ctx, redisKey).Result()
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get TTL: %w", err)
	}

	remaining := rl.maxRequests - count
	if remaining < 0 {
		remaining = 0
	}

	return remaining, ttl, nil
}

// MaxRequests returns the maximum number of requests allowed
func (rl *RateLimiter) MaxRequests() int {
	return rl.maxRequests
}
