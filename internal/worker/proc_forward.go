package worker

import (
	"bytes"
	"context"
	"encoding/binary"
	"io"
	"net"
	"sync"

	"github.com/djosix/med/internal/helper"
	"github.com/djosix/med/internal/logger"
	"github.com/djosix/med/internal/protobuf"
)

//
// Local Port Forwarding
//

type ForwardSpec struct {
	ListenEndpoint  string
	ConnectEndpoint string
}

// Client

type LocalPFProcClient struct {
	ProcInfo
	spec ForwardSpec
}

func NewLocalPFProcClient(spec ForwardSpec) *LocalPFProcClient {
	return &LocalPFProcClient{
		ProcInfo: NewProcInfo(ProcKind_LocalPF, ProcSide_Client),
		spec:     spec,
	}
}

func (p *LocalPFProcClient) Run(ctx *ProcRunCtx) {
	logger := logger.NewLogger(string(p.Kind()))

	// Send spec
	SendProcSpec(ctx, p.spec)

	dataIn := pfDefaultDataInFn(ctx)
	dataOut := pfDefaultDataOutFn(ctx)

	err := pfListen(ctx, p.spec.ListenEndpoint, dataIn, dataOut)
	if err != nil {
		logger.Error(err)
	}
}

// Server

type LocalPFProcServer struct {
	ProcInfo
}

func NewLocalPFProcServer() *LocalPFProcServer {
	return &LocalPFProcServer{
		ProcInfo: NewProcInfo(ProcKind_LocalPF, ProcSide_Server),
	}
}

func (p *LocalPFProcServer) Run(ctx *ProcRunCtx) {
	logger := logger.NewLogger(string(p.Kind()))

	// Get spec
	spec, err := RecvProcSpec[ForwardSpec](ctx)
	if err != nil {
		logger.Error(err)
		return
	}

	dataIn := pfDefaultDataInFn(ctx)
	dataOut := pfDefaultDataOutFn(ctx)

	err = pfConnect(ctx, spec.ConnectEndpoint, dataIn, dataOut)
	if err != nil {
		logger.Error(err)
	}
}

//
// Remote Port Forwarding
//

// Client

type RemotePFProcClient struct {
	ProcInfo
	spec ForwardSpec
}

func NewRemotePFProcClient(spec ForwardSpec) *RemotePFProcClient {
	return &RemotePFProcClient{
		ProcInfo: NewProcInfo(ProcKind_RemotePF, ProcSide_Client),
		spec:     spec,
	}
}

func (p *RemotePFProcClient) Run(ctx *ProcRunCtx) {
	logger := logger.NewLogger(string(p.Kind()))

	// Send spec
	SendProcSpec(ctx, p.spec)

	dataIn := pfDefaultDataInFn(ctx)
	dataOut := pfDefaultDataOutFn(ctx)

	err := pfConnect(ctx, p.spec.ConnectEndpoint, dataIn, dataOut)
	if err != nil {
		logger.Error(err)
	}
}

// Server

type RemotePFProcServer struct {
	ProcInfo
}

func NewRemotePFProcServer() *RemotePFProcServer {
	return &RemotePFProcServer{
		ProcInfo: NewProcInfo(ProcKind_RemotePF, ProcSide_Server),
	}
}

func (p *RemotePFProcServer) Run(ctx *ProcRunCtx) {
	logger := logger.NewLogger(string(p.Kind()))

	// Get spec
	spec, err := RecvProcSpec[ForwardSpec](ctx)
	if err != nil {
		logger.Error(err)
		return
	}

	dataIn := pfDefaultDataInFn(ctx)
	dataOut := pfDefaultDataOutFn(ctx)

	err = pfListen(ctx, spec.ListenEndpoint, dataIn, dataOut)
	if err != nil {
		logger.Error(err)
	}
}

//

