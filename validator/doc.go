// Package validator wraps go-playground/validator/v10 with multi-language
// translated errors and custom rule registration.
//
// For the common case, use the package-level Validate (or ValidateCtx): it runs
// a ready-to-use default instance and returns a structured *CustomValidationError
// (JSON shape {"errors":[...]}). FiberValidator plugs the same instance into
// Fiber v3.
//
// To register custom rules or languages, build an instance with the fluent
// builder — New().AddValidator(...).AddLanguage(...).Load(). Load is mandatory:
// rules are compiled inside it, so call AddValidator/AddLanguage before Load,
// and calling the instance before Load returns ErrNotLoaded.
package validator
