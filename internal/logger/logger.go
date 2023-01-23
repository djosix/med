package logger

import (
	"fmt"
	"io"
	"os"
	"time"
)

var LoggingTarget io.Writer

func init() {
	LoggingTarget = os.Stdout
}

func SwapTarget(w io.Writer) io.Writer {
	oldTarget := LoggingTarget
	LoggingTarget = w
	return oldTarget
}

func Log(a ...any) (n int, err error) {
	if LoggingTarget == nil {
		return 0, nil
	}
	s := fmt.Sprintf("%v | %v", getNowString(), fmt.Sprintln(a...))
	return fmt.Fprint(LoggingTarget, s)
}

func Logf(format string, a ...any) (n int, err error) {
	return Log(fmt.Sprintf(format, a...))
}

func getNowString() string {
	return time.Now().Format(time.RFC3339)
}
