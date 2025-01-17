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
	Error() string
	StackTrace() errors.StackTrace
}

type unwrapper interface {
	Unwrap() error
}

func FormatError(err error) string {
	if err == nil {
		return "nil"
	}

	str := err.Error()

	var topFrames, trace []errors.Frame
	var errWithTrace stackTracer
	var ok bool
	for {
		errWithTrace, ok = err.(stackTracer)
		if !ok {
			break
		}
		trace = errWithTrace.StackTrace()
		topFrames = append(topFrames, trace[0]) // Save the frame where Wrap is called
		err = unwrapStackTracer(errWithTrace)
	}

	if len(topFrames) > 0 {
		topFrames = topFrames[:len(topFrames)-1] // the last top frame is the first frame from trace
	}
	if len(trace) > 2 {
		trace = trace[:len(trace)-2] // skip what calls our main function (main|proc.go:271 and goexit|asm_amd64.s:1695)
	}
	if len(trace) > 5 {
		trace = trace[:5] // if stack trace is still too long, only keep first 5 locations
	}
	trace = append(topFrames, trace...)

	if len(topFrames) > 0 {
		str += ": wrapped by "
	}
	for i, f := range trace {
		if i == len(topFrames) {
			str += " caused by "
		} else if i != 0 {
			str += "->"
		}
		str += fmt.Sprintf("[%n|%v]", f, f)
	}

	return str
}

// unwrapStackTracer calls Unwrap on the given stackTracer error until we reach another stackTracer. Returns nil if one can't be found
func unwrapStackTracer(err stackTracer) error {
	var curErr error = err
	for {
		if curErr == nil {
			return nil
		}
		unwrapperErr, ok := curErr.(unwrapper)
		if !ok {
			return nil
		}
		curErr = unwrapperErr.Unwrap()
		if _, ok = curErr.(stackTracer); ok {
			return curErr
		}
	}
}
