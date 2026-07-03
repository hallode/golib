package custerr

import "errors"

type NonRetryableError struct {
	err error
}

func (e *NonRetryableError) Error() string {
	if e.err != nil {
		return e.err.Error()
	}
	return "non-retryable error"
}

func (e *NonRetryableError) Unwrap() error { return e.err }

func (e *NonRetryableError) Source() string {
	if s, ok := e.err.(Sourcer); ok {
		return s.Source()
	}
	return ""
}

func NewNonRetryable(err error) error {
	if err == nil {
		return nil
	}
	return &NonRetryableError{err: Wrap(err)}
}

func IsNonRetryable(err error) bool {
	var nre *NonRetryableError
	return errors.As(err, &nre)
}
