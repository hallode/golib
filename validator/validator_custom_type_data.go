package validator

import (
	"github.com/go-playground/validator/v10"
)

type CustomType struct {
	CustomFn validator.CustomTypeFunc
	TypeData interface{}
}

type customTypes struct {
	types map[string]*CustomType
}

func (c *customTypes) AddTypeData(name string, customType *CustomType) {
	c.types[name] = customType
}

func newCustomTypes() *customTypes {
	return &customTypes{types: make(map[string]*CustomType)}
}
