package handler

import (
	"context"
	"net"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"

	"github.com/djosix/med/internal/helper"
	"github.com/djosix/med/internal/logger"
)

func Listen(ctx context.Context, endpoint string, handler Handler, maxConn int) error {
	logger := logger.NewLogger("Listen")
	logger.Info("bind on", endpoint)

	logger.Debug("est")
	var listener net.Listener
	{
		network, address := splitEndpoint(endpoint)
		if l, err := net.Listen(network, address); err != nil {
			return err
		} else {
			listener = l
		}
		if network == "unix" {
			unixSockFile := address
			defer os.Remove(unixSockFile)
		}
	}
	defer listener.Close()

	// Graceful shutdown
	{
		var cancel context.CancelFunc
		ctx, cancel = context.WithCancel(ctx)

		handleSignal(func() {
			cancel()
			listener.Close()
		}, syscall.SIGINT, syscall.SIGTERM)
	}

	gate := helper.NewGate(ctx, maxConn)
	wg := sync.WaitGroup{}

	for {
		if !gate.Enter() {
			break // gate closed
		}

		conn, err := listener.Accept()
		if err != nil {
			logger.Error("accept:", err)
			continue
		}

		logger.Info("accept", conn.RemoteAddr())

		wg.Add(1)
		go func() {
			defer func() {
				conn.Close()
				logger.Info("closed connection to", conn.RemoteAddr())

				gate.Leave()
				wg.Done()
			}()

			ctx := context.WithValue(ctx, "conn", conn)
			if err := handler(ctx, conn); err != nil {
				logger.Errorf("handler for [%v] error [%v]", conn.RemoteAddr(), err)
			}
		}()
	}

	wg.Wait()

	return nil
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

func splitEndpoint(endpoint string) (network string, address string) {
	if strings.HasPrefix(endpoint, "unix:") {
		ss := strings.SplitN(endpoint, ":", 2)
		network = ss[0]
		address = ss[1]
	} else {
		network = "tcp"
		address = endpoint
	}
	return
}

func handleSignal(f func(), sig ...os.Signal) {
	sigCh := make(chan os.Signal)
	signal.Notify(sigCh, sig...)
	go func() {
		logger.Info(<-sigCh)
		signal.Stop(sigCh)
		f()
	}()
}
