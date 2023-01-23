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
	// ctx.MsgOutCh <- &pb.MedMsg{
	// }

	if p.tty {
		oldState, err := term.MakeRaw(int(p.stdin.Fd()))
		if err != nil {
			panic(err)
		}
		tempBuf := bytes.NewBuffer([]byte{})
		oldLoggerTarget := logger.SwapTarget(tempBuf)
		defer func() {
			_ = term.Restore(int(p.stdin.Fd()), oldState)
			logger.SwapTarget(oldLoggerTarget)
			tempBuf.WriteTo(oldLoggerTarget)
		}()
	}

	localCtx, localCancel := context.WithCancel(ctx)

	wg := sync.WaitGroup{}

	// Handle SIGWINCH
	if p.tty {
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
				case <-localCtx.Done():
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
		logger := logger.NewLogger("loop[ctx.MsgInCh]")
		logger.Debug("start")

		defer wg.Done()
		defer localCancel()
		defer logger.Debug("done")

		for {
			var pkt *pb.Packet
			select {
			case pkt = <-ctx.PktInCh:
				if pkt == nil {
					return // ctx.MsgInCh is closed
				}
			case <-localCtx.Done():
				return
			}

			switch pkt.Kind {
			case pb.PacketKind_PacketKindCtrl:
				if bytes.Equal(pkt.Data, []byte(ProcExecCtrlQuit)) {
					return
				} else {
					panic(fmt.Sprintln("unknown MedMsgType_MedMsgTypeControl:", pkt.Data))
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
		defer localCancel()
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
			case <-localCtx.Done():
				return
			}
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		defer logger.Debug("p.stdinReader.BreakRead()")

		<-localCtx.Done()
		p.stdinReader.BreakRead()
	}()

	wg.Wait()

	ctx.Loop.Stop()
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

	c := exec.Command("bash")

	ptmx, err := pty.Start(c)
	if err != nil {
		logger.Error("pty.Start:", err)
		return
	}
	defer func() { _ = ptmx.Close() }()

	localCtx, localCancel := context.WithCancel(ctx)
	ptmxIn := make(chan []byte, 4)
	ptmxOut := make(chan []byte, 4)

	wg := sync.WaitGroup{}

	wg.Add(1)
	go func() {
		logger := logger.NewLogger("ProcessKiller")
		logger.Debug("start")
		defer wg.Done()
		defer logger.Debug("done")
		<-localCtx.Done()
		c.Process.Kill()
		c.Process.Wait()
	}()

	wg.Add(1)
	go func() {
		logger := logger.NewLogger("loop[ptmxIn]")
		logger.Debug("start")
		defer wg.Done()
		defer localCancel()
		defer logger.Debug("done")
		for {
			var buf []byte
			select {
			case buf = <-ptmxIn:
				if buf == nil {
					return
				}
			case <-localCtx.Done():
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
		defer localCancel()
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
		logger := logger.NewLogger("loop[MsgInCh]")
		logger.Debug("start")

		defer wg.Done()
		defer localCancel()
		defer close(ptmxIn)
		defer logger.Debug("done")

		for {
			var pkt *pb.Packet
			select {
			case pkt = <-ctx.PktInCh:
				if pkt == nil {
					return
				}
			case <-localCtx.Done():
				return
			}

			switch pkt.Kind {
			case pb.PacketKind_PacketKindData:
				select {
				case ptmxIn <- pkt.Data:
				case <-localCtx.Done():
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
		defer localCancel()
		defer logger.Debug("done")

		for {
			var buf []byte
			select {
			case buf = <-ptmxOut:
				if buf == nil {
					return
				}
			case <-localCtx.Done():
				return
			}

			pkt := &pb.Packet{
				Kind: pb.PacketKind_PacketKindData,
				Data: buf,
			}

			select {
			case ctx.PktOutCh <- pkt:
			case <-localCtx.Done():
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
