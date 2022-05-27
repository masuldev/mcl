package internal

import (
	"fmt"
	"github.com/fatih/color"
	"os"
)

func RealPanic(err error) {
	fmt.Println(color.RedString("[err] %s", err.Error()))
	os.Exit(1)
}

//func WrapError(err error) error {
//	if err != nil {
//		pc, _, line, _ := runtime.Caller(1)
//		fn := runtime.FuncForPC(pc).Name()
//		details := strings.Split(fn, "/")
//		fn = details[len(details)-1]
//	}
//	return nil
//}
