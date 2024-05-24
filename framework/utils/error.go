package utils

import (
	"fmt"

	"github.com/pkg/errors"
)

var (
	ErrNotFoundCallBack = errors.New("not found callBack function")
	ErrServerClosed     = errors.New("server has been closed")
	ErrWouldBlock       = errors.New("would block")
)

// ErrUndefined
type ErrUndefined int32

func (e ErrUndefined) Error() string {
	return fmt.Sprintf("undefined message type %d", e)
}

type stackTracer interface {
	StackTrace() errors.StackTrace
}

func FormatError(err error) string {
	if err == nil {
		return "nil"
	}

	str := err.Error()
	cause := errors.Cause(err)
	if causeStack, ok := cause.(stackTracer); ok {
		str += "  "
		for i, f := range causeStack.StackTrace() {
			if i != 0 {
				str += "->"
			}
			str += fmt.Sprintf("[%n|%s:%d]", f, f, f)
		}
	}
	return str
}
