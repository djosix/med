package readwriter

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"sync"

	"github.com/djosix/med/internal/helper"
	"github.com/djosix/med/internal/logger"
	"github.com/golang/snappy"
)

type FrameReadWriter interface {
	ReadFrame() (frame []byte, err error)
	WriteFrame(frame []byte) (err error)
}

type PlainFrameReadWriter struct {
	inner             *FullReadWriter
	MaxReadFrameSize  uint32
	maxWriteFrameSize uint32
}

func NewPlainFrameReadWriter(inner io.ReadWriter) *PlainFrameReadWriter {
	mb := uint32(1024 * 1024)
	return &PlainFrameReadWriter{
		inner:             &FullReadWriter{inner},
		MaxReadFrameSize:  16 * mb,
		maxWriteFrameSize: 16 * mb,
	}
}

func (f *PlainFrameReadWriter) ReadByte() (byte, error) {
	buf := [1]byte{}
	_, err := f.inner.Read(buf[:])
	return buf[0], err
}

func (f *PlainFrameReadWriter) ReadFrame() (frame []byte, err error) {
	u64n, err := binary.ReadUvarint(f)
	if err != nil {
		return
	}
	n := uint32(u64n)

	if n > f.MaxReadFrameSize {
		return nil, fmt.Errorf("%v exceeds max read frame size %v", n, f.MaxReadFrameSize)
	}

	frame = make([]byte, n)
	_, err = f.inner.Read(frame)

	return
}

func (f *PlainFrameReadWriter) WriteFrame(frame []byte) (err error) {
	size := uint32(len(frame))
	if size > f.maxWriteFrameSize {
		return fmt.Errorf("%v exceeds max write frame size %v", size, f.maxWriteFrameSize)
	}

	buf := binary.AppendUvarint(nil, uint64(size))
	buf = append(buf, frame...)
	if _, err = f.inner.Write(buf); err != nil {
		return
	}

	return
}

type OrderedFrameReadWriter struct {
	inner    FrameReadWriter
	readIdx  byte
	writeIdx byte
}

func NewOrderedFramReadWriter(inner FrameReadWriter) *OrderedFrameReadWriter {
	return &OrderedFrameReadWriter{
		inner:    inner,
		readIdx:  0,
		writeIdx: 0,
	}
}

func (f *OrderedFrameReadWriter) ReadFrame() (frame []byte, err error) {
	if frame, err = f.inner.ReadFrame(); err != nil {
		return nil, err
	}
	i := len(frame) - 1
	if f.readIdx != frame[i] {
		return nil, fmt.Errorf("wrong SeqNo")
	}
	f.readIdx++
	return frame[:i], nil
}

func (f *OrderedFrameReadWriter) WriteFrame(frame []byte) (err error) {
	frame = append(frame, f.writeIdx)
	f.writeIdx++
	return f.inner.WriteFrame(frame)
}

type SnappyFrameReadWriter struct {
	inner FrameReadWriter
}

func NewSnappyFrameReadWriter(inner FrameReadWriter) *SnappyFrameReadWriter {
	return &SnappyFrameReadWriter{inner}
}

func (f *SnappyFrameReadWriter) ReadFrame() (frame []byte, err error) {
	if frame, err = f.inner.ReadFrame(); err != nil {
		return nil, err
	}
	return snappy.Decode(nil, frame)
}

func (f *SnappyFrameReadWriter) WriteFrame(frame []byte) (err error) {
	return f.inner.WriteFrame(snappy.Encode(nil, frame))
}

type CryptedFrameReadWriter struct {
	inner   FrameReadWriter
	rStream cipher.Stream
	wStream cipher.Stream
}

func NewCryptedFrameReadWriter(inner FrameReadWriter, secret []byte) (*CryptedFrameReadWriter, error) {
	secret = helper.HashSalt256(secret, []byte("key"))

	block, err := aes.NewCipher(secret[:16])
	if err != nil {
		return nil, err
	}

	return &CryptedFrameReadWriter{
		inner:   inner,
		rStream: cipher.NewCTR(block, secret[16:]),
		wStream: cipher.NewCTR(block, secret[16:]),
	}, nil
}

func (f *CryptedFrameReadWriter) ReadFrame() (frame []byte, err error) {
	if frame, err = f.inner.ReadFrame(); err != nil {
		return nil, err
	}
	out := make([]byte, len(frame))
	f.rStream.XORKeyStream(out, frame)
	return out, nil
}

func (f *CryptedFrameReadWriter) WriteFrame(frame []byte) (err error) {
	in := make([]byte, len(frame))
	f.wStream.XORKeyStream(in, frame)
	return f.inner.WriteFrame(in)
}

type DebugFrameReadWriter struct {
	inner FrameReadWriter
	mu    sync.Mutex
}

func NewDebugFrameReadWriter(inner FrameReadWriter) *DebugFrameReadWriter {
	return &DebugFrameReadWriter{
		inner: inner,
		mu:    sync.Mutex{},
	}
}

func (f *DebugFrameReadWriter) getMsg(frame []byte, err error) string {
	if err != nil {
		return err.Error()
	} else {
		return hex.EncodeToString(frame)
	}
}

func (f *DebugFrameReadWriter) ReadFrame() (frame []byte, err error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	frame, err = f.inner.ReadFrame()
	logger.Log(">frame", f.getMsg(frame, err))

	return
}

func (f *DebugFrameReadWriter) WriteFrame(frame []byte) (err error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	err = f.inner.WriteFrame(frame)
	logger.Log("<frame", f.getMsg(frame, err))

	return
}
