package helper

// import (
// 	"io"
// 	"os"
// )

// type ReadCh interface {
// 	NewCh() <-chan []byte
// }

// type WriteCh interface {
// 	NewCh() chan<- []byte
// }

// type ReadChImpl struct {
// 	r      io.Reader
// 	ch     chan []byte
// 	update chan struct{}
// 	newCh  chan chan []byte
// }

// func NewReadCh(r io.Reader) *ReadChImpl {
// 	readCh := ReadChImpl{
// 		r:      r,
// 		ch:     make(chan []byte),
// 		update: make(chan struct{}),
// 		newCh:  make(chan chan []byte),
// 	}
// 	go func() {
// 		for {
// 			buf := [1024]byte{}
// 			for {
// 				n, err := r.Read(buf[:])
// 				if err != nil || n == 0 {
// 					return
// 				}
// 				select {
// 				case readCh.ch <- append([]byte{}, buf[:n]...):
// 				case <-readCh.update:
// 					oldCh := readCh.ch
// 					readCh.ch = make(chan []byte)
// 					readCh.newCh <- readCh.ch
// 					close(oldCh)
// 				}
// 			}
// 		}
// 	}()
// 	return &readCh
// }

// func (c *ReadChImpl) NewCh() <-chan []byte {
// 	c.update <- struct{}{}
// 	return <-c.newCh
// }

// type WriteChImpl struct {
// 	w  io.Writer
// 	ch chan []byte
// }

// func NewWriteCh(w io.Writer) *WriteChImpl {
// 	writeCh := WriteChImpl{
// 		w:  w,
// 		ch: make(chan []byte),
// 	}
// 	go func() {
// 		for buf := range writeCh.ch {

// 		}
// 	}()
// 	return &writeCh
// }

// type StdioCh struct {
// 	Stdin  ReadCh
// 	Stdout WriteCh
// 	Stderr WriteCh
// }

// func CreateStdioChans() *StdioCh {
// 	return &StdioCh{
// 		Stdin: NewReadCh(os.Stdin),
// 	}
// }
