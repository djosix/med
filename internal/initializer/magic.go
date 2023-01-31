package initializer

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/djosix/med/internal"
	"github.com/djosix/med/internal/helper"
	"github.com/djosix/med/internal/readwriter"
)

func InitCheckMagic(sendMagic, recvMagic []byte) Initializer {
	return func(ctx context.Context, rwc io.ReadWriteCloser) (ctxOut context.Context, rwcOut io.ReadWriteCloser, err error) {
		initLogger.Debug("CheckMagic")

		ctxOut = ctx
		rwcOut = rwc
		err = CheckMagic(rwc, sendMagic, recvMagic)

		return
	}
}

// const (
// 	MagicIsServer      = 0b00000001
// 	MagicIsClient      = 0b00000010
// 	MagicIsRelay       = 0b00000100
// 	MagicHasPassword   = 0b00001000
// 	MagicHasEncryption = 0b00010000
// 	MagicHasHandshake  = 0b00100000
// )

var (
	ServerMagic = helper.HashSalt256(internal.Nonce, []byte("ServerMagic"))
	ClientMagic = helper.HashSalt256(internal.Nonce, []byte("ClientMagic"))
)

// func MakeMagic() []byte {
// 	randBytes := make([]byte, 4)
// 	rand.Read(randBytes[:])

// }

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
