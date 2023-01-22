package initializer

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"

	"github.com/djosix/med/internal/readwriter"
)

func InitCheckMagic(sendMagic, recvMagic []byte) Initializer {
	return func(ctx context.Context, rw io.ReadWriter) (ctxOut context.Context, rwOut io.ReadWriter, err error) {
		log.Println("init: check magic")

		ctxOut = ctx
		rwOut = rw
		err = CheckMagic(rw, sendMagic, recvMagic)

		return
	}
}

var (
	ServerMagic = []byte{0x92, 0x9a, 0x9b, 0x8c, 0x8d, 0x89}
	ClientMagic = []byte{0x92, 0x9a, 0x9b, 0x9c, 0x93, 0x96}
)

func CheckMagic(rw io.ReadWriter, sendMagic, recvMagic []byte) error {
	rw = readwriter.NewFullReadWriter(rw)

	if _, err := rw.Write(sendMagic); err != nil {
		return err
	}

	buf := make([]byte, len(recvMagic))
	if _, err := rw.Read(buf); err != nil {
		return err
	}

	if !bytes.Equal(buf, recvMagic) {
		return fmt.Errorf("invalid magic")
	}

	return nil
}
