package circuitbreaker_test

import (
	"context"
	"errors"
	"fmt"

	"github.com/hallode/golib/circuitbreaker"
)

func ExampleExecute() {
	cb := circuitbreaker.NewDefaultCircuitBreaker("orders")

	// A successful call returns its typed result.
	total, err := circuitbreaker.Execute(cb, context.Background(), func() (int, error) {
		return 42, nil
	})
	fmt.Println(total, err)

	// Business/validation errors are wrapped so they do not trip the breaker.
	_, err = circuitbreaker.Execute(cb, context.Background(), func() (int, error) {
		return 0, circuitbreaker.NewNonTransientError(errors.New("invalid input"))
	})
	fmt.Println(err)
	// Output:
	// 42 <nil>
	// invalid input
}
