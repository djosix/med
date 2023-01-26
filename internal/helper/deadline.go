package helper

import (
	"time"
)

type Deadlined interface {
	SetDeadline(t time.Time) error
}

func BreakIO(s Deadlined) error {
	return s.SetDeadline(time.Unix(1, 0))
}

func RefreshIO(s Deadlined) error {
	return s.SetDeadline(time.Unix(10000000000, 0))
}
