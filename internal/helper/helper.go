package helper

import (
	"crypto/sha256"
	"fmt"
	"syscall"

	"github.com/djosix/med/internal"
	"golang.org/x/term"
)

func Hash256(data []byte) []byte {
	return HashSalt256(data, []byte{})
}

func HashSalt256(data []byte, salt []byte) []byte {
	hasher := sha256.New()
	hasher.Write(data)
	hasher.Write(internal.Nonce)
	hasher.Write(salt)
	return hasher.Sum(nil)
}

func PromptHiddenInput(message string) (input []byte, err error) {
	fmt.Print(message)
	fd := int(syscall.Stdin)
	return term.ReadPassword(fd)
}
