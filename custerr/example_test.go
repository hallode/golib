package custerr_test

import (
	"fmt"

	"github.com/hallode/golib/custerr"
)

func ExampleNotFoundf() {
	err := custerr.NotFoundf("order %s not found", "A123")

	fmt.Println("message:", err.Error())
	fmt.Println("status:", err.StatusCode)
	fmt.Println("business code:", custerr.BusinessCodeByStatus(err.StatusCode))
	// Output:
	// message: order A123 not found
	// status: 404
	// business code: 2000002
}
