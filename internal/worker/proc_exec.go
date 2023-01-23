package worker

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"sync"
	"syscall"

	"github.com/creack/pty"
	"github.com/djosix/med/internal/helper"
	"github.com/djosix/med/internal/logger"
	pb "github.com/djosix/med/internal/protobuf"
	"golang.org/x/term"
)

///////////////////////////////////////////////////////////////////////////

const (
	ProcExecInfoKindWinSize = 1
	ProcExecCtrlQuit        = "quit"
)

type ProcExecInfo struct {
	Kind int
	Data []byte
}

type ProcExecSpec struct {
	ARGV []string
	TTY  bool
}

///////////////////////////////////////////////////////////////////////////

type ClientExecProc struct {
	stdinReader helper.BreakableReader
	stdin       *os.File
	stdout      *os.File
	stderr      *os.File
	argv        []string
	tty         bool
}

func NewClientExecProc(argv []string, tty bool) *ClientExecProc {
	stdinReader, stdin := helper.GetBreakableStdin()

	return &ClientExecProc{
		stdin:  stdin,
		stdout: os.Stdout,
		stderr: os.Stderr,

		stdinReader: stdinReader,

		argv: argv,
		tty:  tty,
	}
}

func (p *ClientExecProc) Run(ctx ProcRunCtx) {
	ctx.PktOutCh <- &pb.Packet{
		TargetID: ctx.ProcID,
		Kind:     pb.PacketKind_PacketKindInfo,
		Data:     helper.MustEncode(&ProcExecSpec{ARGV: p.argv, TTY: p.tty}),
	}

	if p.tty {
		oldState, err := term.MakeRaw(int(p.stdin.Fd()))
		if err != nil {
			panic(err)
		}

		// Save logger outputs instead of writing to stdout
		loggerOutputBuf := bytes.NewBuffer([]byte{})
		loggerTarget := logger.SwapTarget(loggerOutputBuf)

		defer func() {
			_ = term.Restore(int(p.stdin.Fd()), oldState)

			// Write saved logger outputs to stdout
			logger.SwapTarget(loggerTarget)
			loggerOutputBuf.WriteTo(loggerTarget)
		}()
	}

	ctx1, cancel1 := context.WithCancel(ctx)
	wg := sync.WaitGroup{}

	if p.tty {
		// Handle SIGWINCH

		sigCh := make(chan os.Signal, 1)
		sigCh <- syscall.SIGWINCH
		signal.Notify(sigCh, syscall.SIGWINCH)
		defer func() { signal.Stop(sigCh); close(sigCh) }() // Cleanup signals when done.

		wg.Add(1)
		go func() {
			logger := logger.NewLogger("loop[SIGWINCH]")
			logger.Debug("start")

			defer wg.Done()
			defer ctx.Cancel()
			defer logger.Debug("done")

			for {
				select {
				case <-sigCh:
				case <-ctx1.Done():
					return
				}
				winSize, err := pty.GetsizeFull(p.stdin)
				if err != nil {
					logger.Error("cannot get winSize")
					continue
				}
				ctx.PktOutCh <- &pb.Packet{
					Kind: pb.PacketKind_PacketKindInfo,
					Data: helper.MustEncode(&ProcExecInfo{
						Kind: ProcExecInfoKindWinSize,
						Data: helper.MustEncode(winSize),
					}),
				}
			}
		}()
	}

	// Handle IO
	wg.Add(1)
	go func() {
		logger := logger.NewLogger("loop[ctx.PktInCh]")
		logger.Debug("start")

		defer wg.Done()
		defer cancel1()
		defer logger.Debug("done")

		for {
			var pkt *pb.Packet
			select {
			case pkt = <-ctx.PktInCh:
				if pkt == nil {
					return // ctx.PktInCh is closed
				}
			case <-ctx1.Done():
				return
			}

			switch pkt.Kind {
			case pb.PacketKind_PacketKindCtrl:
				switch string(pkt.Data) {
				case ProcExecCtrlQuit:
					logger.Debug("ProcExecCtrlQuit")
					return
				default:
					logger.Warn("unknown pkt.Data for PacketKind_PacketKindCtrl:", pkt.Data)
				}
			case pb.PacketKind_PacketKindData:
				p.stdout.Write(pkt.Data)
			}
		}
	}()

	wg.Add(1)
	go func() {
		logger := logger.NewLogger("loop[stdin.Read]")
		logger.Debug("start")

		defer wg.Done()
		defer cancel1()
		defer logger.Debug("done")

		buf := make([]byte, 1024)
		for {
			n, err := p.stdinReader.Read(buf)
			if err != nil || n == 0 {
				logger.Debugf("stdin: n=[%v] err=[%v]", n, err)
				return
			}
			pkt := &pb.Packet{
				Kind: pb.PacketKind_PacketKindData,
				Data: helper.Clone(buf[:n]),
			}
			select {
			case ctx.PktOutCh <- pkt:
			case <-ctx1.Done():
				return
			}
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		defer logger.Debug("p.stdinReader.BreakRead()")

		<-ctx1.Done()
		p.stdinReader.BreakRead()
	}()

	wg.Wait()
}

type ChanWriter struct {
	ctx context.Context
	ch  chan []byte
}

func NewChanWriter(ctx context.Context) *ChanWriter {
	return &ChanWriter{
		ctx: ctx,
		ch:  make(chan []byte, 1),
	}
}

func (w *ChanWriter) Write(p []byte) (n int, err error) {
	select {
	case w.ch <- p:
		return len(p), nil
	case <-w.ctx.Done():
		return 0, fmt.Errorf("broken")
	}
}

type ChanReader struct {
	ctx context.Context
	ch  chan []byte
	buf []byte
}

func NewChanReader(ctx context.Context) *ChanReader {
	return &ChanReader{
		ctx: ctx,
		ch:  make(chan []byte, 1),
		buf: []byte{},
	}
}

func (r *ChanReader) Read(p []byte) (n int, err error) {
	if len(r.buf) > 0 {
		n = copy(p, r.buf)
		r.buf = append([]byte{}, r.buf[n:]...)
		return n, nil
	}

	select {
	case buf := <-r.ch:
		n = copy(p, buf)
		buf = buf[n:]
		if len(buf) > 0 {
			r.buf = append(r.buf, buf...)
		}
		return n, nil
	case <-r.ctx.Done():
		return 0, fmt.Errorf("broken")
	}

}

///////////////////////////////////////////////////////////////////////////

type ServerExecProc struct{}

func NewServerExecProc() *ServerExecProc {
	return &ServerExecProc{}
}

func (p *ServerExecProc) Run(ctx ProcRunCtx) {
	logger := logger.NewLogger("ServerExecProc")
	logger.Debug("start")
	defer logger.Debug("done")

	// Get spec from client
	var spec ProcExecSpec
	select {
	case pkt := <-ctx.PktInCh:
		if pkt.Kind != pb.PacketKind_PacketKindInfo {
			logger.Error("the first packet.kind is not info")
			return
		}
		if err := helper.Decode(pkt.Data, &spec); err != nil {
			logger.Error("decode spec:", err)
			return
		}
		if len(spec.ARGV) == 0 {
			logger.Error("invalid argv:", spec.ARGV)
			return
		}
		logger.Debug("spec:", spec)
	case <-ctx.Done():
		return
	}

	cmd := exec.Command(spec.ARGV[0], spec.ARGV[1:]...)

	ctx1, cancel1 := context.WithCancel(ctx)
	defer cancel1()

	wg := sync.WaitGroup{}

	startWriter := func(file *os.File) chan<- []byte {
		ch := make(chan []byte, 1)

		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case buf := <-ch:
					if buf == nil {
						return // ch is closed
					}
					if _, err := file.Write(buf); err != nil {
						logger.Debug("file.Write:", err)
						return
					}
				case <-ctx1.Done():
					return
				}
			}
		}()

		return ch
	}

	startReader := func(file *os.File) <-chan []byte {
		ch := make(chan []byte, 1)
		r := helper.NewBreakableReader(file, 1024)

		wg.Add(1)
		go func() {
			defer wg.Done()
			<-ctx1.Done()
			r.BreakRead()
			r.Cancel()
			close(ch)
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()
			defer r.Cancel()
			defer close(ch)
			buf := [1024]byte{}
			for {
				n, err := r.Read(buf[:])
				if err != nil || n == 0 {
					return
				}
				select {
				case ch <- buf[:n]:
				case <-ctx1.Done():
					return
				}
			}
		}()

		return ch
	}

	startSender := func(fd byte, ch <-chan []byte) {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for {
				var buf []byte
				select {
				case buf = <-ch:
				case <-ctx1.Done():
					return
				}

				pkt := &pb.Packet{
					Kind: pb.PacketKind_PacketKindData,
					Data: append([]byte{fd}, buf...),
				}

				select {
				case ctx.PktOutCh <- pkt:
				case <-ctx1.Done():
					return
				}
			}
		}()
	}

	var inputCh chan<- []byte

	if spec.TTY {
		ptmx, err := pty.Start(cmd)
		if err != nil {
			logger.Error("pty.Start:", err)
			return
		}
		defer func() { _ = ptmx.Close() }()

		inputCh = startWriter(ptmx)
		startSender(1, startReader(ptmx))
	} else {
		var (
			err     error
			stdinW  *os.File
			stdoutR *os.File
			stderrR *os.File
		)

		cmd.Stdin, stdinW, err = os.Pipe()
		if err != nil {
			logger.Error("os.Pipe:", err)
			return
		}
		stdoutR, cmd.Stdout, err = os.Pipe()
		if err != nil {
			logger.Error("os.Pipe:", err)
			return
		}
		stderrR, cmd.Stdin, err = os.Pipe()
		if err != nil {
			logger.Error("os.Pipe:", err)
			return
		}

		if err := cmd.Start(); err != nil {
			logger.Error("cmd.Start:", err)
			return
		}

		inputCh = startWriter(stdinW)
		startSender(1, startReader(stdoutR))
		startSender(2, startReader(stderrR))
	}

	ptmxIn := make(chan []byte, 4)
	ptmxOut := make(chan []byte, 4)

	wg.Add(1)
	go func() {
		logger := logger.NewLogger("ProcessKiller")
		logger.Debug("start")
		defer wg.Done()
		defer logger.Debug("done")
		<-ctx1.Done()
		cmd.Process.Kill()
		cmd.Process.Wait()
	}()

	wg.Add(1)
	go func() {
		logger := logger.NewLogger("loop[ptmxIn]")
		logger.Debug("start")
		defer wg.Done()
		defer cancel1()
		defer logger.Debug("done")
		for {
			var buf []byte
			select {
			case buf = <-ptmxIn:
				if buf == nil {
					return
				}
			case <-ctx1.Done():
				return
			}

			if _, err := ptmx.Write(buf); err != nil {
				logger.Print("cannot ptmx.Write:", err)
				return
			}
		}
	}()

	wg.Add(1)
	go func() {
		logger := logger.NewLogger("loop[ptmx.Read]")
		logger.Debug("start")
		defer wg.Done()
		defer cancel1()
		defer close(ptmxOut)
		defer logger.Debug("done")
		buf := [1024]byte{}
		for {
			n, err := ptmx.Read(buf[:])
			if err != nil || n == 0 {
				logger.Debugf("ptmx.Read: n=[%v] err=[%v]", n, err)
				return
			}
			ptmxOut <- append([]byte{}, buf[:n]...)
		}
	}()

	wg.Add(1)
	go func() {
		logger := logger.NewLogger("loop[PktInCh]")
		logger.Debug("start")

		defer wg.Done()
		defer cancel1()
		defer close(ptmxIn)
		defer logger.Debug("done")

		for {
			var pkt *pb.Packet
			select {
			case pkt = <-ctx.PktInCh:
				if pkt == nil {
					return
				}
			case <-ctx1.Done():
				return
			}

			switch pkt.Kind {
			case pb.PacketKind_PacketKindData:
				select {
				case ptmxIn <- pkt.Data:
				case <-ctx1.Done():
					return
				}
			case pb.PacketKind_PacketKindInfo:
				logger.Debug("got MedMsg<Info>")
				info := ProcExecInfo{}
				if err := helper.Decode(pkt.Data, &info); err != nil {
					logger.Error("decode to info:", err)
					continue
				}
				switch info.Kind {
				case ProcExecInfoKindWinSize:
					winSize := pty.Winsize{}
					if err := helper.Decode(info.Data, &winSize); err != nil {
						logger.Error("decode to winSize:", err)
						continue
					}
					if err := pty.Setsize(ptmx, &winSize); err != nil {
						logger.Error("cannot set terminal size")
					}
				}
			}
		}
	}()

	wg.Add(1)
	go func() {
		logger := logger.NewLogger("loop[ptmxOut]")
		logger.Debug("start")

		defer wg.Done()
		defer cancel1()
		defer logger.Debug("done")

		for {
			var buf []byte
			select {
			case buf = <-ptmxOut:
				if buf == nil {
					return
				}
			case <-ctx1.Done():
				return
			}

			pkt := &pb.Packet{
				Kind: pb.PacketKind_PacketKindData,
				Data: buf,
			}

			select {
			case ctx.PktOutCh <- pkt:
			case <-ctx1.Done():
				return
			}
		}
	}()

	logger.Debug("wg.Wait()")
	wg.Wait()
	logger.Debug("wg.Wait() done")

	ctx.PktOutCh <- &pb.Packet{
		Kind: pb.PacketKind_PacketKindCtrl,
		Data: []byte(ProcExecCtrlQuit),
	}

	ctx.Cancel()
}
