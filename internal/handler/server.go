package handler

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/djosix/med/internal"
	"github.com/djosix/med/internal/helper"
	"github.com/djosix/med/internal/initializer"
	"github.com/djosix/med/internal/logger"
	"github.com/djosix/med/internal/readwriter"
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
	logger := logger.NewLogger("ServerMain")
	defer logger.Debug("Done")

	opts, err := GetServerOpts(cmd, args)
	if err != nil {
		logger.Error("GetServerOpts:", err)
		return
	}

	err = ServerStart(cmd.Context(), opts)
	if err != nil {
		logger.Error("ServerStart:", err)
		return
	}

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
		return Connect(ctx, opts.Endpoint, handler)

	case CommonFlagListen:
		return Listen(ctx, opts.Endpoint, handler, opts.MaxConnIfListen)

	default:
		panic(internal.Unexpected)
	}
}

func ServerHandler(ctx context.Context, rw io.ReadWriter) error {
	logger := logger.NewLogger("ServerHandler")
	logger.Debug("start")
	defer logger.Debug("done")

	var rootProcKind worker.ProcKind
	{
		frame, err := readwriter.NewPlainFrameReadWriter(rw).ReadFrame()
		if err != nil {
			return fmt.Errorf("cannot read root proc kind: %w", err)
		}
		kind, err := helper.DecodeAs[worker.ProcKind](frame)
		if err != nil {
			return fmt.Errorf("cannot decode root proc kind: %w", err)
		}
		rootProcKind = kind
	}

	loop := worker.NewLoop(ctx, rw)
	loop.Start(worker.CreateProcServer(rootProcKind))
	loop.Run()

	return nil
}
