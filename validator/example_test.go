package validator_test

import (
	"fmt"

	"github.com/hallode/golib/validator"
)

// Build a structured validation error by hand — useful for business-rule
// failures that struct tags cannot express (e.g. "email already taken").
func ExampleCustomValidationError() {
	type CreateUser struct {
		Email string `json:"email" label:"Email"`
		Name  string `json:"name"`
	}

	verr := validator.NewCustomValidationError()
	_ = verr.AddField(CreateUser{}, "Email", validator.ValidationTagFormat)
	_ = verr.AddField(CreateUser{}, "Name", validator.ValidationTagRequired)

	fmt.Println(verr.Error())
	// Output: email: Email format is invalid; name: Name is required
}

// ExampleValidate runs the default instance over struct tags; it returns a
// *CustomValidationError describing every failed field.
func ExampleValidate() {
	type SignupForm struct {
		Email string  `json:"email" validate:"required,email"`
		Price float64 `json:"price" validate:"numeric_precision=5.2"`
	}

	if err := validator.Validate(SignupForm{Email: "not-an-email", Price: 12345.67}); err != nil {
		fmt.Println(err)
	}
}
