package worker

import (
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/djosix/med/internal/helper"
	"github.com/djosix/med/internal/logger"
	pb "github.com/djosix/med/internal/protobuf"
)

type MainProcMsgKind string

const (
	MainProcMsgKind_Start  MainProcMsgKind = "Start"
	MainProcMsgKind_Remove MainProcMsgKind = "Remove"
	MainProcMsgKind_Exit   MainProcMsgKind = "Exit"
)

type MainProcMsg struct {
	Kind  MainProcMsgKind
	SeqNo uint32
	Data  []byte
}

type MainProcMsg_Start struct {
	ProcID   uint32
	ProcKind ProcKind
	Error    string // response
}

type MainProcMsg_Remove struct {
	ProcID uint32
	Error  string // response
}

// Client

type MainSpec struct {
	ExitWhenNoProc bool
}

type mainAnyProcKindAndSpec struct {
	kind ProcKind
	spec any
}

type MainProcClient struct {
	ProcInfo
	MainSpec
	newProcCh chan mainAnyProcKindAndSpec
}

func NewMainProcClient(spec MainSpec) *MainProcClient {
	return &MainProcClient{
		ProcInfo:  NewProcInfo(ProcKind_Main, ProcSide_Client),
		MainSpec:  spec,
		newProcCh: make(chan mainAnyProcKindAndSpec),
	}
}

func (p *MainProcClient) Run(ctx *ProcRunCtx) {
	logger := logger.NewLogger("MainProcClient")
	logger.Debug("start")
	defer logger.Debug("done")

	SendProcSpec(ctx, p.MainSpec)

	var seqNo uint32 = 0
	var procCount int32 = 0

	type pendingProc struct {
		procID uint32
		handle func(bool) <-chan struct{}
	}
	pendingProcs := map[uint32]pendingProc{}

	startProc := func(procKind ProcKind, spec any) {
		proc, err := CreateProcClient(procKind, spec)
		if err != nil {
			logger.Error("cannot create proc:", err)
			return
		}

		// Start client proc after server proc is created
		procID, handle := ctx.Loop.StartLater(proc)
		seqNo := atomic.AddUint32(&seqNo, 1)
		pendingProcs[seqNo] = pendingProc{procID: procID, handle: handle}

		// Notify server to create a proc
		ctx.PktOutCh <- helper.NewDataPacket(helper.MustEncode(&MainProcMsg{
			Kind: MainProcMsgKind_Start, SeqNo: seqNo,
			Data: helper.MustEncode(&MainProcMsg_Start{ProcKind: procKind, ProcID: procID}),
		}))

		// go func() {
		// 	time.Sleep(10 * time.Second)
		// 	delete(pendingProcs, seqNo)
		// }()
	}

	removeProc := func(procID uint32) {
		seqNo := atomic.AddUint32(&seqNo, 1)
		ctx.PktOutCh <- helper.NewDataPacket(helper.MustEncode(&MainProcMsg{
			Kind: MainProcMsgKind_Remove, SeqNo: seqNo,
			Data: helper.MustEncode(&MainProcMsg_Remove{ProcID: procID}),
		}))
	}
	_ = removeProc

	handleMessage := func(msg *MainProcMsg) {
		switch msg.Kind {
		case MainProcMsgKind_Start:
			// Handle server proc created
			data, err := helper.DecodeAs[MainProcMsg_Start](msg.Data)
			if err != nil {
				logger.Error("decode start error:", err)
				return
			}

			pp, exists := pendingProcs[msg.SeqNo]
			if !exists {
				logger.Error("no proc to start")
				return
			}
			delete(pendingProcs, msg.SeqNo)

			if data.Error != "" {
				logger.Errorf("start proc[%v] remote: %v", pp.procID, data.Error)
				pp.handle(false) // cancel start
			} else {
				logger.Debugf("start proc[%v]", pp.procID)
				doneCh := pp.handle(true)

				atomic.AddInt32(&procCount, 1)
				go func() {
					<-doneCh
					if atomic.AddInt32(&procCount, -1) == 0 && p.ExitWhenNoProc {
						ctx.Cancel()
					}
				}()
			}

		case MainProcMsgKind_Remove:
			// Handle server proc removed
			data, err := helper.DecodeAs[MainProcMsg_Remove](msg.Data)
			if err != nil {
				logger.Error("decode remove:", err)
				return
			}

			if data.Error != "" {
				logger.Error("remove:", data.Error)
			}
		}
	}

	wg := sync.WaitGroup{}

	// Handle p.StartProc calls
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case ks := <-p.newProcCh:
				startProc(ks.kind, ks.spec)
			case <-ctx.Done():
				return
			}
		}
	}()

	// Main loop
	wg.Add(1)
	go func() {
		defer wg.Done()
	forLoop:
		for {
			var pkt *pb.Packet
			select {
			case pkt = <-ctx.PktInCh:
				if pkt == nil {
					return
				}
			case <-ctx.Done():
				return
			}

			logger.Debugf("receive: [%v]", pkt)

			switch pkt.Kind {
			case pb.PacketKind_PacketKindData:
				msg, err := helper.DecodeAs[MainProcMsg](pkt.Data)
				if err != nil {
					logger.Error("decode as MainProcMsg:", err)
					continue forLoop
				}
				handleMessage(&msg)
			default:
				logger.Error("invalid packet kind:", pkt.Kind)
			}
		}
	}()

	wg.Wait()
}

