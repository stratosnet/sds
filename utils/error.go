package utils

import (
	"errors"
	"fmt"
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
