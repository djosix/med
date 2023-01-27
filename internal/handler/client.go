package handler

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/djosix/med/internal"
	"github.com/djosix/med/internal/initializer"
	"github.com/djosix/med/internal/logger"
	"github.com/djosix/med/internal/worker"
	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"
)

const (
	ClientFlagNoTTY = "notty"
)

func InitClientFlags(cmd *cobra.Command) {
	flags := cmd.Flags()
	flags.SortFlags = false
	flags.BoolP(ClientFlagNoTTY, "T", false, "disable tty")
	InitCommonFlags(cmd)
}

func CheckClientFlags(cmd *cobra.Command, args []string) error {
	return CheckCommonFlags(cmd, args)
}

type ClientOpts struct {
	*CommonOpts
	IsTTY bool
	Args  []string
}

func GetClientOpts(cmd *cobra.Command, args []string) (*ClientOpts, error) {
	flags := cmd.Flags()
	opts := ClientOpts{Args: args}

	var err error
	if opts.CommonOpts, err = GetCommonOpts(cmd, args); err != nil {
		return nil, err
	}

	if noTTY, err := flags.GetBool(ClientFlagNoTTY); err != nil {
		return nil, err
	} else if noTTY {
		opts.IsTTY = false
	} else {
		opts.IsTTY = isatty.IsTerminal(os.Stdin.Fd())
	}

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
		return Connect(ctx, opts.Endpoint, handler, opts.ConnDelayIfConnect)

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

	procKind, procSpec, err := DetermineProc(opts.Args, opts.IsTTY)
	if err != nil {
		return err
	}

	mainProc := worker.NewMainProcClient(worker.MainSpec{
		ExitWhenNoProc: true,
	})
	mainProc.StartProc(procKind, procSpec)

	loop := worker.NewLoop(ctx, rw)
	loop.Start(mainProc)
	loop.Run()

	return nil
}

func DetermineProc(args []string, tty bool) (kind worker.ProcKind, spec any, err error) {
	defaultArgv := []string{"/bin/sh"}
	if len(args) == 0 {
		args = append([]string{"exec"}, defaultArgv...)
	}

	switch args[0] {
	case "exec":
		spec := worker.ExecSpec{TTY: tty, ARGV: args[1:]}
		if len(spec.ARGV) == 0 {
			spec.ARGV = defaultArgv
		}
		return worker.ProcKind_Exec, spec, nil
	case "get":
	case "put":
	case "forward":
	case "socks5":
	case "proxy":
	case "self":
	case "webui":
	case "tmux":
	}

	return worker.ProcKind_None, nil, fmt.Errorf("cannot determine root proc")
}
