package initializer

import (
	"context"
	"io"
)

type Initializer = func(ctx context.Context, rw io.ReadWriter) (context.Context, io.ReadWriter, error)
