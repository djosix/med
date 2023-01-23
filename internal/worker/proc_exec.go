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
	log "github.com/djosix/med/internal/logger"
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
	logger := log.NewLogger("ClientExecProc")

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
		loggerTarget := log.SwapTarget(loggerOutputBuf)

		defer func() {
			_ = term.Restore(int(p.stdin.Fd()), oldState)

			// Write saved logger outputs to stdout
			log.SwapTarget(loggerTarget)
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
				if len(pkt.Data) > 1 {
					lastIdx := len(pkt.Data) - 1
					switch data, fd := pkt.Data[:lastIdx], pkt.Data[lastIdx]; fd {
					case 1:
						p.stdout.Write(data)
					case 2:
						p.stderr.Write(data)
					default:
						logger.Warn("invalid output fd:", fd)
					}
				}
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

	ctx1, cancel1 := context.WithCancel(ctx)
	defer cancel1()

	wg := sync.WaitGroup{}

	startReader := func(file *os.File) <-chan []byte {
		logger := logger.NewLogger(fmt.Sprintf("reader[%v]", file.Fd()))
		ch := make(chan []byte, 1)

		wg.Add(1)
		go func() {
			defer wg.Done()
			defer cancel1()

			logger.Debug("start")
			defer logger.Debug("done")

			buf := [1024]byte{}
			for {
				logger.Debug("file.Read(buf[:])")
				n, err := file.Read(buf[:])
				logger.Debug("file.Read(buf[:]) done")
				if err != nil || n == 0 {
					return
				}

				select {
				case ch <- helper.Clone(buf[:n]):
				case <-ctx1.Done():
					return
				}
			}
		}()

		return ch
	}

	startSender := func(fd byte, ch <-chan []byte) {
		logger := logger.NewLogger(fmt.Sprintf("sender[%v]", fd))

		wg.Add(1)
		go func() {
			defer wg.Done()
			defer cancel1()

			logger.Debug("start")
			defer logger.Debug("done")

			for {
				var buf []byte
				select {
				case buf = <-ch:
					if buf == nil {
						return // ch closed
					}
				case <-ctx1.Done():
					return
				}

				logger.Debug("send", fd, buf)
				pkt := &pb.Packet{
					Kind: pb.PacketKind_PacketKindData,
					Data: append(buf, fd),
				}

				select {
				case ctx.PktOutCh <- pkt:
				case <-ctx1.Done():
					return
				}
			}
		}()
	}

	cmd := exec.Command(spec.ARGV[0], spec.ARGV[1:]...)
	var inputFile *os.File

	if spec.TTY {
		ptmx, err := pty.Start(cmd)
		if err != nil {
			logger.Error("pty.Start:", err)
			return
		}
		defer func() { _ = ptmx.Close() }()

		inputFile = ptmx
		startSender(1, startReader(ptmx))
	} else {
		var (
			rPipes, wPipes [3]*os.File
			err            error
		)
		for i := range rPipes {
			if rPipes[i], wPipes[i], err = os.Pipe(); err != nil {
				logger.Error("os.Pipe:", err)
				return
			}
		}
		cmd.Stdin = rPipes[0]
		cmd.Stdout = wPipes[1]
		cmd.Stderr = wPipes[2]
		cmd.SysProcAttr = &syscall.SysProcAttr{
			Setsid: true,
		}
		if err = cmd.Start(); err != nil {
			logger.Error("cmd.Start:", err)
			return
		}
		inputFile = wPipes[0]
		wPipes[1].Close()
		wPipes[2].Close()
		startSender(1, startReader(rPipes[1]))
		startSender(2, startReader(rPipes[2]))
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := cmd.Wait(); err != nil {
			logger.Error("cmd.Wait():", err)
		}
		cancel1()
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		logger := logger.NewLogger("killer")
		logger.Debug("start")
		defer logger.Debug("done")

		<-ctx1.Done()
		if err := cmd.Process.Kill(); err != nil {
			logger.Error("cmd.Process.Kill():", err)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		defer cancel1()
		defer inputFile.Close()

		logger := logger.NewLogger("loop[PktInCh]")
		logger.Debug("start")
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
				logger.Debug("pkt:", pkt)
				if _, err := inputFile.Write(pkt.Data); err != nil {
					logger.Debug("inputFile.Write:", err)
					return
				}
			case pb.PacketKind_PacketKindInfo:
				info := ProcExecInfo{}
				if err := helper.Decode(pkt.Data, &info); err != nil {
					logger.Error("decode to info:", err)
					continue
				}
				switch info.Kind {
				case ProcExecInfoKindWinSize:
					if spec.TTY {
						winSize := pty.Winsize{}
						if err := helper.Decode(info.Data, &winSize); err != nil {
							logger.Error("decode to winSize:", err)
							continue
						}
						logger.Debug("set winsize:", winSize)
						if err := pty.Setsize(inputFile, &winSize); err != nil {
							logger.Error("set winsize:", err)
						}
					}
				}
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
