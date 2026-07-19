package redis

import (
	"context"
	"errors"
	"reflect"
	"time"

	"github.com/hallode/golib/v2/redis/options"

	"github.com/redis/go-redis/extra/redisotel/v9"
	"github.com/redis/go-redis/v9"
)

var ErrNotFound = redis.Nil

type (
	RedisOption struct {
		Host       string
		MasterName string
		Password   string
		DB         int
		Exp        int
		PoolSize   int
		MinIdleCon int
		UseMsgPack bool // Use MessagePack as default serialization (recommended for high-traffic)

		// EnableOpenTelemetry registers redisotel tracing and metrics hooks on the client.
		// Default is false; set true when the service exports OpenTelemetry data.
		EnableOpenTelemetry bool
	}

	Redis struct {
		*redis.Client
		options *options.Options
	}

	DataSet struct {
		Key   string
		Value any
	}
)

func NewRedisSentinel(opt RedisOption) (*Redis, error) {
	client := redis.NewFailoverClient(&redis.FailoverOptions{
		SentinelAddrs: []string{opt.Host},
		MasterName:    opt.MasterName,
		Password:      opt.Password,
		DB:            opt.DB,
		PoolSize:      opt.PoolSize,
		MinIdleConns:  opt.MinIdleCon,
	})

	return newRedis(client, opt)
}

func NewRedis(opt RedisOption) (*Redis, error) {
	client := redis.NewClient(&redis.Options{
		Addr:         opt.Host,
		Password:     opt.Password,
		DB:           opt.DB,
		PoolSize:     opt.PoolSize,
		MinIdleConns: opt.MinIdleCon,
	})

	return newRedis(client, opt)
}

func newRedis(client *redis.Client, opt RedisOption) (*Redis, error) {
	err := client.Ping(context.Background()).Err()
	if err != nil {
		return nil, err
	}

	if opt.EnableOpenTelemetry {
		if err = redisotel.InstrumentTracing(client); err != nil {
			return nil, err
		}
		if err = redisotel.InstrumentMetrics(client); err != nil {
			return nil, err
		}
	}

	defaultOpts := options.Default()
	if opt.Exp > 0 {
		options.WithTTL(time.Duration(opt.Exp) * time.Second)(defaultOpts)
	}
	if opt.UseMsgPack {
		options.WithMsgPack()(defaultOpts)
	}

	return &Redis{
		Client:  client,
		options: defaultOpts,
	}, nil
}

func (r *Redis) applyOptions(opts ...options.Option) options.Options {
	opt := *r.options
	for _, fn := range opts {
		if fn != nil {
			fn(&opt)
		}
	}
	return opt
}

func (r *Redis) Get(ctx context.Context, key string, value any, opts ...options.Option) error {
	data, err := r.Client.Get(ctx, key).Bytes()
	if err != nil {
		return err
	}

	opt := r.applyOptions(opts...)

	return opt.GetSerialize().Unmarshal(data, value)
}

func (r *Redis) GetInt(ctx context.Context, key string) (int64, error) {
	return r.Client.Get(ctx, key).Int64()
}

func (r *Redis) GetString(ctx context.Context, key string) (string, error) {
	return r.Client.Get(ctx, key).Result()
}

func (r *Redis) MGet(ctx context.Context, keys []string, obj any, opts ...options.Option) error {
	if len(keys) == 0 {
		return nil
	}

	data, err := r.Client.MGet(ctx, keys...).Result()
	if err != nil {
		return err
	}

	return r.readSliceInterfaceToObj(data, obj, opts...)
}

func (r *Redis) readSliceInterfaceToObj(val []any, obj any, opts ...options.Option) error {
	opt := r.applyOptions(opts...)

	rv := reflect.ValueOf(obj)

	if rv.Kind() == reflect.Ptr {
		rv = rv.Elem()
	}

	typ := rv.Type().Elem()
	elemIsPtr := typ.Kind() == reflect.Ptr

	for _, v := range val {
		if v == nil {
			continue
		}
		var elem reflect.Value
		if elemIsPtr {
			elem = reflect.New(typ.Elem())
		} else {
			elem = reflect.New(typ)
		}
		err := opt.GetSerialize().Unmarshal([]byte(v.(string)), elem.Interface())
		if err != nil {
			return err
		}
		if elemIsPtr {
			rv.Set(reflect.Append(rv, elem))
		} else {
			rv.Set(reflect.Append(rv, elem.Elem()))
		}
	}

	return nil
}

func (r *Redis) GetMember(ctx context.Context, key string, obj any, opts ...options.Option) (int, error) {
	keys, err := r.Client.SMembers(ctx, key).Result()
	if err != nil {
		return 0, err
	}

	totalMembers := len(keys)
	if totalMembers == 0 {
		return 0, errors.New("member not found")
	}

	return totalMembers, r.MGet(ctx, keys, obj, opts...)
}

func (r *Redis) Set(ctx context.Context, key string, value any, opts ...options.Option) error {
	opt := r.applyOptions(opts...)

	switch value.(type) {
	case string, bool, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64, []byte:
		return r.Client.Set(ctx, key, value, opt.GetExpire()).Err()
	default:
		data, err := opt.GetSerialize().Marshal(value)
		if err != nil {
			return err
		}

		return r.Client.Set(ctx, key, data, opt.GetExpire()).Err()
	}
}

func (r *Redis) SetMember(ctx context.Context, key string, data []*DataSet, opts ...options.Option) error {
	opt := r.applyOptions(opts...)

	members := make([]string, 0)
	_, err := r.Client.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
		for _, v := range data {
			members = append(members, v.Key)
		}
		pipe.SAdd(ctx, key, members)
		pipe.Expire(ctx, key, options.ApplyJitter(opt.GetExpire()))
		return nil
	})
	if err != nil {
		return err
	}

	return nil
}

