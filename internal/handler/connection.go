package handler

import (
	"context"
	"net"

	"github.com/djosix/med/internal/logger"
)

func Listen(ctx context.Context, endpoint string, handler Handler, maxConn int) error {
	listener, err := net.Listen("tcp", endpoint)
	if err != nil {
		return err
	}
	defer listener.Close()

	gateCh := make(chan struct{}, maxConn)
	connCh := make(chan net.Conn, 0)

	go func() {
		for len(gateCh) < cap(gateCh) {
			gateCh <- struct{}{}
		}
		for {
			<-gateCh
			conn, err := listener.Accept()
			if err != nil {
				logger.Log(err)
				continue
			}
			connCh <- conn
		}
	}()

	for {
		select {
		case conn := <-connCh:
			go func(conn net.Conn, gateCh chan struct{}) {
				defer conn.Close()
				logger.Log("accept:", conn.RemoteAddr())

				ctx := context.WithValue(ctx, "conn", conn)
				if err := handler(ctx, conn); err != nil {
					logger.Log("error:", conn.RemoteAddr(), err)
				}

				gateCh <- struct{}{}
			}(conn, gateCh)
		case <-ctx.Done():
			break
		}
	}
}

func Connect(ctx context.Context, endpoint string, handler Handler) error {
	conn, err := net.Dial("tcp", endpoint)
	if err != nil {
		return err
	}
	defer conn.Close()

	ctx = context.WithValue(ctx, "conn", conn)
	err = handler(ctx, conn)
	if err != nil {
		return err
	}

	return nil
}
