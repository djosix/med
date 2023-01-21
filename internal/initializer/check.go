package initializer

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"math/rand"

	"github.com/djosix/med/internal/helper"
	"github.com/djosix/med/internal/readwriter"
)

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

func InitCheckMagic(sendMagic, recvMagic []byte) Initializer {
	return func(ctx context.Context, rw io.ReadWriter) (ctxOut context.Context, rwOut io.ReadWriter, err error) {
		log.Println("InitCheckMagic")

		ctxOut = ctx
		rwOut = rw
		err = CheckMagic(rw, sendMagic, recvMagic)

		return
	}
}

const (
	StatusAccept byte = 0
	StatusReject byte = 1
	StatusVerify byte = 2
)

func getVerifier(hash []byte, salt []byte) []byte {
	var buf []byte
	buf = append(buf, hash...)
	buf = append(buf, salt...)
	return helper.Hash256(buf)
}

func Verify(rw io.ReadWriter, hash []byte) (err error) {
	f := readwriter.NewOrderedFramedReaderWriter(readwriter.NewPlainFramedReaderWriter(rw))

	// hash is empty, directly send ServerAccept
	if len(hash) == 0 {
		return f.WriteFrame([]byte{StatusAccept})
	}

	// hash is not empty, send ServerVerify and salt
	salt := make([]byte, 32)
	if _, err = rand.Read(salt); err != nil {
		return err
	}
	if err = f.WriteFrame([]byte{StatusVerify}); err != nil {
		return err
	}
	if err = f.WriteFrame(salt); err != nil {
		return err
	}

	// read answer and check
	var answer []byte
	if answer, err = f.ReadFrame(); err != nil {
		return err
	}

	// if answer is wrong, send ServerReject
	if verified := bytes.Equal(answer, getVerifier(hash, salt)); !verified {
		return f.WriteFrame([]byte{StatusReject})
	}

	// ServerAccept
	return f.WriteFrame([]byte{StatusAccept})
}

func InitVerify(hash []byte) Initializer {
	return func(ctx context.Context, rw io.ReadWriter) (ctxOut context.Context, rwOut io.ReadWriter, err error) {
		log.Println("InitVerifyClient")

		ctxOut = ctx
		rwOut = rw
		err = Verify(rw, hash)

		return
	}
}

func GetVerified(rw io.ReadWriter, hash []byte) error {
	rejectErr := fmt.Errorf("server rejected")
	statusErr := fmt.Errorf("unexpected status")

	f := readwriter.NewOrderedFramedReaderWriter(readwriter.NewPlainFramedReaderWriter(rw))

	// read server status
	var status byte
	if buf, err := f.ReadFrame(); err != nil {
		return err
	} else if len(buf) != 1 {
		return fmt.Errorf("status is not one byte")
	} else {
		status = buf[0]
	}

	// check server status
	switch status {
	case StatusAccept:
		return nil
	case StatusReject:
		return rejectErr
	case StatusVerify:
		// get salt and return answer
	default:
		return statusErr
	}

	// get salt
	salt, err := f.ReadFrame()
	if err != nil {
		return err
	}

	// ask user for password
	if len(hash) == 0 {
		password, err := helper.PromptHiddenInput("Password: ")
		if err != nil {
			return err
		}
		hash = helper.Hash256(password)
	}

	// compute answer and send it
	answer := getVerifier(hash, salt)
	if err := f.WriteFrame(answer); err != nil {
		return err
	}

	// read server status again
	if buf, err := f.ReadFrame(); err != nil {
		return err
	} else if len(buf) != 1 {
		return fmt.Errorf("status is not one byte")
	} else {
		status = buf[0]
	}

	// check server status
	switch status {
	case StatusAccept:
		return nil
	case StatusReject, StatusVerify:
		return rejectErr
	default:
		log.Fatalln("unknown remote status:", status)
		return statusErr
	}
}

func InitGetVerified(hash []byte) Initializer {
	return func(ctx context.Context, rw io.ReadWriter) (ctxOut context.Context, rwOut io.ReadWriter, err error) {
		log.Println("InitGetVerified")

		ctxOut = ctx
		rwOut = rw
		err = GetVerified(rw, hash)

		return
	}
}
