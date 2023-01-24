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
	ProcMainMsgKind_Start  ProcMainMsgKind = "start"
	ProcMainMsgKind_Remove ProcMainMsgKind = "remove"
	ProcMainMsgKind_Exit   ProcMainMsgKind = "exit"
)

type ProcMainMsg struct {
	Kind  ProcMainMsgKind
	SeqNo uint32
	Data  []byte
}

type ProcMainMsg_Start struct {
	ProcID   uint32
	ProcKind ProcKind
	Error    error // response
}

type ProcMainMsg_Remove struct {
	ProcID uint32
	Error  error // response
}

////////////////////////////////////////////////////////////////////////////////////////////////

type ClientMainProc struct{}

func NewClientMainProc() *ClientMainProc {
	return &ClientMainProc{}
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
	procToStartBySeqNo := map[uint32]procStartInfo{}

	startProc := func(procKind ProcKind, spec any) {
		seqNo := atomic.AddUint32(&seqNo, 1)
		ctx.PktOutCh <- &pb.Packet{
			Kind: pb.PacketKind_PacketKindData,
			Data: helper.MustEncode(&ProcMainMsg{
				Kind:  ProcMainMsgKind_Start,
				SeqNo: seqNo,
				Data:  helper.MustEncode(&ProcMainMsg_Start{ProcKind: procKind}),
			}),
		}
		proc := NewExampleProc("client")
		procID, handle := ctx.Loop.StartLater(proc)
		procToStartBySeqNo[seqNo] = procStartInfo{procID: procID, handle: handle}
	}
	_ = startProc

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

			p, exists := procToStartBySeqNo[msg.SeqNo]
			if !exists {
				logger.Error("no proc to start")
				return
			}
			delete(procToStartBySeqNo, msg.SeqNo)

			if data.Error != nil {
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

			if data.Error != nil {
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

		logger.Debug("packet:", pkt)

		switch pkt.Kind {
		case pb.PacketKind_PacketKindData:
			msg, err := helper.DecodeAs[ProcMainMsg](pkt.Data)
			if err != nil {
				logger.Error("decode as msg:", err)
				continue
			}
			handleMessage(msg)
		default:
			logger.Error("invalid packet kind:", pkt.Kind)
		}
	}
}

////////////////////////////////////////////////////////////////////////////////////////////////

type ServerMainProc struct{}

func NewServerMainProc() *ServerMainProc {
	return &ServerMainProc{}
}

func HandlePackets(ctx ProcRunCtx, f func(pkt *pb.Packet) (cont bool)) {
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
		if !f(pkt) {
			return
		}
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
			startHandle(true)
			return fmt.Errorf("invalid new procID")
		}

		logger.Debug("start proc[%v] kind=%v", startProcID, data.ProcKind)
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
				err = handleStart(data)
			}
			data.Error = err
			dataToReturn = helper.MustEncode(data)

		case ProcMainMsgKind_Remove:
			data, err := helper.DecodeAs[ProcMainMsg_Remove](msg.Data)
			if err != nil {
				logger.Error("decode ProcMainRemoveSpec:", err)
			} else {
				err = handleRemove(data)
			}
			data.Error = err
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
			handleMessage(msg)
		}
	}

}
