package redis_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	goredis "github.com/hallode/golib/v2/redis"
	"github.com/hallode/golib/v2/redis/options"
)

type cacheItem struct {
	ID   int64  `json:"id" msgpack:"id"`
	Name string `json:"name" msgpack:"name"`
}

func setupRedis(t *testing.T, mutate ...func(*goredis.RedisOption)) *goredis.Redis {
	t.Helper()

	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis.Run() error = %v", err)
	}
	t.Cleanup(mr.Close)

	opt := goredis.RedisOption{
		Host: mr.Addr(),
		Exp:  3600,
	}
	for _, fn := range mutate {
		fn(&opt)
	}

	client, err := goredis.NewRedis(opt)
	if err != nil {
		t.Fatalf("NewRedis() error = %v", err)
	}
	t.Cleanup(func() { _ = client.Close() })

	return client
}

func TestNewRedis_WithOpenTelemetry(t *testing.T) {
	r := setupRedis(t, func(o *goredis.RedisOption) {
		o.EnableOpenTelemetry = true
	})
	if r == nil {
		t.Fatal("expected client")
	}
}

func TestNewRedis_ConnectionFailure(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis.Run() error = %v", err)
	}
	addr := mr.Addr()
	mr.Close()

	_, err = goredis.NewRedis(goredis.RedisOption{Host: addr})
	if err == nil {
		t.Fatal("expected connection error")
	}
}

func TestSetGet_StructJSON(t *testing.T) {
	ctx := context.Background()
	r := setupRedis(t)

	want := cacheItem{ID: 1, Name: "alpha"}
	if err := r.Set(ctx, "item:1", want); err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	var got cacheItem
	if err := r.Get(ctx, "item:1", &got); err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got != want {
		t.Fatalf("Get() = %+v, want %+v", got, want)
	}
}

func TestSetGet_MsgPack(t *testing.T) {
	ctx := context.Background()
	r := setupRedis(t, func(o *goredis.RedisOption) {
		o.UseMsgPack = true
	})

	want := cacheItem{ID: 2, Name: "beta"}
	if err := r.Set(ctx, "item:2", want); err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	var got cacheItem
	if err := r.Get(ctx, "item:2", &got); err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got != want {
		t.Fatalf("Get() = %+v, want %+v", got, want)
	}
}

func TestGet_NotFound(t *testing.T) {
	ctx := context.Background()
	r := setupRedis(t)

	var got cacheItem
	err := r.Get(ctx, "missing", &got)
	if !errors.Is(err, goredis.ErrNotFound) {
		t.Fatalf("Get() error = %v, want ErrNotFound", err)
	}
}

func TestGetStringAndInt(t *testing.T) {
	ctx := context.Background()
	r := setupRedis(t)

	if err := r.Set(ctx, "counter", int64(7)); err != nil {
		t.Fatalf("Set() error = %v", err)
	}
	if err := r.Set(ctx, "label", "hello"); err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	n, err := r.GetInt(ctx, "counter")
	if err != nil || n != 7 {
		t.Fatalf("GetInt() = (%d, %v)", n, err)
	}

	s, err := r.GetString(ctx, "label")
	if err != nil || s != "hello" {
		t.Fatalf("GetString() = (%q, %v)", s, err)
	}
}

