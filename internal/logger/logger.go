package logger

import (
	"fmt"
)

type Logger string

var RootLogger Logger = ""

func NewLogger(name string) Logger {
	return RootLogger.NewLogger(name)
}

func NewLoggerf(format string, a ...any) Logger {
	return RootLogger.NewLoggerf(format, a...)
}

func (l Logger) NewLogger(name string) Logger {
	if l == "" {
		return Logger(name)
	} else {
		return Logger(l + "/" + Logger(name))
	}
}

func (l Logger) NewLoggerf(format string, a ...any) Logger {
	return l.NewLogger(fmt.Sprintf(format, a...))
}

func (l Logger) prependPrefixToString(s string) string {
	if l != "" {
		return string(l) + ": " + s
	}
	return s
}

func (l Logger) prependPrefixToAnySlice(a []any) []any {
	if l != "" {
		return append([]any{string(l) + ":"}, a...)
	}
	return a
}

func (l Logger) Print(a ...any) {
	Print(l.prependPrefixToAnySlice(a)...)
}

func (l Logger) Printf(format string, a ...any) {
	Printf(l.prependPrefixToString(format), a...)
}

func (l Logger) Debug(a ...any) {
	if MaxLogLevel >= LevelDebug {
		Debug(l.prependPrefixToAnySlice(a)...)
	}
}

func (l Logger) Debugf(format string, a ...any) {
	if MaxLogLevel >= LevelDebug {
		Debugf(l.prependPrefixToString(format), a...)
	}
}

func (l Logger) Info(a ...any) {
	if MaxLogLevel >= LevelInfo {
		Info(l.prependPrefixToAnySlice(a)...)
	}
}

func (l Logger) Infof(format string, a ...any) {
	if MaxLogLevel >= LevelInfo {
		Infof(l.prependPrefixToString(format), a...)
	}
}

func (l Logger) Warn(a ...any) {
	if MaxLogLevel >= LevelWarn {
		Warn(l.prependPrefixToAnySlice(a)...)
	}
}

func (l Logger) Warnf(format string, a ...any) {
	if MaxLogLevel >= LevelWarn {
		Warnf(l.prependPrefixToString(format), a...)
	}
}

func (l Logger) Error(a ...any) {
	if MaxLogLevel >= LevelError {
		Error(l.prependPrefixToAnySlice(a)...)
	}
}

func (l Logger) Errorf(format string, a ...any) {
	if MaxLogLevel >= LevelError {
		Errorf(l.prependPrefixToString(format), a...)
	}
}

func (l Logger) Fatal(a ...any) {
	if MaxLogLevel >= LevelFatal {
		Fatal(l.prependPrefixToAnySlice(a)...)
	}
}

func (l Logger) Fatalf(format string, a ...any) {
	if MaxLogLevel >= LevelFatal {
		Fatalf(l.prependPrefixToString(format), a...)
	}
}
