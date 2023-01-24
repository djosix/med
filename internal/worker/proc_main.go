package worker

import (
	"fmt"
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

type MainProcClient struct {
	ProcInfo
}

func NewMainProcClient() *MainProcClient {
	return &MainProcClient{
		ProcInfo: NewProcInfo(ProcKind_Main, ProcSide_Client),
	}
}

func (p *MainProcClient) Run(ctx *ProcRunCtx) {
	logger := logger.NewLogger("MainProcClient")
	logger.Info("start")
	defer logger.Info("done")

	var seqNo uint32 = 0

	type procStartInfo struct {
		procID uint32
		handle func(bool)
	}
	startProcBySeqNo := map[uint32]procStartInfo{}

	startProc := func(procKind ProcKind, spec any) {
		proc := CreateProcClient(procKind, spec)
		if proc == nil {
			return
		}

		procID, handle := ctx.Loop.StartLater(proc)

		seqNo := atomic.AddUint32(&seqNo, 1)
		ctx.PktOutCh <- &pb.Packet{
			Kind: pb.PacketKind_PacketKindData,
			Data: helper.MustEncode(&MainProcMsg{
				Kind:  MainProcMsgKind_Start,
				SeqNo: seqNo,
				Data: helper.MustEncode(&MainProcMsg_Start{
					ProcKind: procKind,
					ProcID:   procID,
				}),
			}),
		}
		startProcBySeqNo[seqNo] = procStartInfo{procID: procID, handle: handle}
	}
	_ = startProc

	startProc(ProcKind_Exec, ExecSpec{
		ARGV: []string{"bash"},
		TTY:  true,
	})
	// startProc(ProcKind_Example, ExampleSpec{
	// 	Name: "djosix",
	// })

	removeProc := func(procID uint32) {
		seqNo := atomic.AddUint32(&seqNo, 1)
		ctx.PktOutCh <- &pb.Packet{
			Kind: pb.PacketKind_PacketKindData,
			Data: helper.MustEncode(&MainProcMsg{
				Kind:  MainProcMsgKind_Remove,
				SeqNo: seqNo,
				Data:  helper.MustEncode(&MainProcMsg_Remove{ProcID: procID}),
			}),
		}
	}
	_ = removeProc

	handleMessage := func(msg *MainProcMsg) {
		switch msg.Kind {
		case MainProcMsgKind_Start:
			data, err := helper.DecodeAs[MainProcMsg_Start](msg.Data)
			if err != nil {
				logger.Error("decode start error:", err)
				return
			}

			p, exists := startProcBySeqNo[msg.SeqNo]
			if !exists {
				logger.Error("no proc to start")
				return
			}
			delete(startProcBySeqNo, msg.SeqNo)

			if data.Error != "" {
				logger.Errorf("start proc[%v]: %v", p.procID, data.Error)
				p.handle(false) // cancel start
			} else {
				logger.Debugf("start proc[%v]", p.procID)
				p.handle(true)
			}

		case MainProcMsgKind_Remove:
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

	for {
		var pkt *pb.Packet
		select {
		case pkt = <-ctx.PktInCh:
			if pkt == nil {
				break
			}
		case <-ctx.Done():
			break
		}

		logger.Debugf("receive: [%v]", pkt)

		switch pkt.Kind {
		case pb.PacketKind_PacketKindData:
			msg, err := helper.DecodeAs[MainProcMsg](pkt.Data)
			if err != nil {
				logger.Error("decode as msg:", err)
				continue
			}
			handleMessage(&msg)
		default:
			logger.Error("invalid packet kind:", pkt.Kind)
		}
	}
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

	handleStart := func(data *MainProcMsg_Start) error {
		// Create proc by kind
		proc := CreateProcServer(data.ProcKind)
		if proc == nil {
			// Send error
			ctx.PktOutCh <- &pb.Packet{
				Kind: pb.PacketKind_PacketKindError,
				Data: []byte(fmt.Sprintf("proc not created: kind=%v", data.ProcKind)),
			}
			return fmt.Errorf("not created")
		}

		startProcID, startHandle := ctx.Loop.StartLater(proc)

		if startProcID != data.ProcID {
			startHandle(false)
			return fmt.Errorf("invalid new procID: required=%v got=%v", data.ProcID, startProcID)
		}

		logger.Debugf("start proc[%v] kind=%v", startProcID, data.ProcKind)
		startHandle(true)

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
				continue
			}
			handleMessage(&msg)
		}
	}

}
