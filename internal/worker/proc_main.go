package worker

import (
	"github.com/djosix/med/internal/helper"
	"github.com/djosix/med/internal/logger"
	pb "github.com/djosix/med/internal/protobuf"
)

////////////////////////////////////////////////////////////////////////////////////////////////

type MedMainMsgKind byte

const (
	MedMainMsgKindStartReq   MedMainMsgKind = 0
	MedMainMsgKindStartResp  MedMainMsgKind = 1
	MedMainMsgKindRemoveReq  MedMainMsgKind = 2
	MedMainMsgKindRemoveResp MedMainMsgKind = 3
)

type MedMainMsg struct {
	Kind MedMainMsgKind
	Data []byte
}

type MedProcKind byte

const (
	MedProcKindExec MedProcKind = 0
)

type MedMainStartReq struct {
	ProcKind MedProcKind
}

type startReq struct {
	ProcID uint32
}

func decodeMedMainMsg(data []byte) (*MedMainMsg, error) {
	msg := MedMainMsg{}
	err := helper.Decode(data, &msg)
	return &msg, err
}

func encodeMedMainMsg(kind MedMainMsgKind, data []byte) []byte {
	msg := MedMainMsg{Kind: kind, Data: data}
	out, err := helper.Encode(&msg)
	if err != nil {
		panic(err)
	}
	return out
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

	{

		ctx.MsgOutCh <- &pb.MedMsg{
			Type:    pb.MedMsgType_MedMsgTypeControl,
			Content: encodeMedMainMsg(MedMainMsgKindStartReq, p.encodeStartReq(MedProcKindExec)),
		}
		logger.Print("sent MedMainMsgKindStartReq")
	}

	{
		msg := <-ctx.MsgInCh
		logger.Print("XXX: msg:", msg)
		mainMsg, err := decodeMedMainMsg(msg.Content)
		if err != nil {
			logger.Print("decodeMedMainMsg:", err)
		}
		switch mainMsg.Kind {
		case MedMainMsgKindStartResp:
			logger.Print("MedMainMsgKindStartResp")
			startResp, err := p.decodeStartResp(mainMsg.Data)
			if err != nil {
				logger.Print("decodeStartResp:", err)
				return
			}
			logger.Print("startResp:", startResp)
		default:
			logger.Print("unexpected MedMainMsgKind:", mainMsg.Kind)
			return
		}
	}
}

func (p *ClientMainProc) encodeStartReq(procKind MedProcKind) (data []byte) {
	data, err := helper.Encode(&MedMainStartReq{ProcKind: procKind})
	if err != nil {
		panic(err)
	}
	return data
}

func (p *ClientMainProc) decodeStartResp(content []byte) (*startReq, error) {
	startResp := startReq{}
	err := helper.Decode(content, &startResp)
	return &startResp, err
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
		mainMsg, err := decodeMedMainMsg(msg.Content)
		if err != nil {
			logger.Print("decodeMedMainMsg:", err)
		}
		switch mainMsg.Kind {
		case MedMainMsgKindStartReq:
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
			Type:    pb.MedMsgType_MedMsgTypeControl,
			Content: encodeMedMainMsg(MedMainMsgKindStartResp, p.encodeStartResp(87)),
		}
		logger.Print("sent MedMainMsgKindStartResp")
	}
}
func (p *ServerMainProc) encodeStartResp(procID uint32) (data []byte) {
	data, err := helper.Encode(&startReq{ProcID: procID})
	if err != nil {
		panic(err)
	}
	return data
}

func (p *ServerMainProc) decodeStartReq(data []byte) (*MedMainStartReq, error) {
	startReq := MedMainStartReq{}
	err := helper.Decode(data, &startReq)
	return &startReq, err
}
