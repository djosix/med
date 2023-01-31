package logger

import (
	"fmt"
	"io"
	"os"
	"reflect"
	"runtime"
	"strings"
	"time"
)

type LogLevel int

const (
	LevelDebug LogLevel = 5
	LevelInfo  LogLevel = 4
	LevelWarn  LogLevel = 3
	LevelError LogLevel = 2
	LevelFatal LogLevel = 1
	LevelNone  LogLevel = 0

	logTimeFormat = "2006-01-02 15:04:05"

	AnsiRed     = "\033[91m"
	AnsiGreen   = "\033[92m"
	AnsiYellow  = "\033[93m"
	AnsiBlue    = "\033[94m"
	AnsiMeganta = "\033[95m"
	AnsiCyan    = "\033[96m"
	AnsiReset   = "\033[0m"
)

var (
	logTarget          io.Writer = os.Stderr
	logLevel           LogLevel  = LevelInfo
	logLevelNames                = []string{"FATAL", "ERROR", "WARN", "INFO", "DEBUG"}
	logLevelAnsiColors           = []string{AnsiMeganta, AnsiRed, AnsiYellow, AnsiCyan, AnsiGreen}
	logTimeAndLevel              = true
	logCaller                    = false
	logColors                    = true
)

func SetLevel(level LogLevel) {
	if level > MaxLogLevel || level < LevelNone || level > LevelDebug {
		panic("invalid log level")
	}
	logLevel = level
}

func SetShowTimeAndLevel(enabled bool, withColors bool) {
	logTimeAndLevel = enabled
	logColors = withColors
}

func SwapTarget(w io.Writer) io.Writer {
	oldTarget := logTarget
	logTarget = w
	return oldTarget
}

func getDateTime() string {
	return time.Now().Format(logTimeFormat)
}

func Print(a ...any) {
	if logTarget == nil {
		return
	}
	fmt.Fprintln(logTarget, a...)
}

func Printf(format string, a ...any) {
	Print(fmt.Sprintf(format, a...))
}

func Log(level LogLevel, a ...any) {
	if logTarget == nil {
		return
	}
	s := []any{}
	if logTimeAndLevel {
		levelName := fmt.Sprintf("%-5s", logLevelNames[level-1])
		if logColors {
			levelColor := logLevelAnsiColors[level-1]
			levelName = fmt.Sprintf("%s%s%s", levelColor, levelName, AnsiReset)
		}
		s = append(s, fmt.Sprintf("%s %-5s", getDateTime(), levelName))
	}
	if logCaller {
		s = append(s, getCaller())
	}
	if len(s) > 0 {
		s = append(s, "|")
	}
	Print(append(s, a...)...)
}

func Logf(level LogLevel, format string, a ...any) {
	Log(level, fmt.Sprintf(format, a...))
}

func Debug(a ...any) {
	if MaxLogLevel >= LevelDebug && logLevel >= LevelDebug {
		Log(LevelDebug, a...)
	}
}

func Debugf(format string, a ...any) {
	if MaxLogLevel >= LevelDebug && logLevel >= LevelDebug {
		Logf(LevelDebug, format, a...)
	}
}

func Info(a ...any) {
	if MaxLogLevel >= LevelInfo && logLevel >= LevelInfo {
		Log(LevelInfo, a...)
	}
}

func Infof(format string, a ...any) {
	if MaxLogLevel >= LevelInfo && logLevel >= LevelInfo {
		Logf(LevelInfo, format, a...)
	}
}

func Warn(a ...any) {
	if MaxLogLevel >= LevelWarn && logLevel >= LevelWarn {
		Log(LevelWarn, a...)
	}
}

func Warnf(format string, a ...any) {
	if MaxLogLevel >= LevelWarn && logLevel >= LevelWarn {
		Logf(LevelWarn, format, a...)
	}
}

func Error(a ...any) {
	if MaxLogLevel >= LevelError && logLevel >= LevelError {
		Log(LevelError, a...)
	}
}

func Errorf(format string, a ...any) {
	if MaxLogLevel >= LevelError && logLevel >= LevelError {
		Logf(LevelError, format, a...)
	}
}

func Fatal(a ...any) {
	if MaxLogLevel >= LevelFatal && logLevel >= LevelFatal {
		Log(LevelFatal, a...)
	}
}

func Fatalf(format string, a ...any) {
	if MaxLogLevel >= LevelFatal && logLevel >= LevelFatal {
		Logf(LevelFatal, format, a...)
	}
}

func getCaller() string {
	minCallerSkip := 3
	loggerPkgPath := reflect.TypeOf(RootLogger).PkgPath()

	pc := make([]uintptr, 8)
	runtime.Callers(minCallerSkip, pc) //
	frames := runtime.CallersFrames(pc)

	for {
		frame, more := frames.Next()

		if strings.HasPrefix(frame.Function, loggerPkgPath) {
			if more {
				continue
			} else {
				break
			}
		}

		var pkgName string
		var funcLongName string
		{
			i := strings.LastIndex(frame.Function, "/")
			s := frame.Function[i+1:]
			i = strings.Index(s, ".")
			pkgName = s[:i]
			funcLongName = s[i+1:]
		}

		return fmt.Sprintf(
			"%v:%v %v.%v",
			frame.File,
			frame.Line,
			pkgName,
			funcLongName,
		)
	}

	return ""
}
