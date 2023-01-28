package handler

import (
	"context"
	"flag"
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
	logger.Debug("start")
	defer logger.Debug("done")

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

func ClientHandler(ctx context.Context, rwc io.ReadWriteCloser) error {
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

	mainProc := worker.NewMainProcClient(worker.MainSpec{ExitWhenNoProc: true})
	mainProc.StartProc(procKind, procSpec)

	loop := worker.NewLoop(ctx, rwc)
	loop.Start(mainProc)
	loop.Run()

	return nil
}

func DetermineProc(args []string, tty bool) (kind worker.ProcKind, spec any, err error) {
	defaultArgv := []string{"/bin/sh"}
	if len(args) == 0 {
		args = append([]string{"exec"}, defaultArgv...)
	}

	action := args[0]
	args = args[1:]

	switch action {
	case "exec", "x":
		spec := worker.ExecSpec{TTY: tty, ARGV: args}
		if len(spec.ARGV) == 0 {
			spec.ARGV = defaultArgv
		}
		return worker.ProcKind_Exec, spec, nil
	case "get":
		spec := parseGetPutArgs(args)
		return worker.ProcKind_Get, spec, nil
	case "put":
		spec := parseGetPutArgs(args)
		return worker.ProcKind_Put, spec, nil
	case "forward":
	case "socks5":
	case "proxy":
	case "self":
	case "webui":
	case "tmux":
	}

	return worker.ProcKind_None, nil, fmt.Errorf("cannot determine root proc")
}

func parseGetPutArgs(args []string) worker.GetPutSpec {
	spec := worker.GetPutSpec{}
	flags := flag.NewFlagSet("get", flag.ExitOnError)
	flags.BoolVar(&spec.IsGzipMode, "z", false, "enable gzip compression")
	flags.BoolVar(&spec.IsTarMode, "t", false, "enable tar mode")
	flags.StringVar(&spec.DestPath, "d", "", "destination")
	flags.Parse(args)
	// if flags.
	// flags.Usage()
	spec.SourcePaths = flags.Args()
	logger.Debugf("parseGetPutArgs: spec = %#v", spec)
	return spec
}
