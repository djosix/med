package worker

import (
	"github.com/djosix/med/internal/helper"
	"github.com/djosix/med/internal/logger"
	pb "github.com/djosix/med/internal/protobuf"
)

////////////////////////////////////////////////////////////////////////////////////////////////

type ProcMainMsgKind byte

const (
	ProcMainMsgKind_Start      ProcMainMsgKind = 0
	ProcMainMsgKind_StartResp  ProcMainMsgKind = 1
	ProcMainMsgKind_RemoveReq  ProcMainMsgKind = 2
	ProcMainMsgKind_RemoveResp ProcMainMsgKind = 3
)

type ProcMainMsg struct {
	Kind ProcMainMsgKind
	Data []byte
}

type ProcMainStart struct {
	ProcKind ProcKind
	SpecData []byte
}

type ProcMainRemove struct {
	ProcID uint32
}

func encodeMainMsg(kind ProcMainMsgKind, data []byte) []byte {
	msg := ProcMainMsg{Kind: kind, Data: data}
	out, err := helper.Encode(&msg)
	if err != nil {
		panic(err)
	}
	return out
}

func decMainMsg(data []byte) (*ProcMainMsg, error) {
	msg := ProcMainMsg{}
	err := helper.Decode(data, &msg)
	return &msg, err
}

////////////////////////////////////////////////////////////////////////////////////////////////

type ClientMainProc struct{}

func NewClientMainProc() *ClientMainProc {
	return &ClientMainProc{}
}

func (p *ClientMainProc) Run(ctx ProcRunCtx) {

	logger := logger.NewLogger("ClientMainProc")
	logger.Info("Begin")
	defer logger.Info("End")

	start := func() {
		ctx.MsgOutCh <- &pb.MedMsg{
			Type: pb.MedMsgType_MedMsgTypeControl,
			Data: encodeMainMsg(ProcMainMsgKind_Start, p.encStartReq(ProcKind_Exec)),
		}
	}

	{

		ctx.MsgOutCh <- &pb.MedMsg{
			Type: pb.MedMsgType_MedMsgTypeControl,
			Data: encodeMainMsg(ProcMainMsgKind_Start, p.encStartReq(ProcKind_Exec)),
		}
		logger.Print("sent MedMainMsgKindStartReq")
	}

	{
		msg := <-ctx.MsgInCh
		logger.Print("XXX: msg:", msg)
		mainMsg, err := decMainMsg(msg.Data)
		if err != nil {
			logger.Print("decodeMedMainMsg:", err)
		}
		switch mainMsg.Kind {
		case ProcMainMsgKind_StartResp:
			logger.Print("ProcMainMsgKind_StartResp")
			startResp, err := p.decodeStartResp(mainMsg.Data)
			if err != nil {
				logger.Print("decodeStartResp:", err)
				return
			}
			logger.Print("startResp:", startResp)
		default:
			logger.Print("unexpected ProcMainMsgKind:", mainMsg.Kind)
			return
		}
	}
}

func (p *ClientMainProc) start(ctx *ProcRunCtx) {
	ctx.MsgOutCh <- &pb.MedMsg{
		Type: pb.MedMsgType_MedMsgTypeControl,
		Data: encodeMainMsg(ProcMainMsgKind_Start, p.encStartReq(ProcKind_Exec)),
	}

	msg := <-ctx.MsgInCh
	logger.Print("XXX: msg:", msg)
	mainMsg, err := decMainMsg(msg.Data)
	if err != nil {
		logger.Print("decodeMedMainMsg:", err)
	}
	switch mainMsg.Kind {
	case ProcMainMsgKind_StartResp:
		logger.Print("ProcMainMsgKind_StartResp")
		startResp, err := p.decodeStartResp(mainMsg.Data)
		if err != nil {
			logger.Print("decodeStartResp:", err)
			return
		}
		logger.Print("startResp:", startResp)
	default:
		logger.Print("unexpected ProcMainMsgKind:", mainMsg.Kind)
		return
	}
}

func (p *ClientMainProc) encStartReq(procKind ProcKind) (data []byte) {
	data, err := helper.Encode(&ProcMainStart{ProcKind: procKind})
	if err != nil {
		panic(err)
	}
	return data
}

////////////////////////////////////////////////////////////////////////////////////////////////

type ServerMainProc struct{}

func NewServerMainProc() *ServerMainProc {
	return &ServerMainProc{}
}

func (p *ServerMainProc) Run(ctx ProcRunCtx) {
	{
		msg := <-ctx.MsgInCh
		logger.Print("XXX: msg:", msg)
		mainMsg, err := decMainMsg(msg.Data)
		if err != nil {
			logger.Print("decodeMedMainMsg:", err)
		}
		switch mainMsg.Kind {
		case ProcMainMsgKind_Start:
			logger.Print("ok")
			startReq, err := p.decodeStartReq(mainMsg.Data)
			if err != nil {
				logger.Print("decodeStartReq:", err)
				return
			}
			logger.Print("startReq:", startReq)
		default:
			logger.Print("unexpected MedMainMsgKind:", mainMsg.Kind)
			return
		}
	}
	{
		ctx.MsgOutCh <- &pb.MedMsg{
			Type: pb.MedMsgType_MedMsgTypeControl,
			Data: encodeMainMsg(ProcMainMsgKind_StartResp, p.encodeStartResp(87)),
		}
		logger.Print("sent MedMainMsgKindStartResp")
	}
}
func (p *ServerMainProc) encodeStartResp(procID uint32) (data []byte) {
	data, err := helper.Encode(&ProcMainStartResp{ProcID: procID})
	if err != nil {
		panic(err)
	}
	return data
}

func (p *ServerMainProc) decodeStartReq(data []byte) (*ProcMainStart, error) {
	startReq := ProcMainStart{}
	err := helper.Decode(data, &startReq)
	return &startReq, err
}
