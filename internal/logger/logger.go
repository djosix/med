package logger

type Logger string

var RootLogger Logger = ""

func NewLogger(name string) Logger {
	return RootLogger.NewLogger(name)
}

func (l Logger) NewLogger(name string) Logger {
	if l == "" {
		return Logger(name)
	} else {
		return Logger(l + "." + Logger(name))
	}
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

func (l Logger) Print(a ...any) (n int, err error) {
	return Print(l.prependPrefixToAnySlice(a)...)
}

func (l Logger) Printf(format string, a ...any) (n int, err error) {
	return Printf(l.prependPrefixToString(format), a...)
}

func (l Logger) Debug(a ...any) (n int, err error) {
	return Debug(l.prependPrefixToAnySlice(a)...)
}

func (l Logger) Debugf(format string, a ...any) (n int, err error) {
	return Debugf(l.prependPrefixToString(format), a...)
}

func (l Logger) Info(a ...any) (n int, err error) {
	return Info(l.prependPrefixToAnySlice(a)...)
}

func (l Logger) Infof(format string, a ...any) (n int, err error) {
	return Infof(l.prependPrefixToString(format), a...)
}

func (l Logger) Warn(a ...any) (n int, err error) {
	return Warn(l.prependPrefixToAnySlice(a)...)
}

func (l Logger) Warnf(format string, a ...any) (n int, err error) {
	return Warnf(l.prependPrefixToString(format), a...)
}

func (l Logger) Error(a ...any) (n int, err error) {
	return Error(l.prependPrefixToAnySlice(a)...)
}

func (l Logger) Errorf(format string, a ...any) (n int, err error) {
	return Errorf(l.prependPrefixToString(format), a...)
}

func (l Logger) Fatal(a ...any) (n int, err error) {
	return Fatal(l.prependPrefixToAnySlice(a)...)
}

func (l Logger) Fatalf(format string, a ...any) (n int, err error) {
	return Fatalf(l.prependPrefixToString(format), a...)
}
