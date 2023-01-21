package helper

import (
	"crypto/sha256"
	"fmt"
	"syscall"

	"golang.org/x/term"
)

func Hash256(data []byte) []byte {
	salt := [4]byte{0x3a, 0x4d, 0x45, 0x44}
	return HashSalt256(data, salt[:])
}

func HashSalt256(data []byte, salt []byte) []byte {
	hasher := sha256.New()
	hasher.Write(data)
	hasher.Write(salt)
	return hasher.Sum(nil)
}

func PromptHiddenInput(msg string) (input []byte, err error) {
	fmt.Print(msg)
	fd := int(syscall.Stdin)
	return term.ReadPassword(fd)
}
