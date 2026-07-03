package validator

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/go-playground/locales"
	"github.com/go-playground/locales/en"
	"github.com/go-playground/locales/id"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
)

// ErrNotLoaded is returned when Validation/ValidationCtx is called before Load.
var ErrNotLoaded = errors.New("validator: Load() must be called before validating")

// CustomValidator wraps go-playground/validator with per-language error
// translation and custom rule registration. Build with the fluent chain:
//
//	v := New().
//		SetTranslateError(&myTranslate{}).
//		AddValidator("my_rule", myRuleFn, map[CountryCode]*TranslateDetail{...}).
//		Load()
type CustomValidator struct {
	*validator.Validate
	labelName            string
	translaterError      TranslaterError
	mappingTranslator    *mappingTranslator
	registerCustomFields *customFields
	customTypes          *customTypes
}

func New() *CustomValidator {
	return &CustomValidator{
		mappingTranslator:    newMappingTranslator(),
		customTypes:          newCustomTypes(),
		registerCustomFields: newCustomFields(),
	}
}

func (c *CustomValidator) defaultLanguage() {
	countries := []struct {
		cc         CountryCode
		translator locales.Translator
	}{
		{cc: CCEN, translator: en.New()},
		{cc: CCID, translator: id.New()},
	}
	for _, country := range countries {
		c.AddLanguage(country.cc, country.translator)
	}
}

func (c *CustomValidator) defaultTranslateError() {
	if c.translaterError == nil {
		c.translaterError = &defaultTranslateError{}
	}
}

func (c *CustomValidator) defaultLabel() {
	if c.labelName == "" {
		c.labelName = "label"
	}
	c.RegisterTagNameFunc(registerLabelName(c.labelName))
}

// SetLabel sets the struct tag used as the field display name (default "label").
func (c *CustomValidator) SetLabel(labelName string) *CustomValidator {
	c.labelName = labelName
	return c
}

// SetTranslateError overrides how validation errors are converted (Strategy).
func (c *CustomValidator) SetTranslateError(translateError TranslaterError) *CustomValidator {
	c.translaterError = translateError
	return c
}

// AddLanguage registers a translator for a country code.
func (c *CustomValidator) AddLanguage(language CountryCode, translator locales.Translator) *CustomValidator {
	uni := ut.New(translator, translator)
	translatorLanguage, found := uni.GetTranslator(language.String())
	if !found {
		fmt.Fprintf(os.Stderr, "validator: the language %s is not found\n", language.String())
		return c
	}

	c.mappingTranslator.AddTranslator(language, translatorLanguage)
	return c
}

// AddValidator registers a custom rule with its translations in one step.
func (c *CustomValidator) AddValidator(tag string, logicFn validator.Func, translates map[CountryCode]*TranslateDetail) *CustomValidator {
	c.registerCustomFields.add(tagValidator(tag), logicFn, translates)
	return c
}

// AddCustomTypeData registers a custom type extractor (validator.CustomTypeFunc).
func (c *CustomValidator) AddCustomTypeData(name string, customType *CustomType) *CustomValidator {
	c.customTypes.AddTypeData(name, customType)
	return c
}

// Load finalizes the configuration and makes the validator usable.
// It panics on misregistration (programmer error, fail fast at startup).
func (c *CustomValidator) Load() *CustomValidator {
	c.Validate = validator.New()
	c.defaultLanguage()
	c.defaultTranslateError()
	c.defaultLabel()

	c.registerCustomTypes()
	c.registerCustomValidation()
	c.registerDefaultTranslator()
	return c
}

// ValidationCtx validates s with context, translating errors to the given
// country code (default English).
func (c *CustomValidator) ValidationCtx(ctx context.Context, s any, cc ...CountryCode) error {
	if c.Validate == nil {
		return ErrNotLoaded
	}

	err := c.Validate.StructCtx(ctx, s)
	return c.translate(err, cc...)
}

// Validation validates s, translating errors to the given country code
// (default English).
func (c *CustomValidator) Validation(s any, cc ...CountryCode) error {
	if c.Validate == nil {
		return ErrNotLoaded
	}

	err := c.Validate.Struct(s)
	return c.translate(err, cc...)
}

func (c *CustomValidator) translate(err error, cc ...CountryCode) error {
	if err == nil {
		return nil
	}

	countryCode := CCEN
	if len(cc) > 0 {
		countryCode = cc[0]
	}

	if _, ok := errors.AsType[validator.ValidationErrors](err); ok {
		return c.translaterError.TranslateError(err, c.mappingTranslator.GetTranslator(countryCode))
	}
	return err
}

func (c *CustomValidator) registerCustomValidation() {
	for tag, field := range c.registerCustomFields.fields {
		if err := c.Validate.RegisterValidation(tag.String(), field.LogicValidation); err != nil {
			panic(err)
		}

		for countryCode, translateDetail := range field.Translator {
			registerFn, transFn := c.addTranslation(tag, translateDetail)
			err := c.Validate.RegisterTranslation(tag.String(), c.mappingTranslator.GetTranslator(countryCode), registerFn, transFn)
			if err != nil {
				panic(err)
			}
		}
	}
}

func (c *CustomValidator) registerCustomTypes() {
	for _, custom := range c.customTypes.types {
		c.Validate.RegisterCustomTypeFunc(custom.CustomFn, custom.TypeData)
	}
}

func (c *CustomValidator) registerDefaultTranslator() {
	for countryCode, translator := range c.mappingTranslator.data {
		c.mappingTranslator.RegisterDefaultTranslator(c.Validate, translator, countryCode)
	}
}
