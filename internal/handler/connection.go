package handler

import (
	"context"
	"net"

	"github.com/djosix/med/internal/logger"
)

func Listen(ctx context.Context, endpoint string, handler Handler, maxConn int) error {
	logger := logger.NewLogger("Listen")
	logger.Info("Bind on", endpoint)

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
				logger.Error("Accept:", err)
				continue
			}
			connCh <- conn
		}
	}()

	for {
		select {
		case conn := <-connCh:
			go func(conn net.Conn, gateCh chan struct{}) {
				defer func() {
					conn.Close()
					logger.Info("closed connection to", conn.RemoteAddr())
				}()

				ctx := context.WithValue(ctx, "conn", conn)
				if err := handler(ctx, conn); err != nil {
					logger.Errorf("handler for [%v] error [%v]", conn.RemoteAddr(), err)
				}

				gateCh <- struct{}{}
			}(conn, gateCh)
		case <-ctx.Done():
			break
		}
	}
}

func Connect(ctx context.Context, endpoint string, handler Handler) error {
	logger := logger.NewLogger("Connect")
	logger.Info(endpoint)

	conn, err := net.Dial("tcp", endpoint)
	if err != nil {
		return err
	}
	defer conn.Close()

	logger.Info("Connected to", conn.RemoteAddr())

	ctx = context.WithValue(ctx, "conn", conn)
	err = handler(ctx, conn)
	if err != nil {
		return err
	}

	return nil
}
