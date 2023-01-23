package worker

import (
	"github.com/djosix/med/internal/logger"
	pb "github.com/djosix/med/internal/protobuf"
	"google.golang.org/protobuf/proto"
)

////////////////////////////////////////////////////////////////////////////////////////////////

type ClientMainProc struct{}

func NewClientMainProc() *ClientMainProc {
	return &ClientMainProc{}
}

func (p *ClientMainProc) Run(ctx ProcRunCtx) {

	{
		mainMsg := p.getStartReqMsg(pb.MedProcType_MedProcTypeExample)
		content, err := proto.Marshal(mainMsg)
		if err != nil {
			panic("cannot marshal protobuf")
		}

		ctx.MsgOutCh <- &pb.MedMsg{
			Type:    pb.MedMsgType_MedMsgTypeControl,
			Content: content,
		}
		logger.Log("sent msg to create example proc on server")
	}

	{
		msg := &pb.MedMainMsg{}

		if err := proto.Unmarshal((<-ctx.MsgInCh).Content, msg); err != nil {
			panic("cannot unmarshal protobuf")
		}

		if resp, ok := msg.Inner.(*pb.MedMainMsg_StartResp); ok {
			logger.Log("recv resp from server:", resp)
		}
	}
}

func (p *ClientMainProc) getStartReqMsg(t pb.MedProcType) *pb.MedMainMsg {
	return &pb.MedMainMsg{
		Inner: &pb.MedMainMsg_StartReq{
			StartReq: &pb.MedMainStartReq{
				Type: t,
			},
		},
	}
}

////////////////////////////////////////////////////////////////////////////////////////////////

type ServerMainProc struct{}

func NewServerMainProc() *ServerMainProc {
	return &ServerMainProc{}
}

func (p *ServerMainProc) Run(ctx ProcRunCtx) {
	// wg := new(sync.WaitGroup)

	// wg.Add(1)
	// go func() {
	// 	defer wg.Done()

	// }()

	// wg.Wait()

	// for {
	// 	switch msg := <-p.msgInCh; msg.Type {
	// 	case pb.MedMsgType_MedMsgTypeControl:

	// 	}
	// }

	// loop.Cancel()
}