func (p *MainProcClient) StartProc(kind ProcKind, spec any) {
	go func() {
		p.newProcCh <- mainAnyProcKindAndSpec{kind: kind, spec: spec}
	}()
}

// Server

type MainProcServer struct {
	ProcInfo
}

func NewMainProcServer() *MainProcServer {
	return &MainProcServer{
		ProcInfo: NewProcInfo(ProcKind_Main, ProcSide_Server),
	}
}

func (p *MainProcServer) Run(ctx *ProcRunCtx) {
	logger := logger.NewLogger("ServerMainProc")
	logger.Debug("start")
	defer logger.Debug("done")

	spec, err := RecvProcSpec[MainSpec](ctx)
	if err != nil {
		logger.Error("cannot decode MainSpec:", err)
		return
	}

	var procCount int32 = 0

	handleStart := func(data *MainProcMsg_Start) error {
		// Create proc by kind
		proc, err := CreateProcServer(data.ProcKind)
		if err != nil {
			return fmt.Errorf("create proc: %w", err)
		}

		startProcID, startHandle := ctx.Loop.StartLater(proc)

		if startProcID != data.ProcID {
			startHandle(false)
			return fmt.Errorf("invalid new procID: required=%v got=%v", data.ProcID, startProcID)
		}

		logger.Debugf("start proc[%v] kind=%v", startProcID, data.ProcKind)
		doneCh := startHandle(true)

		// Subscribe proc exit
		atomic.AddInt32(&procCount, 1)
		go func() {
			<-doneCh
			if atomic.AddInt32(&procCount, -1) == 0 && spec.ExitWhenNoProc {
				ctx.Cancel()
			}
		}()

		return nil
	}

	handleRemove := func(data *MainProcMsg_Remove) error {
		if ok := ctx.Loop.Remove(data.ProcID); !ok {
			logger.Errorf("proc[%v] not removed", data.ProcID)
			return fmt.Errorf("not removed")
		}
		logger.Debugf("removed proc[%v]", data.ProcID)
		return nil
	}

	handleMessage := func(msg *MainProcMsg) {
		var dataToReturn []byte

		switch msg.Kind {
		case MainProcMsgKind_Exit:
			ctx.Cancel()
			ctx.Loop.Stop()
			logger.Info("EXIT")

		case MainProcMsgKind_Start:
			data, err := helper.DecodeAs[MainProcMsg_Start](msg.Data)
			if err != nil {
				logger.Error("decode MainProcMsg_Start:", err)
			} else {
				err = handleStart(&data)
			}
			if err != nil {
				data.Error = err.Error()
			}
			dataToReturn = helper.MustEncode(data)

		case MainProcMsgKind_Remove:
			data, err := helper.DecodeAs[MainProcMsg_Remove](msg.Data)
			if err != nil {
				logger.Error("decode MainProcMsg_Remove:", err)
			} else {
				err = handleRemove(&data)
			}
			data.Error = err.Error()
			dataToReturn = helper.MustEncode(data)
		}

		if dataToReturn != nil {
			msg := *msg // clone
			msg.Data = dataToReturn
			ctx.PktOutCh <- &pb.Packet{
				Kind: pb.PacketKind_PacketKindData,
				Data: helper.MustEncode(msg),
			}
		}
	}

forLoop:
	for {
		// Get new packet
		var pkt *pb.Packet
		select {
		case pkt = <-ctx.PktInCh:
			if pkt == nil {
				return // channel closed
			}
		case <-ctx.Done():
			return
		}

		// Handle packet
		switch pkt.Kind {
		case pb.PacketKind_PacketKindData:
			msg, err := helper.DecodeAs[MainProcMsg](pkt.Data)
			if err != nil {
				logger.Error("decode MainProcMsg:", err)
				continue forLoop
			}
			handleMessage(&msg)
		}
	}

}
