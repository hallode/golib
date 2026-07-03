package json

import (
	stdjson "encoding/json"
	"sync/atomic"

	"github.com/bytedance/sonic"
)

type codecFuncs struct {
	marshal   func(any) ([]byte, error)
	unmarshal func([]byte, any) error
}

var _codec atomic.Pointer[codecFuncs]

func init() {
	_codec.Store(&codecFuncs{
		marshal:   stdjson.Marshal,
		unmarshal: stdjson.Unmarshal,
	})
}

// Init sets the JSON codec. Call once at startup before serving requests.
// kind: "sonic" | "default" (anything else falls back to default)
//
// When "sonic" is selected, each call falls back to the standard library if
// sonic returns an error or panics (e.g. unsupported type, function value).
// Errors returned are always from the standard library; the original sonic
// error is discarded.
func Init(kind string) {
	var c *codecFuncs
	switch kind {
	case "sonic":
		// ConfigStd preserves encoding/json semantics: HTML escaping, map-key
		// sorting, and string copy on unmarshal.
		cfg := sonic.ConfigStd
		sonicMarshal := cfg.Marshal
		sonicUnmarshal := cfg.Unmarshal
		c = &codecFuncs{
			marshal: func(v any) (b []byte, err error) {
				usedFallback := false
				defer func() {
					if recover() != nil && !usedFallback {
						b, err = stdjson.Marshal(v)
					}
				}()
				if b, err = sonicMarshal(v); err != nil {
					usedFallback = true
					b, err = stdjson.Marshal(v)
				}
				return
			},
			unmarshal: func(data []byte, v any) (err error) {
				usedFallback := false
				defer func() {
					if recover() != nil && !usedFallback {
						err = stdjson.Unmarshal(data, v)
					}
				}()
				if err = sonicUnmarshal(data, v); err != nil {
					usedFallback = true
					err = stdjson.Unmarshal(data, v)
				}
				return
			},
		}
	default:
		c = &codecFuncs{
			marshal:   stdjson.Marshal,
			unmarshal: stdjson.Unmarshal,
		}
	}
	_codec.Store(c)
}
