package handler

import (
	"context"
	"fmt"
	"io"
	"log"

	"github.com/djosix/med/internal"
	"github.com/djosix/med/internal/initializer"
	"github.com/spf13/cobra"
)

const (
	ClientFlagExec  = "exec"
	ClientFlagShell = "shell"
	ClientFlagMenu  = "menu"
)

func InitClientFlags(cmd *cobra.Command) {
	flags := cmd.Flags()
	flags.SortFlags = false

	flags.StringP(ClientFlagExec, "x", "", "execute command")
	flags.StringP(ClientFlagShell, "z", "", "shell to execute command")
	flags.BoolP(ClientFlagMenu, "m", false, "display menu")

	InitCommonFlags(cmd)
}

func CheckClientFlags(cmd *cobra.Command, args []string) error {
	return CheckCommonFlags(cmd, args)
}

type ClientOpts struct {
	CommonOpts
	Exec     string
	Shell    string
	ShowMenu bool
}

func GetClientOpts(cmd *cobra.Command, args []string) (*ClientOpts, error) {
	flags := cmd.Flags()

	var err error
	opts := ClientOpts{}

	if cOpts, err := GetCommonOpts(cmd, args); err != nil {
		return nil, err
	} else {
		opts.CommonOpts = *cOpts
	}

	if opts.Exec, err = flags.GetString(ClientFlagExec); err != nil {
		return nil, err
	}

	if opts.Shell, err = flags.GetString(ClientFlagShell); err != nil {
		return nil, err
	}

	if opts.ShowMenu, err = flags.GetBool(ClientFlagMenu); err != nil {
		return nil, err
	}

	return &opts, nil
}

func ClientMain(cmd *cobra.Command, args []string) {
	opts, err := GetClientOpts(cmd, args)
	if err != nil {
		log.Fatalln("error:", err)
		return
	}

	err = ClientStart(cmd.Context(), opts)
	if err != nil {
		log.Fatalln("error:", err)
		return
	}

	log.Println("done")
}

func ClientStart(ctx context.Context, opts *ClientOpts) error {
	log.Println("ClientStart")

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

	switch opts.Mode {
	case CommonFlagConnect:
		log.Println(CommonFlagConnect, opts.Endpoint)
		return Connect(ctx, opts.Endpoint, handler)

	case CommonFlagListen:
		log.Println(CommonFlagListen, opts.Endpoint)
		return Listen(ctx, opts.Endpoint, handler, 1)

	default:
		panic(internal.UnexpectedErr)
	}
}

func ClientHandler(ctx context.Context, rw io.ReadWriter) error {
	log.Println("ClientHandler")

	_ = ctx

	data := []byte("ClientMessage\n")
	log.Println("Write:", string(data))
	n, err := rw.Write(data)
	if err != nil {
		return err
	}
	if n != len(data) {
		return fmt.Errorf("cannot write full")
	}

	buf := make([]byte, 1024)
	log.Println("read")
	n, err = rw.Read(buf)
	log.Println("read done")
	if err != nil {
		return err
	}
	log.Println("Read", n, err, string(buf))

	return nil
}
