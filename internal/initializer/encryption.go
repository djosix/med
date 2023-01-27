package initializer

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"fmt"
	"io"
	"sync"

	"github.com/djosix/med/internal"
	"github.com/djosix/med/internal/helper"
)

func InitEncryption(secret []byte) Initializer {
	return func(ctx context.Context, rwc io.ReadWriteCloser) (ctxOut context.Context, rwcOut io.ReadWriteCloser, err error) {
		initLogger.Debug("Encryption")

		secret := secret
		if secret == nil {
			b, ok := ctx.Value("secret").([]byte)
			if !ok {
				err = fmt.Errorf("secret not in context")
				return
			}
			secret = b
		}

		ctxOut = ctx
		rw, err := NewEncryptionLayer(rwc, secret)
		rwcOut = helper.NewReadWriterCloser(rw, rw, rwc)

		return
	}
}

type EncryptionLayer struct {
	io.ReadWriter
	readStream  cipher.Stream
	readMutex   sync.Mutex
	writeStream cipher.Stream
	writeMutex  sync.Mutex
}

func NewEncryptionLayer(rw io.ReadWriter, secret []byte) (*EncryptionLayer, error) {
	hash := helper.Hash256(secret)
	key, iv := hash[:16], hash[16:]
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	el := EncryptionLayer{
		ReadWriter:  rw,
		readStream:  cipher.NewCTR(block, iv),
		readMutex:   sync.Mutex{},
		writeStream: cipher.NewCTR(block, iv),
		writeMutex:  sync.Mutex{},
	}

	return &el, nil
}

func (el *EncryptionLayer) Read(p []byte) (n int, err error) {
	el.readMutex.Lock()
	defer el.readMutex.Unlock()

	buf := make([]byte, len(p)) // TODO

	n, err = el.ReadWriter.Read(buf)
	if err != nil {
		return
	}

	if n > 0 {
		el.readStream.XORKeyStream(p, buf[:n])
	}

	return
}

func (el *EncryptionLayer) Write(p []byte) (n int, err error) {
	el.writeMutex.Lock()
	defer el.writeMutex.Unlock()

	buf := make([]byte, len(p))
	el.writeStream.XORKeyStream(buf, p)

	n, err = el.ReadWriter.Write(buf)
	if err != nil {
		return
	}
	if n != len(p) {
		panic(internal.Unexpected)
	}

	return
}
