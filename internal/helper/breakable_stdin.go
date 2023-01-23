package helper

import (
	"os"
	"sync"
)

var breakableStdinReader BreakableReader
var originalStdin *os.File
var onceCreateBreakableStdin sync.Once

// GetBreakableStdin takes the ownership of stdin
func GetBreakableStdin() (BreakableReader, *os.File) {
	onceCreateBreakableStdin.Do(func() {
		bufSize := 1024
		breakableStdinReader = NewBreakableReader(os.Stdin, bufSize)
		originalStdin = os.Stdin
		os.Stdin = nil
	})
	return breakableStdinReader, originalStdin
}
