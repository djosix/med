package handler

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/djosix/med/internal"
	"github.com/djosix/med/internal/helper"
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

	procKind, procSpec, err := determineProcByArgs(opts.Args, opts.IsTTY)
	if err != nil {
		return fmt.Errorf("parse args: %w", err)
	}

	mainProc := worker.NewMainProcClient(worker.MainSpec{ExitWhenNoProc: true})
	mainProc.StartProc(procKind, procSpec)

	loop := worker.NewLoop(ctx, rwc)
	loop.Start(mainProc)
	loop.Run()

	return nil
}

func determineProcByArgs(args []string, tty bool) (kind worker.ProcKind, spec any, err error) {
	if len(args) == 0 {
		args = []string{"exec"}
	}
	cmd := args[0]
	args = args[1:]

	switch cmd {
	case "exec", "x": // exec [command] [args]
		spec := worker.ExecSpec{TTY: tty, ARGV: args}
		if len(args) == 0 {
			spec.ARGV = []string{"/usr/bin/env", "bash"}
		}
		return worker.ProcKind_Exec, spec, nil
	case "get", "put":
		return parseGetPutArgs(cmd, args)
	case "forward":
		return parseForwardArgs(args, true)
	case "local", "remote":
		return parseForwardArgs(append([]string{cmd}, args...), true)
	case "proxy":
	case "socks":
		return parseSocksArgs(args)
	case "self":
		spec, err := parseSelfArgs(args)
		return worker.ProcKind_Self, spec, err
	case "webui":
	case "tmux":
	}

	return worker.ProcKind_None, nil, fmt.Errorf("unknown command: %s", cmd)
}

func parseGetPutArgs(action string, args []string) (kind worker.ProcKind, spec worker.GetPutSpec, err error) {
	flags := flag.NewFlagSet("get", flag.ExitOnError)
	flags.BoolVar(&spec.IsGzipMode, "z", false, "enable gzip compression")
	flags.BoolVar(&spec.IsTarMode, "t", false, "enable tar mode")
	flags.StringVar(&spec.DestPath, "d", "", "destination")
	if err = flags.Parse(args); err != nil {
		return
	}
	spec.SourcePaths = flags.Args()
	logger.Debugf("parseGetPutArgs: spec = %#v", spec)
	switch action {
	case "get":
		kind = worker.ProcKind_Get
	case "put":
		kind = worker.ProcKind_Put
	default:
		panic(action)
	}
	return
}

func parseSelfArgs(args []string) (worker.SelfSpec, error) {
	spec := worker.SelfSpec{}
	args = helper.NewSet(args...).Values()
	if len(args) == 0 {
		return spec, fmt.Errorf("need some actions for 'self' command like: exit, remove")
	}
	for _, arg := range args {
		switch arg {
		case "exit":
			spec.Exit = true
		case "remove":
			spec.Remove = true
		default:
			return spec, fmt.Errorf("unknown action for 'self' command: %v", arg)
		}
	}
	return spec, nil
}

func parseForwardArgs(args []string, isLocal bool) (kind worker.ProcKind, spec worker.ForwardSpec, err error) {
	const usage = "expect args: {local|remote} <ListenEndpoint> <ConnectEndpoint>"
	kind = worker.ProcKind_None
	if len(args) != 3 {
		err = fmt.Errorf(usage)
		return
	}
	switch args[0] {
	case "local":
		kind = worker.ProcKind_LocalPF
	case "remote":
		kind = worker.ProcKind_RemotePF
	default:
		err = fmt.Errorf(usage)
		return
	}
	spec.ListenEndpoint = args[1]
	spec.ConnectEndpoint = args[2]
	return
}

func parseSocksArgs(args []string) (kind worker.ProcKind, spec worker.SocksSpec, err error) {
	const usage = "expect args: <ListenEndpoint>"
	if len(args) != 1 {
		kind = worker.ProcKind_None
		err = fmt.Errorf(usage)
		return
	}
	kind = worker.ProcKind_Socks
	spec.ListenEndpoint = args[0]
	return
}
