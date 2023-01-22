package initializer

import (
	"bytes"
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"log"

	"github.com/djosix/med/internal/helper"
	"github.com/djosix/med/internal/readwriter"
)

func InitVerify(hash []byte) Initializer {
	return func(ctx context.Context, rw io.ReadWriter) (ctxOut context.Context, rwOut io.ReadWriter, err error) {
		log.Println("init: verify")

		ctxOut = ctx
		rwOut = rw
		err = Verify(rw, hash)

		return
	}
}

func InitGetVerified(hash []byte) Initializer {
	return func(ctx context.Context, rw io.ReadWriter) (ctxOut context.Context, rwOut io.ReadWriter, err error) {
		log.Println("init: get verified")

		ctxOut = ctx
		rwOut = rw
		err = GetVerified(rw, hash)

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
	f := readwriter.NewOrderedFramReadWriter(readwriter.NewPlainFrameReadWriter(rw))

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

func GetVerified(rw io.ReadWriter, hash []byte) error {
	rejectErr := fmt.Errorf("server rejected")
	statusErr := fmt.Errorf("unexpected status")

	f := readwriter.NewOrderedFramReadWriter(readwriter.NewPlainFrameReadWriter(rw))

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
