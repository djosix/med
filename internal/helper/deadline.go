package helper

import (
	"time"
)

var (
	aDistantPast   = time.Unix(1, 0)
	aDistantFuture = time.Unix(10000000000, 0)
)

type ReadDeadlineSetter interface {
	SetReadDeadline(t time.Time) error
}
type WriteDeadlineSetter interface {
	SetWriteDeadline(t time.Time) error
}

type DeadlineSetter interface {
	SetDeadline(t time.Time) error
}

func BreakReadWrite(s DeadlineSetter) error {
	return s.SetDeadline(aDistantPast)
}

func RefreshReadWrite(s DeadlineSetter) error {
	return s.SetDeadline(aDistantFuture)
}

func BreakRead(s ReadDeadlineSetter) error {
	return s.SetReadDeadline(aDistantPast)
}

func RefreshRead(s ReadDeadlineSetter) error {
	return s.SetReadDeadline(aDistantFuture)
}

func BreakWrite(s WriteDeadlineSetter) error {
	return s.SetWriteDeadline(aDistantPast)
}

func RefreshWrite(s WriteDeadlineSetter) error {
	return s.SetWriteDeadline(aDistantFuture)
}
