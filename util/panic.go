package util

import (
	"runtime"

	"github.com/pkg/errors"
)

func Recover() {
	if e := recover(); e != nil {
		stack := make([]byte, 1024)
		length := runtime.Stack(stack, false)
		err := errors.Errorf("panic: %v\nstatic: %v", e, string(stack[:length]))
		Errorw("panic", "err", err.Error())
	}
}
