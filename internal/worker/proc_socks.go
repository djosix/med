package worker

import (
	"context"
	"io"
	"net"
	"sync"
	"time"

	"github.com/djosix/med/internal/logger"
	"github.com/things-go/go-socks5"
)

type SocksSpec struct {
	ListenEndpoint string
}

// Client

type SocksProcClient struct {
	ProcInfo
	spec SocksSpec
}

func NewSocksProcClient(spec SocksSpec) *SocksProcClient {
	return &SocksProcClient{
		ProcInfo: NewProcInfo(ProcKind_Socks, ProcSide_Client),
		spec:     spec,
	}
}

func (p *SocksProcClient) Run(ctx *ProcRunCtx) {
	logger := logger.NewLogger(string(p.Kind()))
	logger.Debug("start")
	defer logger.Debug("done")

	dataIn := pfDefaultDataInFn(ctx)
	dataOut := pfDefaultDataOutFn(ctx)

	err := pfListen(ctx, p.spec.ListenEndpoint, dataIn, dataOut)
	if err != nil {
		logger.Error(err)
	}
}

// Server

type SocksProcServer struct {
	ProcInfo
}

func NewSocksProcServer() *SocksProcServer {
	return &SocksProcServer{
		ProcInfo: NewProcInfo(ProcKind_Socks, ProcSide_Server),
	}
}

func (p *SocksProcServer) Run(ctx *ProcRunCtx) {
	logger := logger.NewLogger(string(p.Kind()))
	logger.Debug("start")
	defer logger.Debug("done")

	dataIn := pfDefaultDataInFn(ctx)
	dataOut := pfDefaultDataOutFn(ctx)
	l := NewMedSocksListener(ctx, dataIn, dataOut)

	server := socks5.NewServer()
	if err := server.Serve(l); err != nil {
		logger.Error(err)
	}

	l.Wait()
}

//

type MedSocksListener struct {
	ctx       context.Context
	cancel    context.CancelFunc
	conns     sync.Map
	newConnCh chan *MedSocksConn
	dataOut   func(data []byte) bool
	wg        sync.WaitGroup
}

func NewMedSocksListener(
	ctx context.Context,
	dataIn func(ctx context.Context) []byte,
	dataOut func(data []byte) bool,
) *MedSocksListener {

	ctx, cancel := context.WithCancel(ctx)

	l := MedSocksListener{
		ctx:       ctx,
		cancel:    cancel,
		newConnCh: make(chan *MedSocksConn, 0),
		dataOut:   dataOut,
	}

	l.wg.Add(1)
	go func() {
		defer logger.Debug("listener closed")
		defer close(l.newConnCh)
		defer l.cancel()
		defer l.wg.Done()

		for {
			data := dataIn(ctx)
			if len(data) == 0 {
				break
			}
			state, idx, data, err := pfDecode(data)
			if err != nil {
				logger.Error("pfDecode:", err)
				continue
			}

			if connAny, ok := l.conns.Load(idx); ok {
				conn := connAny.(*MedSocksConn)
				switch state {
				case pfStateNone:
					logger.Debug("dataIn conn.inCh <- data")
					conn.inCh <- data
				case pfStateEnd:
					logger.Debug("dataIn conn.Close()")
					conn.Close()
				}
			} else {
				switch state {
				case pfStateBegin:
					logger.Debug("dataIn l.newConnCh <- l.createConn(idx)")
					l.newConnCh <- l.createConn(idx)
				case pfStateNone:
					logger.Debug("dataIn resp end")
					dataOut(pfEncode(pfStateEnd, idx, nil))
				}
			}
		}
	}()

	return &l
}

func (l *MedSocksListener) Accept() (net.Conn, error) {
	conn := <-l.newConnCh
	if conn == nil {
		return nil, net.ErrClosed
	}
	return conn, nil
}

func (l *MedSocksListener) createConn(idx uint64) *MedSocksConn {
	ctx, cancel := context.WithCancel(l.ctx)
	conn := MedSocksConn{
		ctx:    ctx,
		cancel: cancel,
		inCh:   make(chan []byte, 0),
		outCh:  make(chan []byte, 0),
	}

	l.conns.Store(idx, &conn)

	l.wg.Add(1)
	go func() {
		logger.Debug("conn start")
		defer logger.Debug("conn end")
		defer l.dataOut(pfEncode(pfStateEnd, idx, nil))
		defer l.removeConn(idx)
		defer l.wg.Done()
		for {
			select {
			case data := <-conn.outCh:
				ok := l.dataOut(pfEncode(pfStateNone, idx, data))
				if !ok {
					return
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	return &conn
}

func (l *MedSocksListener) removeConn(idx uint64) {
	logger.Debug("removeConn1:", idx)
	if value, ok := l.conns.LoadAndDelete(idx); ok {
		logger.Debug("removeConn:", idx)
		conn := value.(*MedSocksConn)
		close(conn.inCh)
		conn.Close()
	}
}

func (l *MedSocksListener) Wait() {
	l.wg.Wait()
}

func (l *MedSocksListener) Close() error {
	logger.Debug("conn.Close")

	l.conns.Range(func(key, _ any) bool {
		l.removeConn(key.(uint64))
		return true
	})
	l.cancel()

	return nil
}

func (l *MedSocksListener) Addr() net.Addr {
	return &medAddr
}

type MedSocksConn struct {
	ctx      context.Context
	cancel   context.CancelFunc
	unread   []byte
	unreadMu sync.Mutex
	inCh     chan []byte
	outCh    chan []byte
}

func (conn *MedSocksConn) Read(b []byte) (n int, err error) {
	if len(conn.unread) > 0 {
		n = copy(b, conn.unread)
		conn.unread = conn.unread[n:]
		if len(conn.unread) == 0 {
			conn.unread = nil
		}
		return n, nil
	}

	buf := <-conn.inCh
	if len(buf) == 0 {
		conn.Close()
		return 0, io.EOF
	}

	n = copy(b, buf)
	conn.unread = append(conn.unread, buf[n:]...)

	return n, nil
}

func (conn *MedSocksConn) Write(b []byte) (n int, err error) {
	select {
	case <-conn.ctx.Done():
		n, err = 0, io.EOF
	case conn.outCh <- b:
		n, err = len(b), nil
	}
	return
}

func (conn *MedSocksConn) Close() error {
	conn.cancel()
	return nil
}

func (conn *MedSocksConn) LocalAddr() net.Addr                { return &medAddr }
func (conn *MedSocksConn) RemoteAddr() net.Addr               { return &medAddr }
func (conn *MedSocksConn) SetDeadline(t time.Time) error      { return nil }
func (conn *MedSocksConn) SetReadDeadline(t time.Time) error  { return nil }
func (conn *MedSocksConn) SetWriteDeadline(t time.Time) error { return nil }

var medAddr = net.UnixAddr{Net: "med", Name: "med"}
