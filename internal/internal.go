package internal

import "fmt"

var (
	Err        = fmt.Errorf("error")
	Unexpected = fmt.Errorf("unexpected")
)

func Panicf(format string, a ...any) {
	panic(fmt.Sprintf(format, a...))
}

func Panicln(a ...any) {
	panic(fmt.Sprintln(a...))
}
