package internal

import (
	"errors"
	"fmt"
	"github.com/fatih/color"
	"github.com/masuldev/merrwrap"
	"os"
	"runtime"
	"strings"
)

func RealPanic(err error) {
	fmt.Println(color.RedString("err: %s", err.Error()))
	os.Exit(1)
}

func WrapError(err error) error {
	if err != nil {
		pc, _, line, _ := runtime.Caller(2)
		fn := runtime.FuncForPC(pc).Name()
		details := strings.Split(fn, "/")
		fn = details[len(details)-1]
		wrapError := merrwrap.Error(err)
		return wrapError.Wrap(fmt.Errorf("err:[%s:%d]", fn, line))
	}
	return nil
}

var (
	ErrInvalidParam = errors.New("err: invalid params")
)
