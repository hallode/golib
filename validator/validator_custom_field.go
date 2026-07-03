package validator

import "github.com/go-playground/validator/v10"

// tagValidator is the tag name of a custom validation rule.
type tagValidator string

func (t tagValidator) String() string {
	return string(t)
}

type validatorField struct {
	Tag             tagValidator
	Translator      map[CountryCode]*TranslateDetail
	LogicValidation validator.Func
}

type customFields struct {
	fields map[tagValidator]*validatorField
}

func newCustomFields() *customFields {
	return &customFields{fields: make(map[tagValidator]*validatorField)}
}

// add registers a rule's logic and translations in one step, so a field can
// never exist in a half-initialized state.
func (c *customFields) add(tag tagValidator, logicFn validator.Func, translates map[CountryCode]*TranslateDetail) {
	field := &validatorField{
		Tag:             tag,
		Translator:      make(map[CountryCode]*TranslateDetail, len(translates)),
		LogicValidation: logicFn,
	}
	for cc, detail := range translates {
		field.Translator[cc] = detail
	}
	c.fields[tag] = field
}
