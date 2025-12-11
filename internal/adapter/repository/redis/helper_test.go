package redis

import (
	"testing"

	miniredis "github.com/alicebob/miniredis/v2"
	redislib "github.com/redis/go-redis/v9"
)

func newTestRedisClient(t *testing.T) (*redislib.Client, *miniredis.Miniredis) {
	t.Helper()

	mr := miniredis.RunT(t)
	client := redislib.NewClient(&redislib.Options{
		Addr: mr.Addr(),
	})

	return client, mr
}
