package handler

import (
	"context"
	"io"

	"github.com/djosix/med/internal"
	"github.com/djosix/med/internal/initializer"
	"github.com/djosix/med/internal/logger"
	"github.com/djosix/med/internal/worker"

	"github.com/spf13/cobra"
)

const ()

func InitServerFlags(cmd *cobra.Command) {
	flags := cmd.Flags()
	flags.SortFlags = false

	InitCommonFlags(cmd)
}

func CheckServerFlags(cmd *cobra.Command, args []string) (err error) {
	err = CheckCommonFlags(cmd, args)
	if err != nil {
		return err
	}
	return
}

type ServerOpts struct {
	*CommonOpts
}

func GetServerOpts(cmd *cobra.Command, args []string) (*ServerOpts, error) {
	opts := ServerOpts{}

	var err error
	if opts.CommonOpts, err = GetCommonOpts(cmd, args); err != nil {
		return nil, err
	}

	return &opts, nil
}

func ServerMain(cmd *cobra.Command, args []string) {
	logger := logger.NewLogger("server")
	logger.Debug("start")
	defer logger.Debug("done")

	opts, err := GetServerOpts(cmd, args)
	if err != nil {
		logger.Error("options:", err)
		return
	}

	err = ServerStart(cmd.Context(), opts)
	if err != nil {
		logger.Error(err)
		return
	}

}

func ServerStart(ctx context.Context, opts *ServerOpts) error {
	handler := ServerHandler
	{
		// inits := []initializer.Initializer{
		// 	initializer.InitCheckMagic(initializer.ServerMagic, initializer.ClientMagic),
		// 	initializer.InitVerify(opts.PasswordHash),
		// }
		inits := []initializer.Initializer{}
		if opts.SecretHash != nil {
			inits = append(inits, initializer.InitEncryption(opts.SecretHash))
		} else if !opts.UseRaw {
			inits = append(inits, initializer.InitHandshake(opts.PrivateKey, opts.TrustedPublicKeys))
			inits = append(inits, initializer.InitEncryption(nil))
		}
		inits = append(inits, initializer.InitVerify(opts.PasswordHash))
		handler = BindInitializers(handler, inits...)
	}

	switch opts.Mode {
	case CommonFlagConnect:
		return Connect(ctx, opts.Endpoint, handler, opts.ConnDelayIfConnect)

	case CommonFlagListen:
		return Listen(ctx, opts.Endpoint, handler, opts.MaxConnIfListen)

	default:
		panic(internal.Unexpected)
	}
}

func ServerHandler(ctx context.Context, rwc io.ReadWriteCloser) error {
	logger := logger.NewLogger("server/handler")
	logger.Debug("start")
	defer logger.Debug("done")

	loop := worker.NewLoop(ctx, rwc)
	loop.Start(worker.NewMainProcServer())
	loop.Run()

	return nil
}
