package custerr

import (
	"fmt"
	"runtime"
	"strings"
)

var appModule = "github.com/hallode/golib/"

// SetAppModule sets the module path prefix stripped from captured source paths (e.g. "my-service/").
// Call once at startup.
func SetAppModule(module string) {
	if module != "" && !strings.HasSuffix(module, "/") {
		module += "/"
	}
	appModule = module
}

type Sourcer interface {
	Source() string
}

type Messager interface {
	Message() string
}

type stackError struct {
	err    error
	msg    string
	source string
}

func (e *stackError) Error() string {
	if e.msg != "" {
		return e.msg + ": " + e.err.Error()
	}
	return e.err.Error()
}

func (e *stackError) Message() string {
	if e.msg != "" {
		return e.msg
	}
	if m, ok := e.err.(Messager); ok {
		return m.Message()
	}
	return e.err.Error()
}

func (e *stackError) Unwrap() error  { return e.err }
func (e *stackError) Source() string { return e.source }

func Wrap(err error) error {
	if err == nil {
		return nil
	}
	return &stackError{err: err, source: captureSource(2)}
}

func Wrapf(err error, format string, args ...any) error {
	if err == nil {
		return nil
	}
	return &stackError{
		err:    err,
		msg:    fmt.Sprintf(format, args...),
		source: captureSource(2),
	}
}

// CaptureSource returns the formatted source location of a caller.
// skip follows runtime.Caller semantics: 1 = direct caller of CaptureSource.
func CaptureSource(skip int) string {
	return captureSource(skip + 1)
}

func captureSource(skip int) string {
	pc, file, line, ok := runtime.Caller(skip)
	if !ok {
		return ""
	}

	relFile := file
	if _, after, ok0 := strings.Cut(file, appModule); ok0 {
		relFile = after
	}
	fullPath := "/" + strings.TrimSuffix(appModule, "/") + "/" + relFile

	src := fmt.Sprintf("%s:%d", fullPath, line)
	if fn := runtime.FuncForPC(pc); fn != nil {
		fullName := fn.Name()
		qualifiedFunc := fullName
		if i := strings.LastIndex(fullName, "/"); i >= 0 {
			qualifiedFunc = fullName[i+1:]
		}
		src = fmt.Sprintf("%s:%d (%s)", fullPath, line, qualifiedFunc)
	}

	return src
}
