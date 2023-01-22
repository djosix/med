package helper

import (
	"net"
	"time"
)

func BreakConnRW(conn net.Conn) {
	conn.SetDeadline(time.Unix(1, 0))
}

func RefreshConnRW(conn net.Conn) {
	conn.SetDeadline(time.Unix(10000000000, 0))
}
