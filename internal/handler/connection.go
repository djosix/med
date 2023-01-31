package handler

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/djosix/med/internal/helper"
	"github.com/djosix/med/internal/logger"
)

func Connect(ctx context.Context, endpoint string, handler Handler, connInt time.Duration) error {
	logger := logger.NewLogger("connect")
	logger.Info("target", endpoint)

	connect := func() error {
		conn, err := net.Dial(helper.SplitEndpoint(endpoint))
		if err != nil {
			return err
		}
		defer conn.Close()

		logger.Info("connected to", conn.RemoteAddr())

		ctx := context.WithValue(ctx, "conn", conn)
		if err = handler(ctx, conn); err != nil {
			return err
		}

		return nil
	}

	for {
		err := connect()
		if connInt <= 0 {
			return err
		}
		time.Sleep(connInt)
	}
}

func Listen(ctx context.Context, endpoint string, handler Handler, maxConn int) error {
	logger := logger.NewLogger("listen")
	logger.Info("bind on", endpoint)

	var listener net.Listener
	{
		network, address := helper.SplitEndpoint(endpoint)
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
				logger.Errorf("handler addr=%#v error=%#v", fmt.Sprint(conn.RemoteAddr()), err.Error())
			}
		}()
	}

	wg.Wait()

	return nil
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
