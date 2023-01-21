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
	"github.com/golang/snappy"
)

type FrameReadWriter interface {
	ReadFrame() (frame []byte, err error)
	WriteFrame(frame []byte) (err error)
}

type PlainFrameReaderWriter struct {
	inner             *FullReadWriter
	MaxReadFrameSize  uint32
	maxWriteFrameSize uint32
}

func NewPlainFramedReaderWriter(inner io.ReadWriter) *PlainFrameReaderWriter {
	mb := uint32(1024 * 1024)
	return &PlainFrameReaderWriter{
		inner:             &FullReadWriter{inner},
		MaxReadFrameSize:  16 * mb,
		maxWriteFrameSize: 16 * mb,
	}
}

func (f *PlainFrameReaderWriter) ReadByte() (byte, error) {
	buf := [1]byte{}
	_, err := f.inner.Read(buf[:])
	return buf[0], err
}

func (f *PlainFrameReaderWriter) ReadFrame() (frame []byte, err error) {
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

func (f *PlainFrameReaderWriter) WriteFrame(frame []byte) (err error) {
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

type OrderedFrameReaderWriter struct {
	inner    FrameReadWriter
	readIdx  byte
	writeIdx byte
}

func NewOrderedFramedReaderWriter(inner FrameReadWriter) *OrderedFrameReaderWriter {
	return &OrderedFrameReaderWriter{
		inner:    inner,
		readIdx:  0,
		writeIdx: 0,
	}
}

func (f *OrderedFrameReaderWriter) ReadFrame() (frame []byte, err error) {
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

func (f *OrderedFrameReaderWriter) WriteFrame(frame []byte) (err error) {
	frame = append(frame, f.writeIdx)
	f.writeIdx++
	return f.inner.WriteFrame(frame)
}

type SnappyFrameReaderWriter struct {
	inner FrameReadWriter
}

func NewSnappyFramedReaderWriter(inner FrameReadWriter) *SnappyFrameReaderWriter {
	return &SnappyFrameReaderWriter{inner}
}

func (f *SnappyFrameReaderWriter) ReadFrame() (frame []byte, err error) {
	if frame, err = f.inner.ReadFrame(); err != nil {
		return nil, err
	}
	return snappy.Decode(nil, frame)
}

func (f *SnappyFrameReaderWriter) WriteFrame(frame []byte) (err error) {
	return f.inner.WriteFrame(snappy.Encode(nil, frame))
}

type CryptedFrameReaderWriter struct {
	inner   FrameReadWriter
	rStream cipher.Stream
	wStream cipher.Stream
}

func NewCryptedFrameReaderWriter(inner FrameReadWriter, secret []byte) (*CryptedFrameReaderWriter, error) {
	secret = helper.HashSalt256(secret, []byte("key"))

	block, err := aes.NewCipher(secret[:16])
	if err != nil {
		return nil, err
	}

	return &CryptedFrameReaderWriter{
		inner:   inner,
		rStream: cipher.NewCTR(block, secret[16:]),
		wStream: cipher.NewCTR(block, secret[16:]),
	}, nil
}

func (f *CryptedFrameReaderWriter) ReadFrame() (frame []byte, err error) {
	if frame, err = f.inner.ReadFrame(); err != nil {
		return nil, err
	}
	out := make([]byte, len(frame))
	f.rStream.XORKeyStream(out, frame)
	return out, nil
}

func (f *CryptedFrameReaderWriter) WriteFrame(frame []byte) (err error) {
	in := make([]byte, len(frame))
	f.wStream.XORKeyStream(in, frame)
	return f.inner.WriteFrame(in)
}

type DebugFrameReaderWriter struct {
	inner FrameReadWriter
	mu    sync.Mutex
}

func NewDebugFrameReaderWriter(inner FrameReadWriter) *DebugFrameReaderWriter {
	return &DebugFrameReaderWriter{
		inner: inner,
		mu:    sync.Mutex{},
	}
}

func (f *DebugFrameReaderWriter) getMsg(frame []byte, err error) string {
	if err != nil {
		return err.Error()
	} else {
		return hex.EncodeToString(frame)
	}
}

func (f *DebugFrameReaderWriter) ReadFrame() (frame []byte, err error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	frame, err = f.inner.ReadFrame()
	fmt.Println(">frame", f.getMsg(frame, err))

	return
}

func (f *DebugFrameReaderWriter) WriteFrame(frame []byte) (err error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	err = f.inner.WriteFrame(frame)
	fmt.Println("<frame", f.getMsg(frame, err))

	return
}
