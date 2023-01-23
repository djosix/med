package readwriter

import (
	"context"
	"fmt"
)

type ChanWriter struct {
	ctx context.Context
	ch  chan []byte
}

func NewChanWriter(ctx context.Context) *ChanWriter {
	return &ChanWriter{
		ctx: ctx,
		ch:  make(chan []byte, 1),
	}
}

func (w *ChanWriter) Write(p []byte) (n int, err error) {
	select {
	case w.ch <- p:
		return len(p), nil
	case <-w.ctx.Done():
		return 0, fmt.Errorf("broken")
	}
}

type ChanReader struct {
	ctx context.Context
	ch  chan []byte
	buf []byte
}

func NewChanReader(ctx context.Context) *ChanReader {
	return &ChanReader{
		ctx: ctx,
		ch:  make(chan []byte, 1),
		buf: []byte{},
	}
}

func (r *ChanReader) Read(p []byte) (n int, err error) {
	if len(r.buf) > 0 {
		n = copy(p, r.buf)
		r.buf = append([]byte{}, r.buf[n:]...)
		return n, nil
	}

	select {
	case buf := <-r.ch:
		n = copy(p, buf)
		buf = buf[n:]
		if len(buf) > 0 {
			r.buf = append(r.buf, buf...)
		}
		return n, nil
	case <-r.ctx.Done():
		return 0, fmt.Errorf("broken")
	}

}
