package validator

import (
	"fmt"
	"reflect"
	"strings"
)

type ValidationErrorItem struct {
	Field   string `json:"field"`
	Tag     string `json:"tag"`
	Message string `json:"message"`
	Args    any    `json:"args,omitempty"`
}

type CustomValidationError struct {
	Errors []ValidationErrorItem `json:"errors"`
}

func (e *CustomValidationError) Error() string {
	if len(e.Errors) == 0 {
		return "validation error"
	}
	messages := make([]string, len(e.Errors))
	for i, err := range e.Errors {
		messages[i] = fmt.Sprintf("%s: %s", err.Field, err.Message)
	}
	return strings.Join(messages, "; ")
}

func NewCustomValidationError() *CustomValidationError {
	return &CustomValidationError{
		Errors: make([]ValidationErrorItem, 0),
	}
}

func (e *CustomValidationError) AddError(field, tag, message string, args ...any) {
	vei := ValidationErrorItem{
		Field:   field,
		Tag:     tag,
		Message: message,
	}
	if len(args) > 0 {
		vei.Args = args[0]
	}

	e.Errors = append(e.Errors, vei)
}

func (e *CustomValidationError) HasErrors() bool {
	return len(e.Errors) > 0
}

// Validation tag names usable with CustomValidationError.AddField.
const (
	ValidationTagNotFound  = "not_found"
	ValidationTagDuplicate = "duplicate"
	ValidationTagInvalid   = "invalid"
	ValidationTagRequired  = "required"
	ValidationTagMaxLength = "max_length"
	ValidationTagMinLength = "min_length"
	ValidationTagFormat    = "format"
	ValidationTagUnique    = "unique"
	ValidationTagMismatch  = "mismatch"
)

var errorTemplates = map[string]string{
	ValidationTagNotFound:  "%s not found",
	ValidationTagDuplicate: "%s is already exists",
	ValidationTagInvalid:   "%s is invalid",
	ValidationTagRequired:  "%s is required",
	ValidationTagMaxLength: "%s exceeds maximum length",
	ValidationTagMinLength: "%s is below minimum length",
	ValidationTagFormat:    "%s format is invalid",
	ValidationTagUnique:    "%s must be unique",
	ValidationTagMismatch:  "%s does not match",
}

func GetErrorTemplate(tag string) (string, bool) {
	template, ok := errorTemplates[tag]
	return template, ok
}

func getJSONFieldName(structType reflect.Type, fieldName string) string {
	field, found := structType.FieldByName(fieldName)
	if !found {
		return strings.ToLower(fieldName)
	}

	jsonTag := field.Tag.Get("json")
	if jsonTag == "" || jsonTag == "-" {
		return strings.ToLower(fieldName)
	}

	jsonName := strings.Split(jsonTag, ",")[0]
	if jsonName == "" {
		return strings.ToLower(fieldName)
	}

	return jsonName
}

func getFieldLabel(structType reflect.Type, fieldName string) string {
	field, found := structType.FieldByName(fieldName)
	if !found {
		return fieldName
	}

	labelTag := field.Tag.Get("label")
	if labelTag != "" {
		return labelTag
	}

	jsonTag := field.Tag.Get("json")
	if jsonTag != "" && jsonTag != "-" {
		jsonName := strings.Split(jsonTag, ",")[0]
		if jsonName != "" {
			parts := strings.Split(jsonName, "_")
			for i, part := range parts {
				if len(part) > 0 {
					parts[i] = strings.ToUpper(part[:1]) + strings.ToLower(part[1:])
				}
			}
			return strings.Join(parts, " ")
		}
	}

	return fieldName
}

func (e *CustomValidationError) AddField(structPtr any, fieldName, tag string, args ...any) error {
	structType := reflect.Indirect(reflect.ValueOf(structPtr)).Type()

	if structType.Kind() != reflect.Struct {
		return fmt.Errorf("structPtr must be a pointer to struct or struct")
	}

	_, found := structType.FieldByName(fieldName)
	if !found {
		return fmt.Errorf("field %s not found in struct", fieldName)
	}

	jsonFieldName := getJSONFieldName(structType, fieldName)
	label := getFieldLabel(structType, fieldName)

	template, ok := GetErrorTemplate(tag)
	if !ok {
		return fmt.Errorf("error template for tag %s not found", tag)
	}

	message := fmt.Sprintf(template, label)

	e.AddError(jsonFieldName, tag, message, args...)
	return nil
}
