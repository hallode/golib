package redis_test

import (
	"context"
	"fmt"

	"github.com/hallode/golib/v2/redis"
	"github.com/hallode/golib/v2/redis/options"
)

func ExampleNewRedis() {
	rdb, err := redis.NewRedis(redis.RedisOption{
		Host:       "localhost:6379",
		Exp:        60, // default TTL in SECONDS
		UseMsgPack: true,
	})
	if err != nil {
		fmt.Println(err)
		return
	}

	ctx := context.Background()
	type user struct{ Name string }

	// The serialization format must match on write and read.
	_ = rdb.Set(ctx, "user:1", user{Name: "Ada"})

	var u user
	_ = rdb.Get(ctx, "user:1", &u, options.WithMsgPack())
	fmt.Println(u.Name)
}
