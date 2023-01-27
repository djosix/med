package helper

import (
	"time"
)

type CanSetDeadline interface {
	SetDeadline(t time.Time) error
}

func BreakIO(s CanSetDeadline) error {
	return s.SetDeadline(time.Unix(1, 0))
}

func RefreshIO(s CanSetDeadline) error {
	return s.SetDeadline(time.Unix(10000000000, 0))
}
