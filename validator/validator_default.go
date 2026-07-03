package validator

import (
	"github.com/go-playground/validator/v10"
	"reflect"
	"strings"
)

func registerLabelName(labelName string) validator.TagNameFunc {
	return func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get(labelName), ",", 2)[0]

		if name == "-" {
			return ""
		}

		if name == "" {
			return fld.Name
		}

		return name
	}
}
