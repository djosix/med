package handler

import (
	"context"
	"io"

	"github.com/djosix/med/internal/initializer"
	"github.com/djosix/med/internal/logger"
)

type Handler = func(ctx context.Context, rw io.ReadWriter) error

func BindInitializers(h Handler, inits ...initializer.Initializer) Handler {
	return func(ctx context.Context, rw io.ReadWriter) (err error) {
		for _, init := range inits {
			ctx, rw, err = init(ctx, rw)
			if err != nil {
				return err
			}
		}
		logger.Log("initialized")
		return h(ctx, rw)
	}
}
