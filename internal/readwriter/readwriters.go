package readwriter

import (
	"encoding/hex"
	"fmt"
	"io"
	"sync"

	"github.com/djosix/med/internal/logger"
)

type FullReadWriter struct {
	inner io.ReadWriter
}

func NewFullReadWriter(inner io.ReadWriter) *FullReadWriter {
	return &FullReadWriter{inner}
}

func (rw *FullReadWriter) Read(p []byte) (n int, err error) {
	n, err = io.ReadFull(rw.inner, p)
	if err == nil && n != len(p) {
		err = fmt.Errorf("cannot read full %v", len(p))
	}
	return
}

func (rw *FullReadWriter) Write(p []byte) (n int, err error) {
	n, err = rw.inner.Write(p)
	if err == nil && n != len(p) {
		err = fmt.Errorf("cannot write full %v", len(p))
	}
	return
}

type DebugReadWriter struct {
	inner io.ReadWriter
	mu    sync.Mutex
}

func NewDebugReadWriter(inner io.ReadWriter) *DebugReadWriter {
	return &DebugReadWriter{
		inner: inner,
		mu:    sync.Mutex{},
	}
}

func (rw *DebugReadWriter) getMsg(data []byte, err error) string {
	if err != nil {
		return err.Error()
	} else {
		return hex.EncodeToString(data)
	}
}

func (rw *DebugReadWriter) Read(b []byte) (n int, err error) {
	rw.mu.Lock()
	defer rw.mu.Unlock()

	n, err = rw.inner.Read(b)
	logger.Debug(">", rw.getMsg(b[:n], err))

	return
}

func (rw *DebugReadWriter) Write(b []byte) (n int, err error) {
	rw.mu.Lock()
	defer rw.mu.Unlock()

	n, err = rw.inner.Write(b)
	logger.Debug("<", rw.getMsg(b[:n], err))

	return
}
