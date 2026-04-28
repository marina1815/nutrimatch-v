local refill_per_second = tonumber(ARGV[1])
local burst = tonumber(ARGV[2])
local now_ms = tonumber(ARGV[3])
local ttl_seconds = tonumber(ARGV[4])

local bucket = redis.call("HMGET", KEYS[1], "tokens", "updated_at")
local tokens = tonumber(bucket[1])
local updated_at = tonumber(bucket[2])

if tokens == nil or updated_at == nil then
  tokens = burst
  updated_at = now_ms
end

if now_ms > updated_at then
  local elapsed = (now_ms - updated_at) / 1000
  tokens = math.min(burst, tokens + (elapsed * refill_per_second))
end

local allowed = 0
if tokens >= 1 then
  tokens = tokens - 1
  allowed = 1
end

redis.call("HSET", KEYS[1], "tokens", tokens, "updated_at", now_ms)
redis.call("EXPIRE", KEYS[1], ttl_seconds)

return allowed
