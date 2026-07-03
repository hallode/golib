package validator

import (
	"context"
	"errors"
	"math"
	"reflect"
	"strconv"
	"strings"
	"unicode"

	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
)

// validate is the default instance used by Validate/ValidateCtx. It ships
// with one sample custom rule (numeric_precision) — services register their
// own domain rules by building an instance with New().AddValidator(...).Load().
var validate *CustomValidator

func init() {
	validate = New().
		SetTranslateError(&customTranslate{}).
		// Sample custom validator: numeric_precision=P.S validates that a
		// numeric field fits DECIMAL(P,S). Copy this pattern for your own rules.
		AddValidator("numeric_precision", validateNumericPrecision, map[CountryCode]*TranslateDetail{
			CCID: NewTranslateDetail("{0} melewati batas presisi numerik"),
			CCEN: NewTranslateDetail("{0} exceeds numeric precision"),
		}).
		Load()
}

// Validate validates a struct using the default instance.
func Validate(dest any, cc ...CountryCode) error {
	return validate.Validation(dest, cc...)
}

// ValidateCtx validates a struct with context using the default instance.
func ValidateCtx(ctx context.Context, dest any, cc ...CountryCode) error {
	return validate.ValidationCtx(ctx, dest, cc...)
}

// FiberValidator implements fiber.StructValidator, delegating to Validate.
// Plug it into fiber.Config.StructValidator so c.Bind().Body() auto-validates.
type FiberValidator struct{}

func (v *FiberValidator) Validate(out any) error {
	return Validate(out)
}

// customTranslate converts go-playground validation errors into
// CustomValidationError with JSON field names.
type customTranslate struct{}

func (d *customTranslate) TranslateError(errValidate error, translator ut.Translator) error {
	var validatorErrs validator.ValidationErrors
	errors.As(errValidate, &validatorErrs)

	customErr := NewCustomValidationError()

	for _, e := range validatorErrs {
		fieldName := d.fieldName(e)
		customErr.AddError(fieldName, e.Tag(), e.Translate(translator))
	}

	return customErr
}

func (d *customTranslate) fieldName(e validator.FieldError) string {
	structFieldName := e.StructField()
	if structFieldName == "" {
		return toSnakeCase(e.Field())
	}

	fieldValue := e.Value()
	if fieldValue == nil {
		return toSnakeCase(structFieldName)
	}

	fieldVal := reflect.ValueOf(fieldValue)
	for fieldVal.Kind() == reflect.Ptr {
		fieldVal = fieldVal.Elem()
	}

	if fieldVal.IsValid() && fieldVal.Kind() == reflect.Struct {
		if fieldInfo, found := fieldVal.Type().FieldByName(structFieldName); found {
			jsonTag := fieldInfo.Tag.Get("json")
			if jsonTag != "" && jsonTag != "-" {
				if jsonName := strings.Split(jsonTag, ",")[0]; jsonName != "" {
					return jsonName
				}
			}
		}
	}

	return toSnakeCase(structFieldName)
}

func toSnakeCase(s string) string {
	if s == "" {
		return s
	}

	var result strings.Builder
	result.Grow(len(s) + len(s)/3)

	runes := []rune(s)
	for i, r := range runes {
		if unicode.IsUpper(r) {
			if i > 0 {
				prevIsLower := unicode.IsLower(runes[i-1])
				prevIsUpper := unicode.IsUpper(runes[i-1])
				nextIsLower := i+1 < len(runes) && unicode.IsLower(runes[i+1])

				if prevIsLower || (prevIsUpper && nextIsLower) {
					result.WriteByte('_')
				}
			}
			result.WriteRune(unicode.ToLower(r))
		} else {
			result.WriteRune(r)
		}
	}

	return result.String()
}

// validateNumericPrecision is the sample custom rule shipped with the default
// instance: `validate:"numeric_precision=5.2"` checks the value fits
// DECIMAL(5,2). Nil pointers pass (combine with required if needed).
func validateNumericPrecision(fl validator.FieldLevel) bool {
	precision, scale, ok := parseNumericPrecisionParam(fl.Param())
	if !ok {
		return false
	}

	value, exists := numericFieldValue(fl.Field())
	if !exists {
		return true
	}
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return false
	}

	abs := math.Abs(value)
	maxAbs := math.Pow10(precision-scale) - math.Pow10(-scale)
	return abs <= maxAbs+floatTolerance(scale)
}

func parseNumericPrecisionParam(param string) (int, int, bool) {
	precisionStr, scaleStr, ok := strings.Cut(param, ".")
	if !ok {
		return 0, 0, false
	}
	precision, err := strconv.Atoi(precisionStr)
	if err != nil {
		return 0, 0, false
	}
	scale, err := strconv.Atoi(scaleStr)
	if err != nil {
		return 0, 0, false
	}
	if precision <= 0 || scale < 0 || scale > precision {
		return 0, 0, false
	}
	return precision, scale, true
}

func numericFieldValue(field reflect.Value) (float64, bool) {
	for field.Kind() == reflect.Ptr {
		if field.IsNil() {
			return 0, false
		}
		field = field.Elem()
	}

	switch field.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return float64(field.Int()), true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return float64(field.Uint()), true
	case reflect.Float32, reflect.Float64:
		return field.Float(), true
	default:
		return 0, false
	}
}

func floatTolerance(scale int) float64 {
	return math.Pow10(-(scale + 6))
}