func (r *Redis) DeleteWithPattern(ctx context.Context, pattern string) error {
	iter := r.Client.Scan(ctx, 0, pattern, 0).Iterator()
	var localKeys []string

	for iter.Next(ctx) {
		localKeys = append(localKeys, iter.Val())
	}

	if err := iter.Err(); err != nil {
		return err
	}

	if len(localKeys) > 0 {
		_, err := r.Client.Pipelined(ctx, func(pipeline redis.Pipeliner) error {
			pipeline.Del(ctx, localKeys...)
			return nil
		})

		if err != nil {
			return err
		}
	}

	return nil
}

// SetMultiple applies per-item TTL jitter to reduce cache stampede on batch writes.
func (r *Redis) SetMultiple(ctx context.Context, data []*DataSet, opts ...options.Option) error {
	if len(data) == 0 {
		return nil
	}

	opt := r.applyOptions(opts...)

	_, err := r.Client.Pipelined(ctx, func(pipe redis.Pipeliner) error {
		for _, v := range data {
			itemExpire := options.ApplyJitter(opt.GetExpire())

			switch v.Value.(type) {
			case string, bool, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64, []byte:
				pipe.Set(ctx, v.Key, v.Value, itemExpire)
			default:
				vb, err := opt.GetSerialize().Marshal(v.Value)
				if err != nil {
					return err
				}
				pipe.Set(ctx, v.Key, vb, itemExpire)
			}
		}
		return nil
	})
	if err != nil {
		return err
	}

	return nil
}

func (r *Redis) HGetAll(ctx context.Context, key string, obj any, opts ...options.Option) error {
	data, err := r.Client.HGetAll(ctx, key).Result()
	if err != nil {
		return err
	}

	if len(data) == 0 {
		return errors.New("no data")
	}

	allData := make([]any, 0, len(data))
	for _, v := range data {
		allData = append(allData, v)
	}

	return r.readSliceInterfaceToObj(allData, obj, opts...)
}

func (r *Redis) HMGetPipelined(ctx context.Context, requests map[string]string, _ ...options.Option) (map[string][]byte, error) {
	if len(requests) == 0 {
		return map[string][]byte{}, nil
	}

	type entry struct {
		key string
		cmd *redis.StringCmd
	}
	entries := make([]entry, 0, len(requests))

	_, _ = r.Client.Pipelined(ctx, func(pipe redis.Pipeliner) error {
		for key, field := range requests {
			cmd := pipe.HGet(ctx, key, field)
			entries = append(entries, entry{key: key, cmd: cmd})
		}
		return nil
	})

	result := make(map[string][]byte, len(entries))
	for _, e := range entries {
		data, err := e.cmd.Bytes()
		if err != nil {
			continue
		}
		result[e.key] = data
	}
	return result, nil
}

func (r *Redis) HMSetPipelined(ctx context.Context, writes map[string][]*DataSet, opts ...options.Option) error {
	if len(writes) == 0 {
		return nil
	}

	opt := r.applyOptions(opts...)

	_, err := r.Client.Pipelined(ctx, func(pipe redis.Pipeliner) error {
		for key, datasets := range writes {
			if len(datasets) == 0 {
				continue
			}
			hashData := make(map[string]any, len(datasets))
			for _, v := range datasets {
				vb, err := opt.GetSerialize().Marshal(v.Value)
				if err != nil {
					return err
				}
				hashData[v.Key] = vb
			}
			pipe.HSet(ctx, key, hashData)
			pipe.Expire(ctx, key, options.ApplyJitter(opt.GetExpire()))
		}
		return nil
	})
	return err
}

func (r *Redis) UnmarshalValue(raw []byte, dest any, opts ...options.Option) error {
	opt := r.applyOptions(opts...)
	return opt.GetSerialize().Unmarshal(raw, dest)
}

func (r *Redis) HGet(ctx context.Context, key string, field string, dest any, opts ...options.Option) error {
	opt := r.applyOptions(opts...)

	data, err := r.Client.HGet(ctx, key, field).Bytes()
	if err != nil {
		return err
	}

	return opt.GetSerialize().Unmarshal(data, dest)
}

func (r *Redis) HSet(ctx context.Context, key string, data []*DataSet, opts ...options.Option) error {
	if len(data) == 0 {
		return nil
	}

	opt := r.applyOptions(opts...)

	hashData := make(map[string]any, len(data))
	for _, v := range data {
		vb, err := opt.GetSerialize().Marshal(v.Value)
		if err != nil {
			return err
		}
		hashData[v.Key] = vb
	}

	_, err := r.Client.Pipelined(ctx, func(pipe redis.Pipeliner) error {
		pipe.HSet(ctx, key, hashData)
		pipe.Expire(ctx, key, options.ApplyJitter(opt.GetExpire()))
		return nil
	})

	return err
}

func (r *Redis) Del(ctx context.Context, keys ...string) error {
	if len(keys) == 0 {
		return nil
	}
	return r.Client.Del(ctx, keys...).Err()
}

func (r *Redis) Exists(ctx context.Context, key ...string) bool {
	return r.Client.Exists(ctx, key...).Val() > 0
}

func (r *Redis) Increment(ctx context.Context, key string, opts ...options.Option) (int64, error) {
	opt := r.applyOptions(opts...)
	pipe := r.Client.Pipeline()
	incr := pipe.Incr(ctx, key)
	pipe.Expire(ctx, key, opt.GetExpire())
	_, err := pipe.Exec(ctx)
	if err != nil {
		return 0, err
	}
	return incr.Val(), nil
}
