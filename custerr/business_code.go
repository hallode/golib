package custerr

import "net/http"

const (
	// Business code format: CCDDNNN (7 digits)
	// CC = category, DD = domain, NNN = sequence.
	// Services define their own domain codes as plain constants and attach
	// them via Custer.WithCode; only generic fallback codes live here.
	//
	// auth/general (10 00 NNN)
	CodeAuthUnauthorized = 1000001
	CodeAuthForbidden    = 1000002

	// validation/general (20 00 NNN)
	CodeValidationBadRequest    = 2000001
	CodeValidationNotFound      = 2000002
	CodeValidationConflict      = 2000003
	CodeValidationUnprocessable = 2000004

	// integration/general (50 00 NNN)
	CodeIntegrationExternal = 5000001

	// internal/unknown (90 00 NNN)
	CodeInternalUnknown = 9000001
)

var statusFallbacks = map[int]int{
	http.StatusUnauthorized:        CodeAuthUnauthorized,
	http.StatusForbidden:           CodeAuthForbidden,
	http.StatusBadRequest:          CodeValidationBadRequest,
	http.StatusNotFound:            CodeValidationNotFound,
	http.StatusConflict:            CodeValidationConflict,
	http.StatusUnprocessableEntity: CodeValidationUnprocessable,
	http.StatusBadGateway:          CodeIntegrationExternal,
}

func RegisterStatusFallback(httpStatus, code int) {
	statusFallbacks[httpStatus] = code
}

func BusinessCodeByStatus(status int) int {
	if code, ok := statusFallbacks[status]; ok {
		return code
	}
	if status >= http.StatusInternalServerError {
		return CodeInternalUnknown
	}
	return CodeValidationBadRequest
}
