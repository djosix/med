package handler

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/djosix/med/internal"
	"github.com/djosix/med/internal/helper"
	"github.com/djosix/med/internal/initializer"
	"github.com/djosix/med/internal/logger"
	"github.com/djosix/med/internal/readwriter"
	"github.com/djosix/med/internal/worker"
	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"
)

// const (
// 	ClientFlagExec  = "exec"
// 	ClientFlagShell = "shell"
// 	ClientFlagMenu  = "menu"
// )

func InitClientFlags(cmd *cobra.Command) {
	flags := cmd.Flags()
	flags.SortFlags = false

	// flags.StringP(ClientFlagExec, "x", "", "execute command")
	// flags.StringP(ClientFlagShell, "z", "", "shell to execute command")
	// flags.BoolP(ClientFlagMenu, "m", false, "display menu")

	InitCommonFlags(cmd)
}

func CheckClientFlags(cmd *cobra.Command, args []string) error {
	return CheckCommonFlags(cmd, args)
}

type ClientOpts struct {
	CommonOpts
	SubArgs []string
}

func GetClientOpts(cmd *cobra.Command, args []string) (*ClientOpts, error) {
	// flags := cmd.Flags()

	// var err error
	opts := ClientOpts{}

	if cOpts, err := GetCommonOpts(cmd, args); err != nil {
		return nil, err
	} else {
		opts.CommonOpts = *cOpts
	}

	opts.SubArgs = args

	// if opts.Exec, err = flags.GetString(ClientFlagExec); err != nil {
	// 	return nil, err
	// }

	// if opts.Shell, err = flags.GetString(ClientFlagShell); err != nil {
	// 	return nil, err
	// }

	// if opts.ShowMenu, err = flags.GetBool(ClientFlagMenu); err != nil {
	// 	return nil, err
	// }

	return &opts, nil
}

func ClientMain(cmd *cobra.Command, args []string) {
	logger := logger.NewLogger("ClientMain")
	defer logger.Debug("Done")

	opts, err := GetClientOpts(cmd, args)
	if err != nil {
		logger.Error("GetClientOpts:", err)
		return
	}

	err = ClientStart(cmd.Context(), opts)
	if err != nil {
		logger.Error("ClientStart:", err)
		return
	}

}

func ClientStart(ctx context.Context, opts *ClientOpts) error {
	handler := ClientHandler
	{
		inits := []initializer.Initializer{
			initializer.InitCheckMagic(initializer.ClientMagic, initializer.ServerMagic),
			initializer.InitGetVerified(opts.PasswordHash),
		}
		if opts.SecretHash != nil {
			inits = append(inits, initializer.InitEncryption(opts.SecretHash))
		} else if !opts.UseRaw {
			inits = append(inits, initializer.InitHandshake(opts.PrivateKey, opts.TrustedPublicKeys))
			inits = append(inits, initializer.InitEncryption(nil))
		}
		handler = BindInitializers(handler, inits...)
	}

	ctx = context.WithValue(ctx, "opts", opts)

	switch opts.Mode {
	case CommonFlagConnect:
		return Connect(ctx, opts.Endpoint, handler)
	case CommonFlagListen:
		return Listen(ctx, opts.Endpoint, handler, 1)
	default:
		panic(internal.Unexpected)
	}
}

func ClientHandler(ctx context.Context, rw io.ReadWriter) error {
	logger := logger.NewLogger("ClientHandler")
	logger.Debug("start")
	defer logger.Debug("done")

	opts, ok := ctx.Value("opts").(*ClientOpts)
	if !ok {
		panic("cannot get opts from ctx")
	}

	// if len(opts.SubArgs) == 0 {

	// }
	argv := []string{"bash"}
	if len(opts.SubArgs) != 0 {
		argv = opts.SubArgs
	}
	fmt.Println("args:", opts.SubArgs)
	// return nil

	// rootProc := worker.NewExampleProc("message from client")
	// rootProc := worker.NewMainProcClient()
	rootProc := worker.NewExecProcClient(worker.ExecSpec{
		ARGV: argv,
		TTY:  isatty.IsTerminal(os.Stdin.Fd()),
	})

	// Send root proc kind to server
	{
		frameRw := readwriter.NewPlainFrameReadWriter(rw)
		frame := helper.MustEncode(rootProc.Kind())
		if err := frameRw.WriteFrame(frame); err != nil {
			return fmt.Errorf("cannot write root proc kind: %w", err)
		}
	}

	loop := worker.NewLoop(ctx, rw)
	loop.Start(rootProc)
	loop.Run()

	return nil
}
