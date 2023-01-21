package handler

import (
	"context"
	"log"
	"net"
)

func Listen(ctx context.Context, endpoint string, handler Handler, maxConn int) error {
	listener, err := net.Listen("tcp", endpoint)
	if err != nil {
		return err
	}
	defer listener.Close()

	gateCh := make(chan struct{}, maxConn)
	connCh := make(chan net.Conn, 4)

	go func() {
		for len(gateCh) < cap(gateCh) {
			gateCh <- struct{}{}
		}
		for {
			<-gateCh
			conn, err := listener.Accept()
			if err != nil {
				log.Fatalln(err)
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
				log.Println("accept:", conn.RemoteAddr())

				if err := handler(ctx, conn); err != nil {
					log.Println("error:", conn.RemoteAddr(), err)
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

	err = handler(ctx, conn)
	if err != nil {
		return err
	}

	return nil
}
