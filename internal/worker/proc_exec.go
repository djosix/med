package worker

import (
	"bytes"
	"context"
	"io"
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

type ExecInfoKind byte

const (
	ExecInfoKind_WinSize ExecInfoKind = 1
)

type ExecInfo struct {
	Kind ExecInfoKind
	Data []byte
}

type ExecSpec struct {
	ARGV []string
	TTY  bool
}

// Client

type ExecProcClient struct {
	ProcInfo
	ExecSpec
	stdin      *os.File
	stdout     *os.File
	stderr     *os.File
	reader     io.Reader
	cancelRead func()
}

func NewExecProcClient(spec ExecSpec) *ExecProcClient {
	reader, cancelRead, stdin := helper.GetCancelStdin()

	return &ExecProcClient{
		ProcInfo:   NewProcInfo(ProcKind_Exec, ProcSide_Client),
		ExecSpec:   spec,
		stdin:      stdin,
		stdout:     os.Stdout,
		stderr:     os.Stderr,
		reader:     reader,
		cancelRead: cancelRead,
	}
}

func (p *ExecProcClient) Run(ctx *ProcRunCtx) {
	logger := logger.NewLogger(string(p.Kind()))
	logger.Debug("start")
	defer logger.Debug("done")

	SendProcSpec(ctx, p.ExecSpec)

	if p.TTY {
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

	if p.TTY {
		// Handle SIGWINCH

		sigCh := make(chan os.Signal, 1)
		sigCh <- syscall.SIGWINCH
		signal.Notify(sigCh, syscall.SIGWINCH)
		defer func() { signal.Stop(sigCh); close(sigCh) }()

		wg.Add(1)
		go func() {
			defer wg.Done()
			defer ctx.Cancel()

			logger := logger.NewLogger("loop[SIGWINCH]")
			logger.Debug("start")
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

				ctx.OutputPacket(&pb.Packet{
					Kind: pb.PacketKind_PacketKindInfo,
					Data: helper.MustEncode(&ExecInfo{
						Kind: ExecInfoKind_WinSize,
						Data: helper.MustEncode(winSize),
					}),
				})
			}
		}()
	}

	// Handle IO
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer cancel1()

		logger := logger.NewLogger("loop[input]")
		logger.Debug("start")
		defer logger.Debug("done")

		for {
			pkt := ctx.InputPacket()
			if pkt == nil {
				return
			}

			switch pkt.Kind {
			case pb.PacketKind_PacketKindCtrl:
				if pb.IsPacketWithCtrlKind(pkt, pb.PacketCtrl_Exit) {
					logger.Debug("PacketCtrl_Exit")
					return
				} else {
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
		logger := logger.NewLogger("loop[stdin]")
		logger.Debug("start")

		defer wg.Done()
		defer cancel1()
		defer logger.Debug("done")

		buf := make([]byte, 1024)
		for {
			n, err := p.reader.Read(buf)
			if err != nil || n == 0 {
				logger.Debugf("stdin: n=[%v] err=[%v]", n, err)
				return
			}
			ctx.OutputPacket(&pb.Packet{
				Kind: pb.PacketKind_PacketKindData,
				Data: helper.Clone(buf[:n]),
			})
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		defer logger.Debug("BreakRead")

		<-ctx1.Done()
		p.cancelRead()
	}()

	wg.Wait()
}

// Server

type ExecProcServer struct {
	ProcInfo
}

func NewExecProcServer() *ExecProcServer {
	return &ExecProcServer{
		ProcInfo: NewProcInfo(ProcKind_Exec, ProcSide_Server),
	}
}

func (p *ExecProcServer) Run(ctx *ProcRunCtx) {
	logger := logger.NewLogger(string(p.Kind()))
	logger.Debug("start")
	defer logger.Debug("done")

	// Get spec from client
	spec, err := RecvProcSpec[ExecSpec](ctx)
	if err != nil {
		logger.Error("spec:", err)
		return
	}
	logger.Debug("spec:", spec)

	ctx1, cancel1 := context.WithCancel(ctx)
	defer cancel1()

	wg := sync.WaitGroup{}

	startReader := func(file *os.File) <-chan []byte {
		logger := logger.NewLoggerf("reader[fd=%v]", file.Fd())
		ch := make(chan []byte)

		wg.Add(1)
		go func() {
			defer wg.Done()
			defer cancel1()
			defer close(ch)

			logger.Debug("start")
			defer logger.Debug("done")

			buf := [1024]byte{}
			for {
				n, err := file.Read(buf[:])
				if err != nil || n == 0 {
					logger.Debugf("read: %v", err)
					return
				}

				ch <- helper.Clone(buf[:n])
			}
		}()

		return ch
	}

	startSender := func(fd byte, ch <-chan []byte) {
		logger := logger.NewLoggerf("sender[fd=%v]", fd)

		wg.Add(1)
		go func() {
			defer wg.Done()
			defer cancel1()

			logger.Debug("start")
			defer logger.Debug("done")

			for {
				buf := <-ch
				if buf == nil {
					return // channel is closed
				}

				// logger.Debugf("send %#v", string(buf))
				ctx.OutputPacket(&pb.Packet{
					Kind: pb.PacketKind_PacketKindData,
					Data: append(buf, fd), // the last byte represents the output fd
				})
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
		cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
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

		err := cmd.Wait()

		if err != nil {
			logger.Warn("cmd.Wait:", err)
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

		err := cmd.Process.Kill() // wait in other goroutine
		if err != nil {
			logger.Warn("kill:", err)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		defer cancel1()
		defer inputFile.Close()

		logger := logger.NewLogger("loop[input]")
		logger.Debug("start")
		defer logger.Debug("done")

	forLoop:
		for {
			pkt := ctx.InputPacketWithDone(ctx1.Done())
			if pkt == nil {
				return
			}

			// logger.Debugf("packet [%v]", pkt)

			switch pkt.Kind {
			case pb.PacketKind_PacketKindData:
				if _, err := inputFile.Write(pkt.Data); err != nil {
					logger.Debug("inputFile.Write:", err)
					return
				}
			case pb.PacketKind_PacketKindInfo:
				info := ExecInfo{}
				if err := helper.Decode(pkt.Data, &info); err != nil {
					logger.Error("decode to info:", err)
					continue forLoop
				}
				switch info.Kind {
				case ExecInfoKind_WinSize:
					if spec.TTY {
						winSize := pty.Winsize{}
						if err := helper.Decode(info.Data, &winSize); err != nil {
							logger.Error("decode to winSize:", err)
							continue forLoop
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

	logger.Debug("wait")
	wg.Wait()
	logger.Debug("wait done")

	ctx.OutputPacket(pb.NewCtrlPacket(ctx.ProcID, pb.PacketCtrl_Exit))
}
