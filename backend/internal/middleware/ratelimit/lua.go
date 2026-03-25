package ratelimit

// Lua scripts ensure atomicity of rate limit operations in Redis.
// This prevents race conditions when multiple instances check and update limits concurrently.

// slidingWindowScript implements the sliding window log algorithm.
// KEYS[1] = rate limit key (sorted set)
// ARGV[1] = current timestamp in microseconds
// ARGV[2] = window start timestamp in microseconds
// ARGV[3] = limit
// ARGV[4] = TTL in seconds for the key
// ARGV[5] = unique request ID (member for the sorted set)
//
// Returns: {allowed (0/1), current_count, ttl_remaining_ms}
const slidingWindowScript = `
local key = KEYS[1]
local now = tonumber(ARGV[1])
local window_start = tonumber(ARGV[2])
local limit = tonumber(ARGV[3])
local ttl = tonumber(ARGV[4])
local request_id = ARGV[5]

-- Remove expired entries outside the window
redis.call('ZREMRANGEBYSCORE', key, '-inf', window_start)

-- Get current count
local current = redis.call('ZCARD', key)

if current < limit then
    -- Add the new request
    redis.call('ZADD', key, now, request_id)
    redis.call('EXPIRE', key, ttl)
    return {1, current + 1, -1}
else
    -- Get the oldest entry to calculate retry-after
    local oldest = redis.call('ZRANGE', key, 0, 0, 'WITHSCORES')
    local retry_after = -1
    if #oldest > 0 then
        local oldest_score = tonumber(oldest[2])
        retry_after = oldest_score + (tonumber(ARGV[4]) * 1000000) - now
        if retry_after < 0 then
            retry_after = 0
        end
    end
    return {0, current, retry_after}
end
`

// fixedWindowScript implements the fixed window counter algorithm.
// KEYS[1] = rate limit key (string counter)
// ARGV[1] = limit
// ARGV[2] = TTL in seconds
//
// Returns: {allowed (0/1), current_count, ttl_remaining_ms}
const fixedWindowScript = `
local key = KEYS[1]
local limit = tonumber(ARGV[1])
local ttl = tonumber(ARGV[2])

local current = tonumber(redis.call('GET', key) or '0')

if current < limit then
    current = redis.call('INCR', key)
    -- Only set TTL on first request in window
    if current == 1 then
        redis.call('EXPIRE', key, ttl)
    end
    local remaining_ttl = redis.call('PTTL', key)
    return {1, current, remaining_ttl}
else
    local remaining_ttl = redis.call('PTTL', key)
    return {0, current, remaining_ttl}
end
`

// tokenBucketScript implements the token bucket algorithm.
// KEYS[1] = rate limit key (hash with tokens and last_refill)
// ARGV[1] = max_tokens (bucket capacity / burst limit)
// ARGV[2] = refill_rate (tokens per second)
// ARGV[3] = current timestamp in microseconds
// ARGV[4] = TTL in seconds
//
// Returns: {allowed (0/1), remaining_tokens, retry_after_us}
// #nosec G101
const tokenBucketScript = `
local key = KEYS[1]
local max_tokens = tonumber(ARGV[1])
local refill_rate = tonumber(ARGV[2])
local now = tonumber(ARGV[3])
local ttl = tonumber(ARGV[4])

local bucket = redis.call('HMGET', key, 'tokens', 'last_refill')
local tokens = tonumber(bucket[1])
local last_refill = tonumber(bucket[2])

if tokens == nil then
    -- Initialize bucket
    tokens = max_tokens
    last_refill = now
end

-- Calculate tokens to add based on elapsed time
local elapsed = (now - last_refill) / 1000000  -- Convert microseconds to seconds
local new_tokens = elapsed * refill_rate
tokens = math.min(max_tokens, tokens + new_tokens)
last_refill = now

if tokens >= 1 then
    tokens = tokens - 1
    redis.call('HMSET', key, 'tokens', tokens, 'last_refill', last_refill)
    redis.call('EXPIRE', key, ttl)
    return {1, math.floor(tokens), -1}
else
    -- Calculate retry after: time until 1 token is available
    local deficit = 1 - tokens
    local retry_after = (deficit / refill_rate) * 1000000  -- in microseconds
    redis.call('HMSET', key, 'tokens', tokens, 'last_refill', last_refill)
    redis.call('EXPIRE', key, ttl)
    return {0, 0, math.floor(retry_after)}
end
`
