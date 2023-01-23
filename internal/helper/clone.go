package helper

func Clone[T any](s []T) []T {
	return append([]T{}, s...)
}
