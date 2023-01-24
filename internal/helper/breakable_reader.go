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
	Cancel()
}

type BreakableReaderImpl struct {
	ctx    context.Context
	cancel context.CancelFunc
	reqCh  chan *brReadBufReq
	reqMu  sync.Mutex
	brkCh  chan struct{}
	brkMu  sync.Mutex
}

type brReadBufReq struct {
	buf    []byte
	respCh chan<- *brReadBufResp
}

type brReadBufResp struct {
	buf []byte
	err error
}

func NewBreakableReader(r io.Reader, bufSize int) *BreakableReaderImpl {
	ctx, cancel := context.WithCancel(context.Background())
	br := BreakableReaderImpl{
		ctx:    ctx,
		cancel: cancel,
		reqCh:  make(chan *brReadBufReq),
		reqMu:  sync.Mutex{},
		brkCh:  make(chan struct{}),
		brkMu:  sync.Mutex{},
	}
	nextCh := make(chan struct{}, 1)
	bufCh := make(chan []byte, 0)
	errCh := make(chan error, 0)

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

			var resp brReadBufResp
			if n <= len(unreadBuf) {
				resp.buf = unreadBuf[:n]
				unreadBuf = Clone(unreadBuf[n:])
			} else {
				select {
				case <-brkCh:
				default:
					select {
					case <-brkCh:
					case resp.buf = <-bufCh:
					default:
						select {
						case <-brkCh:
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
			default:
				req.respCh <- &resp
				flushed = true
			}
			if !flushed {
				unreadBuf = append(unreadBuf, resp.buf...)
				resp.err = fmt.Errorf("broken")
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
	respCh := make(chan *brReadBufResp, 1)
	r.reqCh <- &brReadBufReq{
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
	case <-r.brkCh:
		return
	default:
		close(r.brkCh)
		r.brkCh = make(chan struct{})
	}
}

func (r *BreakableReaderImpl) Cancel() {
	r.cancel()
}
