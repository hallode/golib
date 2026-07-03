package validator

import (
	"errors"
	"fmt"
	"strings"

	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	"github.com/go-playground/validator/v10/translations/ar"
	"github.com/go-playground/validator/v10/translations/en"
	"github.com/go-playground/validator/v10/translations/es"
	"github.com/go-playground/validator/v10/translations/fa"
	"github.com/go-playground/validator/v10/translations/fr"
	"github.com/go-playground/validator/v10/translations/id"
	"github.com/go-playground/validator/v10/translations/it"
	"github.com/go-playground/validator/v10/translations/ja"
	"github.com/go-playground/validator/v10/translations/lv"
	"github.com/go-playground/validator/v10/translations/nl"
	"github.com/go-playground/validator/v10/translations/pt"
	"github.com/go-playground/validator/v10/translations/pt_BR"
	"github.com/go-playground/validator/v10/translations/ru"
	"github.com/go-playground/validator/v10/translations/tr"
	"github.com/go-playground/validator/v10/translations/vi"
	"github.com/go-playground/validator/v10/translations/zh"
	"github.com/go-playground/validator/v10/translations/zh_tw"
)

type mappingTranslator struct {
	data              map[CountryCode]ut.Translator
	defaultTranslator map[CountryCode]func(v *validator.Validate, trans ut.Translator) error
}

type TranslaterError interface {
	TranslateError(errValidate error, translator ut.Translator) (err error)
}

func newMappingTranslator() *mappingTranslator {
	return &mappingTranslator{
		data: make(map[CountryCode]ut.Translator),
		defaultTranslator: map[CountryCode]func(v *validator.Validate, trans ut.Translator) error{
			CCAR:   ar.RegisterDefaultTranslations,
			CCES:   es.RegisterDefaultTranslations,
			CCEN:   en.RegisterDefaultTranslations,
			CCFA:   fa.RegisterDefaultTranslations,
			CCFR:   fr.RegisterDefaultTranslations,
			CCID:   id.RegisterDefaultTranslations,
			CCIT:   it.RegisterDefaultTranslations,
			CCJA:   ja.RegisterDefaultTranslations,
			CCLV:   lv.RegisterDefaultTranslations,
			CCNL:   nl.RegisterDefaultTranslations,
			CCPT:   pt.RegisterDefaultTranslations,
			CCPTBR: pt_BR.RegisterDefaultTranslations,
			CCRU:   ru.RegisterDefaultTranslations,
			CCTR:   tr.RegisterDefaultTranslations,
			CCVI:   vi.RegisterDefaultTranslations,
			CCZH:   zh.RegisterDefaultTranslations,
			CCZHTW: zh_tw.RegisterDefaultTranslations,
		},
	}
}

func (m *mappingTranslator) AddTranslator(cc CountryCode, translator ut.Translator) {
	m.data[cc] = translator
}

func (m *mappingTranslator) GetTranslator(ccs ...CountryCode) ut.Translator {
	cc := CCEN

	if len(ccs) > 0 {
		cc = ccs[0]
	}

	if translator, ok := m.data[cc]; ok {
		return translator
	}

	return m.data[CCEN]
}

func (m *mappingTranslator) RegisterDefaultTranslator(v *validator.Validate, trans ut.Translator, cc CountryCode) {
	registerFn, ok := m.defaultTranslator[cc]
	if !ok {
		panic(fmt.Sprintf("validator: no default translations registered for language %q", cc))
	}
	if err := registerFn(v, trans); err != nil {
		panic(err)
	}
}

type defaultTranslateError struct{}

func (d *defaultTranslateError) TranslateError(errValidate error, translator ut.Translator) error {
	var validatorErrs validator.ValidationErrors
	if !errors.As(errValidate, &validatorErrs) {
		return errValidate
	}

	messages := make([]string, len(validatorErrs))
	for i, e := range validatorErrs {
		messages[i] = e.Translate(translator)
	}

	return errors.New(strings.Join(messages, "\n"))
}

type ParamFn func(param string) string

type TranslateDetail struct {
	Message string
	Param   ParamFn
}

func NewTranslateDetail(message string, params ...ParamFn) *TranslateDetail {
	var param ParamFn
	if len(params) > 0 {
		param = params[0]
	}

	return &TranslateDetail{
		Message: message,
		Param:   param,
	}
}

func (c *CustomValidator) addTranslation(tagName tagValidator, transDetail *TranslateDetail) (registerFn func(translator ut.Translator) error, transFn func(ut ut.Translator, fe validator.FieldError) string) {
	registerFn = func(ut ut.Translator) error {
		return ut.Add(tagName.String(), transDetail.Message, true)
	}

	transFn = func(ut ut.Translator, fe validator.FieldError) string {
		param := fe.Param()
		if transDetail.Param != nil {
			param = transDetail.Param(param)
		}

		tag := fe.Tag()

		t, err := ut.T(tag, fe.Field(), param)
		if err != nil {
			return fe.(error).Error()
		}
		return t
	}

	return
}
