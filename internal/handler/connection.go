package handler

import (
	"context"
	"net"
	"strings"

	"github.com/djosix/med/internal/logger"
)

func Listen(ctx context.Context, endpoint string, handler Handler, maxConn int) error {
	logger := logger.NewLogger("Listen")
	logger.Info("bind on", endpoint)

	listener, err := net.Listen(splitEndpoint(endpoint))
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
			logger.Info("accept", conn.RemoteAddr())
			if err != nil {
				logger.Error("accept:", err)
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
	logger.Info("target", endpoint)

	conn, err := net.Dial(splitEndpoint(endpoint))
	if err != nil {
		return err
	}
	defer conn.Close()

	logger.Info("connected to", conn.RemoteAddr())

	ctx = context.WithValue(ctx, "conn", conn)
	err = handler(ctx, conn)
	if err != nil {
		return err
	}

	return nil
}

func splitEndpoint(endpoint string) (string, string) {
	if strings.HasPrefix(endpoint, "unix:") {
		parts := strings.SplitN(endpoint, ":", 2)
		return parts[0], parts[1]
	}
	return "tcp", endpoint
}