func pfListen(
	ctx context.Context,
	endpoint string,
	dataIn func(context.Context) []byte,
	dataOut func([]byte) bool,
) error {
	logger := logger.NewLoggerf("pfListen[%v]", endpoint)

	listener, err := net.Listen(helper.SplitEndpoint(endpoint))
	if err != nil {
		return err
	}
	defer listener.Close()

	ctx, cancel := context.WithCancel(ctx)
	wg := sync.WaitGroup{}

	conns := map[uint64]chan []byte{}
	connsMu := sync.Mutex{}

	removeConn := func(idx uint64) {
		connsMu.Lock()
		defer connsMu.Unlock()

		if inCh, ok := conns[idx]; ok {
			close(inCh)
			delete(conns, idx)
		}
	}

	handleConn := func(idx uint64, conn net.Conn, inCh chan []byte) {
		logger := logger.NewLoggerf("conn[%v]", idx)
		logger.Debug("start")

		dataOut(pfEncode(pfStateBegin, idx, nil))

		cleanupOnce := sync.Once{}
		cleanup := func() {
			cleanupOnce.Do(func() {
				conn.Close()
				removeConn(idx)
				dataOut(pfEncode(pfStateEnd, idx, nil))
			})
		}

		wg.Add(1)
		go func() {
			defer wg.Done()
			defer cleanup()
			defer logger.Debug("pfReadLoop done")

			pfReadLoop(conn, func(buf []byte) bool {
				return dataOut(pfEncode(pfStateNone, idx, buf))
			})
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()
			defer cleanup()
			defer logger.Debug("pfWriteLoop done")

			pfWriteLoop(conn, func() []byte {
				return <-inCh
			})
		}()
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		defer cancel()

		for i := 0; ; i++ {
			idx := uint64(i)

			conn, err := listener.Accept()
			if err != nil {
				logger.Error(err)
				return
			}
			logger.Debug("accept:", conn.RemoteAddr())

			inCh := make(chan []byte)
			connsMu.Lock()
			conns[idx] = inCh
			connsMu.Unlock()

			wg.Add(1)
			go func() {
				defer wg.Done()
				handleConn(idx, conn, inCh)
			}()
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		defer listener.Close()

		for {
			data := dataIn(ctx)
			if data == nil {
				logger.Debug("dataIn closed")
				return
			}

			state, idx, buf, err := pfDecode(data)
			if err != nil {
				logger.Error(err)
				continue
			}

			connsMu.Lock()
			if inCh, ok := conns[idx]; ok {
				switch state {
				case pfStateNone:
					inCh <- buf
				case pfStateEnd:
					go removeConn(idx)
				}
			} else {
				switch state {
				case pfStateNone:
					ok := dataOut(pfEncode(pfStateEnd, idx, nil))
					if !ok {
						return
					}
				}
			}
			connsMu.Unlock()
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		<-ctx.Done()
		listener.Close()
	}()

	wg.Wait()

	return nil
}

func pfConnect(
	ctx context.Context,
	endpoint string,
	dataIn func(context.Context) []byte,
	dataOut func([]byte) bool,
) error {
	// ctx, cancel := context.WithCancel(ctx)
	wg := sync.WaitGroup{}

	type conn struct {
		conn net.Conn
		inCh chan []byte
	}

	conns := map[uint64]chan<- []byte{}
	connsMu := sync.Mutex{}

	removeConn := func(idx uint64) {
		connsMu.Lock()
		defer connsMu.Unlock()

		if ch, ok := conns[idx]; ok {
			close(ch)
			delete(conns, idx)
		}
	}

	connect := func(idx uint64, ch chan []byte) {
		conn, err := net.Dial(helper.SplitEndpoint(endpoint))
		if err != nil {
			dataOut(pfEncode(pfStateEnd, idx, nil))
			logger.Error(err)
			return
		}

		dataOut(pfEncode(pfStateBegin, idx, nil))

		cleanupOnce := sync.Once{}
		cleanup := func() {
			cleanupOnce.Do(func() {
				conn.Close()
				removeConn(idx)
				dataOut(pfEncode(pfStateEnd, idx, nil))
			})
		}

		wg.Add(1)
		go func() {
			defer wg.Done()
			defer cleanup()

			pfReadLoop(conn, func(data []byte) bool {
				return dataOut(pfEncode(pfStateNone, idx, data))
			})
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()
			defer cleanup()

			pfWriteLoop(conn, func() []byte {
				return <-ch
			})
		}()
	}

	wg.Add(1)
	go func() {
		defer wg.Done()

		for {
			data := dataIn(ctx)
			if data == nil {
				logger.Debug("dataIn closed")
				return
			}

			state, idx, buf, err := pfDecode(data)
			if err != nil {
				logger.Error(err)
				continue
			}

			connsMu.Lock()
			if inCh, ok := conns[idx]; ok {
				switch state {
				case pfStateNone:
					inCh <- buf
				case pfStateEnd:
					go removeConn(idx)
				}
			} else {
				switch state {
				case pfStateBegin:
					inCh := make(chan []byte)
					conns[idx] = inCh

					wg.Add(1)
					go func() {
						defer wg.Done()
						connect(idx, inCh)
					}()
				case pfStateNone:
					dataOut(pfEncode(pfStateEnd, idx, nil))
				}
			}
			connsMu.Unlock()
		}
	}()

	wg.Wait()

	return nil
}

func pfReadLoop(r io.Reader, out func([]byte) bool) {
	buf := make([]byte, 4096)

	for {
		n, err := r.Read(buf)
		if n == 0 || err != nil {
			break
		}

		ok := out(buf[:n])
		if !ok {
			return
		}
	}
}

func pfWriteLoop(w io.Writer, in func() []byte) {
	for {
		buf := in()
		if buf == nil {
			break
		}

		n, err := w.Write(buf)
		if n != len(buf) || err != nil {
			break
		}
	}
}

// start: new=true idx=* buf=.*
// data: new=false idx=* buf=.+
// end: new=false idx=* buf=

type pfState byte

const (
	pfStateNone  pfState = 0
	pfStateBegin pfState = 1
	pfStateEnd   pfState = 2
)

func pfEncode(state pfState, idx uint64, buf []byte) []byte {
	if buf == nil {
		buf = []byte{}
	}

	data := []byte{byte(state)}
	data = binary.AppendUvarint(data, idx)
	data = append(data, buf...)

	return data
}

func pfDecode(data []byte) (state pfState, idx uint64, buf []byte, err error) {
	dataBuf := bytes.NewBuffer(data)

	b, err := dataBuf.ReadByte()
	if err != nil {
		return
	}

	state = pfState(b)

	idx, err = binary.ReadUvarint(dataBuf)
	if err != nil {
		return
	}

	buf = dataBuf.Bytes()
	return
}

type pfDataInFn = func(ctx context.Context) []byte

func pfDefaultDataInFn(ctx *ProcRunCtx) pfDataInFn {
	return func(c context.Context) []byte {
		for {
			pkt := ctx.InputPacketWithDone(c.Done())
			if pkt == nil {
				return nil
			}
			if pkt.Kind == protobuf.PacketKind_PacketKindData {
				// logger.Debug("dataIn:", pkt.Data)
				return pkt.Data
			}
		}
	}
}

type pfDataOutFn = func(data []byte) bool

func pfDefaultDataOutFn(ctx *ProcRunCtx) pfDataOutFn {
	return func(data []byte) bool {
		// logger.Debug("dataOut:", data)
		pkt := helper.NewDataPacket(data)
		return ctx.OutputPacket(pkt)
	}
}
