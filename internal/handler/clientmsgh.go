package handler

import (
	"sync/atomic"

	"github.com/djosix/med/internal/helper"
	pb "github.com/djosix/med/internal/protobuf"
)

type ClientMsgHandler struct {
	MedProcessorImpl
	loop       *helper.ValCond[*MedLoop]
	isTermUsed int32
}

func NewClientMsgHandler() *ClientMsgHandler {
	var loop *MedLoop = nil
	return &ClientMsgHandler{
		loop:       helper.NewValCond(loop),
		isTermUsed: 0,
	}
}

func (h *ClientMsgHandler) RunLoop(loop *MedLoop, msgOutCh chan<- *pb.MedMsg) {
	// check
	{
		if h.loop.IsNot(nil) {
			panic("loop is already running")
		}
		h.loop.Set(loop)
		defer h.loop.Set(nil)
	}

	for {

	}
}

func (h *ClientMsgHandler) StartShell() {
	// check
	{
		if atomic.AddInt32(&h.isTermUsed, 1) != 1 {
			panic("cannot start shell when terminal is being used")
		}
		defer atomic.StoreInt32(&h.isTermUsed, 0)
		h.loop.WaitNotEq(nil)
	}
}
