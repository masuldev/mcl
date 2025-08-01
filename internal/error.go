package internal

import (
	"errors"
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/masuldev/merrwrap"
)

func RealPanic(err error) {
	LogError("err: %s", err.Error())
	os.Exit(1)
}

func PrintError(err error) {
	LogError("err: %s", err.Error())
}

func WrapError(err error) error {
	if err != nil {
		pc, _, line, _ := runtime.Caller(2)
		fn := runtime.FuncForPC(pc).Name()
		details := strings.Split(fn, "/")
		fn = details[len(details)-1]
		wrapError := merrwrap.Error(err)
		return wrapError.Wrap(fmt.Errorf("[%s:%d]", fn, line))
	}
	return nil
}

var (
	ErrInvalidParam = errors.New("err: invalid params")
)
