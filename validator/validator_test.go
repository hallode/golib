package validator

import (
	"math"
	"testing"
)

func TestValidateNumericPrecision(t *testing.T) {
	type payload struct {
		Value *float64 `validate:"omitempty,numeric_precision=5.2"`
	}

	ptr := func(v float64) *float64 { return &v }

	tests := []struct {
		name    string
		value   *float64
		wantErr bool
	}{
		{name: "nil pointer", value: nil},
		{name: "zero", value: ptr(0)},
		{name: "positive boundary", value: ptr(999.99)},
		{name: "negative boundary", value: ptr(-999.99)},
		{name: "integer overflow", value: ptr(1000), wantErr: true},
		{name: "extra fraction", value: ptr(1.234)},
		{name: "nan", value: ptr(math.NaN()), wantErr: true},
		{name: "inf", value: ptr(math.Inf(1)), wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Validate(payload{Value: tt.value})
			if (err != nil) != tt.wantErr {
				t.Fatalf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateNumericPrecisionWithRequiredField(t *testing.T) {
	type payload struct {
		Value float64 `validate:"required,numeric_precision=6.2"`
	}

	if err := Validate(payload{Value: 9999.99}); err != nil {
		t.Fatalf("Validate() valid boundary error = %v", err)
	}

	if err := Validate(payload{Value: 10000}); err == nil {
		t.Fatal("Validate() expected overflow error")
	}
}
