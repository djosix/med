package initializer

import (
	"context"
	"io"

	"github.com/djosix/med/internal/logger"
)

type Initializer = func(ctx context.Context, rwc io.ReadWriteCloser) (context.Context, io.ReadWriteCloser, error)

var initLogger = logger.NewLogger("initialize")
