package sql_test

import (
	"context"
	"fmt"

	"github.com/hallode/golib/storage/sql"
)

func ExampleNew() {
	db, err := sql.New(&sql.Options{
		DSN:      "postgres://user:pass@localhost:5432/app?sslmode=disable",
		MaxConns: 10,
	})
	if err != nil {
		fmt.Println(err)
		return
	}
	defer db.Close()

	// All query access goes through the exposed pgx pool.
	var count int
	_ = db.Pool.QueryRow(context.Background(), "SELECT count(*) FROM users").Scan(&count)
	fmt.Println(count)
}
