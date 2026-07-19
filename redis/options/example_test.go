package options_test

import (
	"fmt"
	"time"

	"github.com/hallode/golib/redis/options"
)

// Options are normally passed to golib/redis methods (Get, Set, …). Here one is
// built directly to show the resolved values.
func ExampleOptions() {
	o := options.Default()
	options.WithTTL(90 * time.Second)(o)
	options.WithMsgPack()(o)

	fmt.Println("ttl:", o.GetExpire())
	fmt.Println("msgpack:", o.GetSerialize() == options.SerializeMessagePack)
	// Output:
	// ttl: 1m30s
	// msgpack: true
}
