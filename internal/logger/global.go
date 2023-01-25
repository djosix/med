package logger

import (
	"fmt"
	"io"
	"os"
	"time"
)

type LogLevel int

const (
	LevelDebug    LogLevel = 5
	LevelInfo     LogLevel = 4
	LevelWarn     LogLevel = 3
	LevelError    LogLevel = 2
	LevelFatal    LogLevel = 1
	LevelNone     LogLevel = 0
	logTimeFormat          = "2006-01-02 15:04:05"
)

var (
	logTarget    io.Writer = os.Stdout
	logLevel     LogLevel  = LevelInfo
	logLevelText           = [6]string{
		"",
		"Fatal",
		"ERROR",
		"WARN",
		"INFO",
		"DEBUG",
	}
	logTimeLevel = true
)

func SetLevel(level LogLevel) {
	if int(level) < 0 || int(level) >= len(logLevelText) {
		panic("invalid level")
	}
	logLevel = level
}

func SetAddPrefix(value bool) {
	logTimeLevel = value
}

func SwapTarget(w io.Writer) io.Writer {
	oldTarget := logTarget
	logTarget = w
	return oldTarget
}

func getDateTime() string {
	return time.Now().Format(logTimeFormat)
}

func Print(a ...any) (n int, err error) {
	if logTarget == nil {
		return 0, nil
	}
	return fmt.Fprintln(logTarget, a...)
}

func Printf(format string, a ...any) (n int, err error) {
	return Print(fmt.Sprintf(format, a...))
}

func Log(level LogLevel, a ...any) (n int, err error) {
	if logTarget == nil {
		return 0, nil
	}
	s := []any{}
	if logTimeLevel {
		s = append(s, fmt.Sprintf("%s [%5s]", getDateTime(), logLevelText[level]))
	}
	return Print(append(s, a...)...)
}

func Logf(level LogLevel, format string, a ...any) (n int, err error) {
	return Log(level, fmt.Sprintf(format, a...))
}

func Debug(a ...any) (n int, err error) {
	if logLevel >= LevelDebug {
		return Log(LevelDebug, a...)
	}
	return 0, nil
}

func Debugf(format string, a ...any) (n int, err error) {
	if logLevel >= LevelDebug {
		return Logf(LevelDebug, format, a...)
	}
	return 0, nil
}

func Info(a ...any) (n int, err error) {
	if logLevel >= LevelInfo {
		return Log(LevelInfo, a...)
	}
	return 0, nil
}

func Infof(format string, a ...any) (n int, err error) {
	if logLevel >= LevelInfo {
		return Logf(LevelInfo, format, a...)
	}
	return 0, nil
}

func Warn(a ...any) (n int, err error) {
	if logLevel >= LevelWarn {
		return Log(LevelWarn, a...)
	}
	return 0, nil
}

func Warnf(format string, a ...any) (n int, err error) {
	if logLevel >= LevelWarn {
		return Logf(LevelWarn, format, a...)
	}
	return 0, nil
}

func Error(a ...any) (n int, err error) {
	if logLevel >= LevelError {
		return Log(LevelError, a...)
	}
	return 0, nil
}

func Errorf(format string, a ...any) (n int, err error) {
	if logLevel >= LevelError {
		return Logf(LevelError, format, a...)
	}
	return 0, nil
}

func Fatal(a ...any) (n int, err error) {
	if logLevel >= LevelFatal {
		return Log(LevelFatal, a...)
	}
	return 0, nil
}

func Fatalf(format string, a ...any) (n int, err error) {
	if logLevel >= LevelFatal {
		return Logf(LevelFatal, format, a...)
	}
	return 0, nil
}
