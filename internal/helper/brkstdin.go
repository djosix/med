package helper

import (
	"os"
	"sync"
)

var breakableStdin BreakableReader
var onceCreateBreakableStdin sync.Once

func GetBreakableStdin() BreakableReader {
	onceCreateBreakableStdin.Do(func() {
		breakableStdin = NewBreakableReader(os.Stdin, 1024)
	})
	return breakableStdin
}
