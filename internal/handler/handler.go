package handler

import (
	"context"
	"io"

	"github.com/djosix/med/internal/initializer"
	"github.com/djosix/med/internal/logger"
)

type Handler = func(ctx context.Context, rwc io.ReadWriteCloser) error

func BindInitializers(h Handler, inits ...initializer.Initializer) Handler {
	return func(ctx context.Context, rwc io.ReadWriteCloser) (err error) {
		for _, init := range inits {
			ctx, rwc, err = init(ctx, rwc)
			if err != nil {
				return err
			}
		}
		logger.Debug("Init: Done")
		return h(ctx, rwc)
	}
}
