package mongo_test

import (
	"context"
	"fmt"

	"github.com/hallode/golib/v2/storage/mongo"
)

func ExampleNew() {
	m, err := mongo.New(&mongo.Options{
		URI:                 "mongodb://localhost:27017",
		DB:                  "app",
		AppName:             "orders",
		EnableOpenTelemetry: true, // requires a global OTel provider (golib/otel)
	})
	if err != nil {
		fmt.Println(err)
		return
	}
	defer m.Close(context.Background())

	// Use DB for collection operations; Client() for multi-document transactions.
	coll := m.DB.Collection("orders")
	_ = coll
}
