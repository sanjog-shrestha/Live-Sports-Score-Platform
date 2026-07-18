package main

import (
	"time"

	"github.com/redis/go-redis/v9"
)

const scoresCacheKey = "scores:all"
const scoresCacheTTL = 30 * time.Second

var redisClient = redis.NewClient(&redis.Options{
	Addr: getRedisAddr(),
})
