package helper

import (
	"context"
	"fmt"
	"io"
	"sync"
)

type BreakableReader interface {
	Read(p []byte) (n int, err error)
	BreakRead()
}

type BreakableReaderImpl struct {
	ctx    context.Context
	cancel context.CancelFunc
	reqCh  chan *readBufReq
	reqMu  sync.Mutex
	brkCh  chan struct{}
	brkMu  sync.Mutex
}

type readBufReq struct {
	buf    []byte
	respCh chan<- *readBufResp
}

type readBufResp struct {
	buf []byte
	err error
}

func NewBreakableReader(r io.Reader, bufSize int) *BreakableReaderImpl {
	ctx, cancel := context.WithCancel(context.Background())
	br := BreakableReaderImpl{
		ctx:    ctx,
		cancel: cancel,
		reqCh:  make(chan *readBufReq),
		reqMu:  sync.Mutex{},
		brkCh:  make(chan struct{}),
		brkMu:  sync.Mutex{},
	}
	nextCh := make(chan struct{}, 1)
	bufCh := make(chan []byte, 1)
	errCh := make(chan error, 1)

	go func() {
		<-ctx.Done()
		br.brkMu.Lock()
		close(br.brkCh)
		br.brkMu.Unlock()
	}()

	go func() {
		nextCh <- struct{}{}
		var (
			buf = make([]byte, bufSize)
			n   int
			err error
		)
		for {
			<-nextCh
			n, err = r.Read(buf)
			if err != nil {
				break
			}
			select {
			case bufCh <- buf[:n]:
			case <-ctx.Done():
				return
			}
		}
		for {
			select {
			case errCh <- err:
			case <-ctx.Done():
				return
			}
		}
	}()

	go func() {
		unreadBuf := []byte{}
		for {
			br.brkMu.Lock()
			brkCh := br.brkCh
			br.brkMu.Unlock()

			req := <-br.reqCh
			n := len(req.buf)

			var resp readBufResp
			if n < len(unreadBuf) {
				resp.buf = unreadBuf[:n]
				unreadBuf = append([]byte{}, unreadBuf[:n]...)
			} else if n == len(unreadBuf) {
				resp.buf = unreadBuf
				unreadBuf = []byte{}
			} else {
				select {
				case <-brkCh:
				case <-ctx.Done():
				default:
					select {
					case <-brkCh:
					case <-ctx.Done():
					case resp.buf = <-bufCh:
					default:
						select {
						case <-brkCh:
						case <-ctx.Done():
						case resp.buf = <-bufCh:
						case resp.err = <-errCh:
						}
					}
				}
				if resp.buf != nil {
					nextCh <- struct{}{}
				}
			}

			flushed := false
			select {
			case <-brkCh:
			case <-ctx.Done():
			default:
				select {
				case <-brkCh:
				case <-ctx.Done():
				case req.respCh <- &resp:
					flushed = true
				}
			}
			if !flushed {
				unreadBuf = append(unreadBuf, resp.buf...)
				resp.err = fmt.Errorf("read broken")
				req.respCh <- &resp
			}

			select {
			case <-ctx.Done():
				return
			default:
			}
		}
	}()

	return &br
}

func (r *BreakableReaderImpl) Read(p []byte) (n int, err error) {
	r.reqMu.Lock()
	defer r.reqMu.Unlock()
	respCh := make(chan *readBufResp, 1)
	r.reqCh <- &readBufReq{
		buf:    p,
		respCh: respCh,
	}
	resp := <-respCh
	if resp.err != nil {
		return 0, resp.err
	}
	return copy(p, resp.buf), nil
}

func (r *BreakableReaderImpl) BreakRead() {
	r.brkMu.Lock()
	defer r.brkMu.Unlock()
	select {
	case <-r.ctx.Done():
		return
	default:
		close(r.brkCh)
		r.brkCh = make(chan struct{})
	}
}
