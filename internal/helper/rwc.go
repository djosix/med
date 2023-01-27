package helper

import "io"

type ReadWriteCloser struct {
	io.Reader
	io.Writer
	io.Closer
}

func NewReadWriterCloser(r io.Reader, w io.Writer, c io.Closer) io.ReadWriteCloser {
	return &ReadWriteCloser{Reader: r, Writer: w, Closer: c}
}
