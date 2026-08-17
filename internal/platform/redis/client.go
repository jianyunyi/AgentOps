package redis

import redisv9 "github.com/redis/go-redis/v9"

const AnalysisStream = "agentscope:analysis:v1"

func NewClient(addr string) *redisv9.Client {
	return redisv9.NewClient(&redisv9.Options{Addr: addr})
}