func TestMGet(t *testing.T) {
	ctx := context.Background()
	r := setupRedis(t)

	_ = r.Set(ctx, "item:a", cacheItem{ID: 1, Name: "a"})
	_ = r.Set(ctx, "item:b", cacheItem{ID: 2, Name: "b"})

	var items []cacheItem
	if err := r.MGet(ctx, []string{"item:a", "item:b", "item:missing"}, &items); err != nil {
		t.Fatalf("MGet() error = %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("MGet() len = %d, want 2", len(items))
	}
}

func TestMGet_EmptyKeys(t *testing.T) {
	ctx := context.Background()
	r := setupRedis(t)

	var items []cacheItem
	if err := r.MGet(ctx, nil, &items); err != nil {
		t.Fatalf("MGet() error = %v", err)
	}
}

func TestExistsDel(t *testing.T) {
	ctx := context.Background()
	r := setupRedis(t)

	if err := r.Set(ctx, "temp", "value"); err != nil {
		t.Fatalf("Set() error = %v", err)
	}
	if !r.Exists(ctx, "temp") {
		t.Fatal("Exists() = false, want true")
	}
	if err := r.Del(ctx, "temp"); err != nil {
		t.Fatalf("Del() error = %v", err)
	}
	if r.Exists(ctx, "temp") {
		t.Fatal("Exists() = true after Del")
	}
}

func TestIncrement(t *testing.T) {
	ctx := context.Background()
	r := setupRedis(t)

	n, err := r.Increment(ctx, "hits")
	if err != nil || n != 1 {
		t.Fatalf("Increment() first = (%d, %v)", n, err)
	}
	n, err = r.Increment(ctx, "hits", options.WithTTL(5*time.Minute))
	if err != nil || n != 2 {
		t.Fatalf("Increment() second = (%d, %v)", n, err)
	}
}

func TestSetMultiple(t *testing.T) {
	ctx := context.Background()
	r := setupRedis(t)

	err := r.SetMultiple(ctx, []*goredis.DataSet{
		{Key: "batch:1", Value: "one"},
		{Key: "batch:2", Value: cacheItem{ID: 9, Name: "nine"}},
	})
	if err != nil {
		t.Fatalf("SetMultiple() error = %v", err)
	}

	var item cacheItem
	if err := r.Get(ctx, "batch:2", &item); err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if item.ID != 9 {
		t.Fatalf("Get() = %+v", item)
	}
}

func TestHSetHGetHGetAll(t *testing.T) {
	ctx := context.Background()
	r := setupRedis(t)

	err := r.HSet(ctx, "profile:1", []*goredis.DataSet{
		{Key: "meta", Value: cacheItem{ID: 1, Name: "alice"}},
		{Key: "extra", Value: cacheItem{ID: 2, Name: "bob"}},
	})
	if err != nil {
		t.Fatalf("HSet() error = %v", err)
	}

	var one cacheItem
	if err := r.HGet(ctx, "profile:1", "meta", &one); err != nil {
		t.Fatalf("HGet() error = %v", err)
	}
	if one.Name != "alice" {
		t.Fatalf("HGet() = %+v", one)
	}

	var all []cacheItem
	if err := r.HGetAll(ctx, "profile:1", &all); err != nil {
		t.Fatalf("HGetAll() error = %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("HGetAll() len = %d, want 2", len(all))
	}
}

func TestHMSetAndHMGetPipelined(t *testing.T) {
	ctx := context.Background()
	r := setupRedis(t)

	err := r.HMSetPipelined(ctx, map[string][]*goredis.DataSet{
		"hash:a": {
			{Key: "field1", Value: cacheItem{ID: 1, Name: "a"}},
		},
		"hash:b": {
			{Key: "field2", Value: cacheItem{ID: 2, Name: "b"}},
		},
	})
	if err != nil {
		t.Fatalf("HMSetPipelined() error = %v", err)
	}

	raw, err := r.HMGetPipelined(ctx, map[string]string{
		"hash:a":  "field1",
		"hash:b":  "field2",
		"missing": "field",
	})
	if err != nil {
		t.Fatalf("HMGetPipelined() error = %v", err)
	}
	if len(raw) != 2 {
		t.Fatalf("HMGetPipelined() len = %d, want 2", len(raw))
	}

	var item cacheItem
	if err := r.UnmarshalValue(raw["hash:a"], &item); err != nil {
		t.Fatalf("UnmarshalValue() error = %v", err)
	}
	if item.Name != "a" {
		t.Fatalf("UnmarshalValue() = %+v", item)
	}
}

func TestSetMemberGetMember(t *testing.T) {
	ctx := context.Background()
	r := setupRedis(t)

	if err := r.Set(ctx, "member:1", cacheItem{ID: 1, Name: "one"}); err != nil {
		t.Fatalf("Set() error = %v", err)
	}
	if err := r.Set(ctx, "member:2", cacheItem{ID: 2, Name: "two"}); err != nil {
		t.Fatalf("Set() error = %v", err)
	}
	if err := r.SetMember(ctx, "members", []*goredis.DataSet{
		{Key: "member:1"},
		{Key: "member:2"},
	}); err != nil {
		t.Fatalf("SetMember() error = %v", err)
	}

	var items []cacheItem
	count, err := r.GetMember(ctx, "members", &items)
	if err != nil {
		t.Fatalf("GetMember() error = %v", err)
	}
	if count != 2 || len(items) != 2 {
		t.Fatalf("GetMember() = (%d, %d items)", count, len(items))
	}
}

func TestDeleteWithPattern(t *testing.T) {
	ctx := context.Background()
	r := setupRedis(t)

	_ = r.Set(ctx, "cache:1", "a")
	_ = r.Set(ctx, "cache:2", "b")
	_ = r.Set(ctx, "other:1", "c")

	if err := r.DeleteWithPattern(ctx, "cache:*"); err != nil {
		t.Fatalf("DeleteWithPattern() error = %v", err)
	}
	if r.Exists(ctx, "cache:1", "cache:2") {
		t.Fatal("pattern delete should remove cache:* keys")
	}
	if !r.Exists(ctx, "other:1") {
		t.Fatal("pattern delete should not remove unrelated keys")
	}
}

func TestPerCallSerializationOverride(t *testing.T) {
	ctx := context.Background()
	r := setupRedis(t)

	want := cacheItem{ID: 3, Name: "msgpack"}
	if err := r.Set(ctx, "override", want, options.WithMsgPack()); err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	var got cacheItem
	if err := r.Get(ctx, "override", &got, options.WithMsgPack()); err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got != want {
		t.Fatalf("Get() = %+v, want %+v", got, want)
	}
}
