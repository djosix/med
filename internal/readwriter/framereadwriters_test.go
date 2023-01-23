package readwriter

import (
	"bytes"
	"encoding/binary"
	"io"
	"math/rand"
	"testing"
)

type TestReadWriter struct {
	ReadBuf  []byte
	ReadIdx  int
	WriteBuf []byte
}

func NewTestReadWriter() *TestReadWriter {
	return &TestReadWriter{
		ReadBuf:  []byte{},
		ReadIdx:  0,
		WriteBuf: []byte{},
	}
}

func (f *TestReadWriter) Read(p []byte) (n int, err error) {
	sizeToRead := len(p)
	if sizeToRead == 0 {
		return 0, nil
	}

	sizeUnread := len(f.ReadBuf) - f.ReadIdx
	if sizeUnread == 0 {
		return 0, io.EOF
	}

	sizeToRead = rand.Intn(sizeToRead) + 1
	if sizeToRead < 1 {
		sizeToRead = 1
	}
	if sizeToRead > sizeUnread {
		sizeToRead = sizeUnread
	}

	n = copy(p[:sizeToRead], f.ReadBuf[f.ReadIdx:])
	if n != sizeToRead {
		panic(nil)
	}
	f.ReadIdx += n

	return n, nil
}

func (f *TestReadWriter) Write(p []byte) (n int, err error) {
	f.WriteBuf = append(f.WriteBuf, p...)
	return len(p), nil
}

func TestPlainFramedReaderWriter(t *testing.T) {
	inner := NewTestReadWriter()

	buf := []byte{}
	for i := 0; i < 1000; i++ {
		buf = append(buf, 87)
	}
	inner.ReadBuf = binary.AppendUvarint(inner.ReadBuf, uint64(len(buf)))
	inner.ReadBuf = append(inner.ReadBuf, buf...)
	for i := 0; i < 1000; i++ {
		inner.ReadBuf = append(inner.ReadBuf, 0)
	}

	rw := NewPlainFrameReadWriter(inner)
	frame, err := rw.ReadFrame()
	// logger.Log(buf)
	// logger.Log(inner.ReadBuf)
	// logger.Log(frame)

	if err != nil {
		t.Errorf("error: %v", err)
	}

	if !bytes.Equal(frame, buf) {
		t.Errorf("frame, buf inequal")
	}

	if err := rw.WriteFrame(buf); err != nil {
		t.Errorf("error: %v", err)
	}

	inner.ReadBuf = inner.WriteBuf
	inner.ReadIdx = 0
	for i := 0; i < 1000; i++ {
		inner.ReadBuf = append(inner.ReadBuf, 0)
	}

	frame, err = rw.ReadFrame()
	if err != nil {
		t.Errorf("error: %v", err)
	}

	if !bytes.Equal(frame, buf) {
		t.Errorf("not equal")
	}
}

func TestSnappyFramedReaderWriter(t *testing.T) {
	inner := NewTestReadWriter()
	rw1 := NewPlainFrameReadWriter(inner)
	rw2 := NewSnappyFrameReadWriter(rw1)

	buf := []byte{}
	for i := 0; i < 1000; i++ {
		buf = append(buf, 87)
	}

	for i := 0; i < 5; i++ {
		if err := rw2.WriteFrame(buf); err != nil {
			t.Error(err)
			return
		}
	}

	for i := 0; i < 1000; i++ {
		inner.WriteBuf = append(inner.WriteBuf, 0)
	}
	inner.ReadBuf = inner.WriteBuf

	for i := 0; i < 5; i++ {
		frame, err := rw2.ReadFrame()
		if err != nil {
			t.Error(err)
			return
		}
		if !bytes.Equal(frame, buf) {
			t.Errorf("not equal")
		}
	}
}
func TestCryptedFrameReaderWriter(t *testing.T) {
	inner := NewTestReadWriter()
	rw1 := NewPlainFrameReadWriter(inner)
	rw2, err := NewCryptedFrameReadWriter(rw1, []byte("asodinaosdnoainsd"))
	if err != nil {
		t.Error(err)
		return
	}

	buf := []byte{}
	for i := 0; i < 1000; i++ {
		buf = append(buf, 87)
	}

	for i := 0; i < 5; i++ {
		if err := rw2.WriteFrame(buf); err != nil {
			t.Error(err)
			return
		}
	}

	for i := 0; i < 1000; i++ {
		inner.WriteBuf = append(inner.WriteBuf, 0)
	}
	inner.ReadBuf = inner.WriteBuf

	for i := 0; i < 5; i++ {
		frame, err := rw2.ReadFrame()
		if err != nil {
			t.Error(err)
			return
		}
		if !bytes.Equal(frame, buf) {
			t.Errorf("not equal")
		}
	}
}
