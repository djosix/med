package helper

import (
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/muesli/cancelreader"
)

var (
	cancelStdinReader cancelreader.CancelReader
	cancelStdin       *os.File
	cancelStdinLock   = sync.Mutex{}
)

func GetCancelStdin() (reader io.Reader, cancel func(), stdin *os.File) {
	cancelStdinLock.Lock()
	defer cancelStdinLock.Unlock()

	if cancelStdin == nil {
		if os.Stdin == nil {
			panic("stdin is already taken")
		}
		cancelStdin = os.Stdin
		os.Stdin = nil
	}

	if cancelStdinReader != nil {
		// logger.Debug("CancelStdin: cancel global reader")
		cancelStdinReader.Cancel()
	}

	// logger.Debug("CancelStdin: create new reader")
	r, err := cancelreader.NewReader(cancelStdin)
	if err != nil {
		panic(fmt.Sprintf("cannot create CancelReader[%v] for stdin: %v", cancelStdinReader, err))
	}

	cancelStdinReader = r

	return r, func() {
		// logger.Debug("CancelStdin: cancel")
		r.Cancel()
	}, cancelStdin
}
