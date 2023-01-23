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
	pb "github.com/djosix/med/internal/protobuf"
	"golang.org/x/term"
)

///////////////////////////////////////////////////////////////////////////

const (
	ExecProcCtrlKindWinSize = 1
)

type ExecProcCtrl struct {
	Kind int
	Data []byte
}

///////////////////////////////////////////////////////////////////////////

type ClientExecProc struct {
	stdin  *os.File
	stdout *os.File
	stderr *os.File
	tty    bool
}

func NewClientExecProc() *ClientExecProc {
	return &ClientExecProc{
		stdin:  os.Stdin,
		stdout: os.Stdout,
		stderr: os.Stderr,
		tty:    false,
	}
}

func (p *ClientExecProc) Run(ctx ProcRunCtx) {
	if p.tty {
		oldState, err := term.MakeRaw(int(p.stdin.Fd()))
		if err != nil {
			panic(err)
		}
		defer func() {
			_ = term.Restore(int(p.stdin.Fd()), oldState)
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
			defer wg.Done()
			defer ctx.Cancel()
			defer fmt.Println("done SIGWINCH loop")
			for {
				select {
				case <-sigCh:
				case <-localCtx.Done():
					return
				}
				winSize, err := pty.GetsizeFull(p.stdin)
				if err != nil {
					fmt.Println("cannot get winSize")
					continue
				}
				data, err := helper.Encode(winSize)
				if err != nil {
					fmt.Println("cannot encode winSize")
					continue
				}
				ctrl := ExecProcCtrl{
					Kind: ExecProcCtrlKindWinSize,
					Data: data,
				}
				data, err = helper.Encode(ctrl)
				if err != nil {
					fmt.Println("cannot encode ctrl")
					continue
				}
				ctx.MsgOutCh <- &pb.MedMsg{
					Type:    pb.MedMsgType_MedMsgTypeControl,
					Content: data,
				}
			}
		}()
	}

	// Handle IO
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer localCancel()
		defer fmt.Println("done msgInCh loop")
		for {
			var msg *pb.MedMsg
			select {
			case msg = <-ctx.MsgInCh:
				if msg == nil {
					return // ctx.MsgInCh is closed
				}
			case <-localCtx.Done():
				return
			}

			switch msg.Type {
			case pb.MedMsgType_MedMsgTypeControl:
				if bytes.Equal(msg.Content, []byte("end")) {
					fmt.Println("[end]")
					return
				} else {
					fmt.Println("unknown MedMsgType_MedMsgTypeControl:", msg.Content)
				}
			case pb.MedMsgType_MedMsgTypeData:
				p.stdout.Write(msg.Content)
			}
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		defer localCancel()
		defer fmt.Println("done stdin loop")
		buf := make([]byte, 1024)
		for {
			n, err := helper.BreakableStdin.Read(buf)
			if err != nil || n == 0 {
				fmt.Println("stdin:", err)
				return
			}
			msg := &pb.MedMsg{
				Type:    pb.MedMsgType_MedMsgTypeData,
				Content: append([]byte{}, buf[:n]...),
			}
			select {
			case ctx.MsgOutCh <- msg:
			case <-localCtx.Done():
				return
			}

		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		<-localCtx.Done()
		helper.BreakableStdin.BreakRead()
	}()

	fmt.Println("start wg.Wait()")
	wg.Wait()
	fmt.Println("done wg.Wait()")

	ctx.Loop.Cancel()
}

///////////////////////////////////////////////////////////////////////////

type ServerExecProc struct{}

func NewServerExecProc() *ServerExecProc {
	return &ServerExecProc{}
}

func (p *ServerExecProc) Run(ctx ProcRunCtx) {
	c := exec.Command("bash")

	ptmx, err := pty.Start(c)
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	defer func() { _ = ptmx.Close() }()

	localCtx, localCancel := context.WithCancel(ctx)
	ptmxIn := make(chan []byte, 4)
	ptmxOut := make(chan []byte, 4)

	wg := sync.WaitGroup{}

	wg.Add(1)
	go func() {
		defer wg.Done()
		defer fmt.Println("done kill")
		<-localCtx.Done()
		c.Process.Kill()
		c.Process.Wait()
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		defer localCancel()
		defer fmt.Println("done ptmxIn loop")
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
				fmt.Println("cannot ptmx.Write:", err)
				return
			}
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		defer localCancel()
		defer close(ptmxOut)
		defer fmt.Println("done ptmx.Read loop")
		buf := [1024]byte{}
		for {
			n, err := ptmx.Read(buf[:])
			if err != nil {
				fmt.Println("cannot ptmx.Read:", err)
				return
			}
			if n == 0 {
				fmt.Println("empty from ptmx.Read")
				return
			}
			ptmxOut <- append([]byte{}, buf[:n]...)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		defer localCancel()
		defer close(ptmxIn)
		defer fmt.Println("done ctx.MsgInCh loop")

		for {
			var msg *pb.MedMsg
			select {
			case msg = <-ctx.MsgInCh:
				if msg == nil {
					return
				}
			case <-localCtx.Done():
				return
			}

			switch msg.Type {
			case pb.MedMsgType_MedMsgTypeData:
				fmt.Println("MedMsgType_MedMsgTypeData")
				select {
				case ptmxIn <- msg.Content:
				case <-localCtx.Done():
					return
				}
			case pb.MedMsgType_MedMsgTypeControl:
				fmt.Println("MedMsgType_MedMsgTypeControl")
				ctrl := ExecProcCtrl{}
				if err := helper.Decode(msg.Content, &ctrl); err != nil {
					fmt.Println("error:", err)
					continue
				}
				switch ctrl.Kind {
				case ExecProcCtrlKindWinSize:
					winSize := pty.Winsize{}
					if err := helper.Decode(ctrl.Data, &winSize); err != nil {
						fmt.Println("error:", err)
						continue
					}
					if err := pty.Setsize(ptmx, &winSize); err != nil {
						fmt.Println("cannot set terminal size")
					}
				}
			}
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		defer localCancel()
		defer fmt.Println("done ptmxOut loop")
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

			msg := &pb.MedMsg{
				Type:    pb.MedMsgType_MedMsgTypeData,
				Content: buf,
			}

			select {
			case ctx.MsgOutCh <- msg:
			case <-localCtx.Done():
				return
			}
		}
	}()

	fmt.Println("start wg.Wait()")
	wg.Wait()
	fmt.Println("done wg.Wait()")

	ctx.MsgOutCh <- &pb.MedMsg{
		Type:    pb.MedMsgType_MedMsgTypeControl,
		Content: []byte("end"),
	}

	ctx.Cancel()
}
