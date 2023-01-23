package helper

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/djosix/med/internal/logger"
)

type MyReader struct {
	rootCtx context.Context
	ctx     context.Context
	cancel  context.CancelFunc
	idx     int
}

func NewMyReader() *MyReader {
	r := MyReader{}
	r.rootCtx = context.Background()
	r.ctx, r.cancel = context.WithCancel(r.rootCtx)
	return &r
}

func (r *MyReader) Read(p []byte) (int, error) {
	<-r.ctx.Done()
	for i := range p {
		p[i] = byte(r.idx % 256)
		r.idx++
	}
	return len(p), nil
}

func (r *MyReader) FlushRead() {
	r.cancel()
	r.ctx, r.cancel = context.WithCancel(r.rootCtx)
}

func TestBreakableReader(t *testing.T) {
	r := NewMyReader()
	br := NewBreakableReader(r, 4)

	breakReadLater := func(d time.Duration) {
		time.Sleep(d)
		logger.Print("Break Read")
		br.BreakRead()
	}

	testRead := func() ([]byte, error) {
		buf := make([]byte, 32)
		logger.Print("Test Read")
		n, err := br.Read(buf)
		logger.Printf("Test Read Done: buf=%v err=%v", buf[:n], err)
		return buf[:n], err
	}

	flushReadLater := func(d time.Duration) {
		time.Sleep(d)
		logger.Print("Flush Read")
		r.FlushRead()
	}

	go breakReadLater(100 * time.Millisecond)
	if buf, err := testRead(); true {
		if bytes.Equal(buf, []byte{}) && err != nil {
			logger.Print("ok")
		} else {
			t.Error(buf, err)
		}
	}

	go flushReadLater(100 * time.Millisecond)
	if buf, err := testRead(); true {
		if bytes.Equal(buf, []byte{0, 1, 2, 3}) && err == nil {
			logger.Print("ok")
		} else {
			t.Error(buf, err)
		}
	}

	go flushReadLater(100 * time.Millisecond)
	if buf, err := testRead(); true {
		if bytes.Equal(buf, []byte{4, 5, 6, 7}) && err == nil {
			logger.Print("ok")
		} else {
			t.Error(buf, err)
		}
	}
}
