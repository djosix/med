package handler

import (
	"context"
	"io"
	"time"

	"github.com/djosix/med/internal"
	"github.com/djosix/med/internal/initializer"
	"github.com/djosix/med/internal/logger"
	"github.com/djosix/med/internal/worker"

	"github.com/spf13/cobra"
)

const (
	ServerFlagDaemon   = "daemon"
	ServerFlagInterval = "interval"
)

func InitServerFlags(cmd *cobra.Command) {
	flags := cmd.Flags()
	flags.SortFlags = false

	flags.BoolP(ServerFlagDaemon, "d", false, "run as a daemon")
	flags.DurationP(ServerFlagInterval, "i", time.Duration(0)*time.Second, "with --connect, connect back interval, 0s for no loop")

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
	CommonOpts
	IsDaemon bool
	Interval time.Duration
}

func GetServerOpts(cmd *cobra.Command, args []string) (*ServerOpts, error) {
	flags := cmd.Flags()

	var err error
	opts := ServerOpts{}

	if cOpts, err := GetCommonOpts(cmd, args); err != nil {
		return nil, err
	} else {
		opts.CommonOpts = *cOpts
	}

	if opts.IsDaemon, err = flags.GetBool(ServerFlagDaemon); err != nil {
		return nil, err
	}

	if opts.Interval, err = flags.GetDuration(ServerFlagInterval); err != nil {
		return nil, err
	}

	return &opts, nil
}

func ServerMain(cmd *cobra.Command, args []string) {
	opts, err := GetServerOpts(cmd, args)
	if err != nil {
		logger.Print("error:", err)
		return
	}

	err = ServerStart(cmd.Context(), opts)
	if err != nil {
		logger.Print("error:", err)
		return
	}

	logger.Print("done")
}

func ServerStart(ctx context.Context, opts *ServerOpts) error {
	handler := ServerHandler
	{
		inits := []initializer.Initializer{
			initializer.InitCheckMagic(initializer.ServerMagic, initializer.ClientMagic),
			initializer.InitVerify(opts.PasswordHash),
		}
		if opts.SecretHash != nil {
			inits = append(inits, initializer.InitEncryption(opts.SecretHash))
		} else if !opts.UseRaw {
			inits = append(inits, initializer.InitHandshake(opts.PrivateKey, opts.TrustedPublicKeys))
			inits = append(inits, initializer.InitEncryption(nil))
		}
		handler = BindInitializers(handler, inits...)
	}

	switch opts.Mode {
	case CommonFlagConnect:
		logger.Print(CommonFlagConnect, opts.Endpoint)
		return Connect(ctx, opts.Endpoint, handler)

	case CommonFlagListen:
		logger.Print(CommonFlagListen, opts.Endpoint)
		return Listen(ctx, opts.Endpoint, handler, opts.MaxConnIfListen)

	default:
		panic(internal.Unexpected)
	}
}

func ServerHandler(ctx context.Context, rw io.ReadWriter) error {
	logger.Print("ServerHandler BEGIN")
	defer logger.Print("ServerHandler END")

	var loop worker.Loop = worker.NewLoop(ctx, rw)
	// loop.Start(worker.NewExampleProc("message from server"))
	loop.Start(worker.NewServerExecProc())
	loop.Run()

	{
		buf := []byte("server loop closed")
		if _, err := rw.Write(buf); err == nil {
			if n, err := rw.Read(buf); err == nil {
				logger.Print("remote:", string(buf[:n]))
			}
		}
	}

	return nil
}
