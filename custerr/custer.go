package custerr

import (
	"fmt"
	"net/http"
)

type Custer struct {
	Code       int    `json:"code,omitempty"`
	StatusCode int    `json:"status_code"`
	Err        error  `json:"error"`
	Args       any    `json:"args,omitempty"`
	source     string // caller source location, auto-captured
}

func (e *Custer) Error() string {
	if e.Err != nil {
		return e.Err.Error()
	}
	return "unknown error"
}

func (e *Custer) Message() string {
	if e.Err != nil {
		return e.Err.Error()
	}
	return "unknown error"
}

func (e *Custer) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

func (e *Custer) Source() string { return e.source }

func (e *Custer) WithCode(code int) *Custer {
	e.Code = code
	return e
}

func (e *Custer) WithArgs(args any) *Custer {
	e.Args = args
	return e
}

func New(statusCode int, err error) *Custer {
	return newCuster(statusCode, err, 3)
}

func Newf(statusCode int, format string, args ...any) *Custer {
	return newCusterf(statusCode, 3, format, args...)
}

// newCuster is the internal constructor; skip controls how many frames
// runtime.Caller skips so the captured source points to the public caller.
func newCuster(statusCode int, err error, skip int) *Custer {
	return &Custer{StatusCode: statusCode, Err: err, source: captureSource(skip)}
}

func newCusterf(statusCode int, skip int, format string, args ...any) *Custer {
	return &Custer{
		StatusCode: statusCode,
		Err:        fmt.Errorf(format, args...),
		source:     captureSource(skip),
	}
}

func BadRequest(err error) *Custer { return newCuster(http.StatusBadRequest, err, 3) }

func BadRequestf(format string, args ...any) *Custer {
	return newCusterf(http.StatusBadRequest, 3, format, args...)
}

func Unauthorized(err error) *Custer { return newCuster(http.StatusUnauthorized, err, 3) }

func Unauthorizedf(format string, args ...any) *Custer {
	return newCusterf(http.StatusUnauthorized, 3, format, args...)
}

func Forbidden(err error) *Custer { return newCuster(http.StatusForbidden, err, 3) }

func Forbiddenf(format string, args ...any) *Custer {
	return newCusterf(http.StatusForbidden, 3, format, args...)
}

func NotFound(err error) *Custer { return newCuster(http.StatusNotFound, err, 3) }

func NotFoundf(format string, args ...any) *Custer {
	return newCusterf(http.StatusNotFound, 3, format, args...)
}

func Conflict(err error) *Custer { return newCuster(http.StatusConflict, err, 3) }

func Conflictf(format string, args ...any) *Custer {
	return newCusterf(http.StatusConflict, 3, format, args...)
}

func InternalServerError(err error) *Custer {
	return newCuster(http.StatusInternalServerError, err, 3)
}

func InternalServerErrorf(format string, args ...any) *Custer {
	return newCusterf(http.StatusInternalServerError, 3, format, args...)
}
