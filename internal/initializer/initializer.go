package initializer

import (
	"context"
	"io"

	"github.com/djosix/med/internal/logger"
)

type Initializer = func(ctx context.Context, rw io.ReadWriter) (context.Context, io.ReadWriter, error)

var initLogger = logger.NewLogger("Init")
