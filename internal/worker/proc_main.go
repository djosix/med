package worker

import (
	"fmt"
	"sync/atomic"

	"github.com/djosix/med/internal/helper"
	"github.com/djosix/med/internal/logger"
	pb "github.com/djosix/med/internal/protobuf"
)

////////////////////////////////////////////////////////////////////////////////////////////////

type ProcMainMsgKind string

const (
	ProcMainMsgKind_Start  ProcMainMsgKind = "Start"
	ProcMainMsgKind_Remove ProcMainMsgKind = "Remove"
	ProcMainMsgKind_Exit   ProcMainMsgKind = "Exit"
)

type ProcMainMsg struct {
	Kind  ProcMainMsgKind
	SeqNo uint32
	Data  []byte
}

type ProcMainMsg_Start struct {
	ProcID   uint32
	ProcKind ProcKind
	Error    string // response
}

type ProcMainMsg_Remove struct {
	ProcID uint32
	Error  string // response
}

////////////////////////////////////////////////////////////////////////////////////////////////

type ClientMainProc struct {
	ProcInfo
}

func NewClientMainProc() *ClientMainProc {
	return &ClientMainProc{
		ProcInfo: NewProcInfo(ProcKind_Main, ProcSide_Client),
	}
}

func (p *ClientMainProc) Run(ctx ProcRunCtx) {
	logger := logger.NewLogger("ClientMainProc")
	logger.Info("start")
	defer logger.Info("done")

	var seqNo uint32 = 0

	type procStartInfo struct {
		procID uint32
		handle func(bool)
	}
	startProcBySeqNo := map[uint32]procStartInfo{}

	startProc := func(procKind ProcKind, spec any) {
		proc := CreateClientProc(procKind, spec)
		if proc == nil {
			return
		}

		procID, handle := ctx.Loop.StartLater(proc)

		seqNo := atomic.AddUint32(&seqNo, 1)
		ctx.PktOutCh <- &pb.Packet{
			Kind: pb.PacketKind_PacketKindData,
			Data: helper.MustEncode(&ProcMainMsg{
				Kind:  ProcMainMsgKind_Start,
				SeqNo: seqNo,
				Data: helper.MustEncode(&ProcMainMsg_Start{
					ProcKind: procKind,
					ProcID:   procID,
				}),
			}),
		}
		startProcBySeqNo[seqNo] = procStartInfo{procID: procID, handle: handle}
	}
	_ = startProc

	startProc(ProcKind_Exec, ProcExecSpec{
		ARGV: []string{"bash"},
		TTY:  true,
	})

	removeProc := func(procID uint32) {
		seqNo := atomic.AddUint32(&seqNo, 1)
		ctx.PktOutCh <- &pb.Packet{
			Kind: pb.PacketKind_PacketKindData,
			Data: helper.MustEncode(&ProcMainMsg{
				Kind:  ProcMainMsgKind_Remove,
				SeqNo: seqNo,
				Data:  helper.MustEncode(&ProcMainMsg_Remove{ProcID: procID}),
			}),
		}
	}
	_ = removeProc

	handleMessage := func(msg *ProcMainMsg) {
		switch msg.Kind {
		case ProcMainMsgKind_Start:
			data, err := helper.DecodeAs[ProcMainMsg_Start](msg.Data)
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

		case ProcMainMsgKind_Remove:
			data, err := helper.DecodeAs[ProcMainMsg_Remove](msg.Data)
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
			msg, err := helper.DecodeAs[ProcMainMsg](pkt.Data)
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

////////////////////////////////////////////////////////////////////////////////////////////////

type ServerMainProc struct {
	ProcInfo
}

func NewServerMainProc() *ServerMainProc {
	return &ServerMainProc{
		ProcInfo: NewProcInfo(ProcKind_Main, ProcSide_Server),
	}
}

func (p *ServerMainProc) Run(ctx ProcRunCtx) {
	logger := logger.NewLogger("ServerMainProc")
	logger.Debug("start")
	defer logger.Debug("done")

	handleStart := func(data *ProcMainMsg_Start) error {
		// Create proc by kind
		var proc Proc
		switch data.ProcKind {
		case ProcKind_Exec:
			proc = NewServerExecProc()
		default:
			logger.Error("unknown proc kind:", data.ProcKind)
		}

		if proc == nil {
			// Send error
			select {
			case ctx.PktOutCh <- &pb.Packet{
				Kind: pb.PacketKind_PacketKindError,
				Data: []byte(fmt.Sprintf("proc not created: kind=%v", data.ProcKind)),
			}:
			case <-ctx.Done():
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

	handleRemove := func(data *ProcMainMsg_Remove) error {
		if ok := ctx.Loop.Remove(data.ProcID); !ok {
			logger.Errorf("proc[%v] not removed", data.ProcID)
			return fmt.Errorf("not removed")
		}
		logger.Debugf("removed proc[%v]", data.ProcID)
		return nil
	}

	handleMessage := func(msg *ProcMainMsg) {
		var dataToReturn []byte

		switch msg.Kind {
		case ProcMainMsgKind_Exit:
			ctx.Cancel()
			ctx.Loop.Stop()
			logger.Info("EXIT")

		case ProcMainMsgKind_Start:
			data, err := helper.DecodeAs[ProcMainMsg_Start](msg.Data)
			if err != nil {
				logger.Error("decode ProcMainStartSpec:", err)
			} else {
				err = handleStart(&data)
			}
			if err != nil {
				data.Error = err.Error()
			}
			dataToReturn = helper.MustEncode(data)

		case ProcMainMsgKind_Remove:
			data, err := helper.DecodeAs[ProcMainMsg_Remove](msg.Data)
			if err != nil {
				logger.Error("decode ProcMainRemoveSpec:", err)
			} else {
				err = handleRemove(&data)
			}
			data.Error = err.Error()
			dataToReturn = helper.MustEncode(data)
		}

		if dataToReturn != nil {
			msg := *msg // clone
			msg.Data = dataToReturn
			pkt := &pb.Packet{
				Kind: pb.PacketKind_PacketKindData,
				Data: helper.MustEncode(msg),
			}

			select {
			case ctx.PktOutCh <- pkt:
			case <-ctx.Done():
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
			msg, err := helper.DecodeAs[ProcMainMsg](pkt.Data)
			if err != nil {
				logger.Error("decode msg:", err)
				continue
			}
			handleMessage(&msg)
		}
	}

}
