package handler

import (
	"context"
	"io"
	"log"
	"time"

	"github.com/djosix/med/internal"
	"github.com/djosix/med/internal/initializer"

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
		log.Fatalln("error:", err)
		return
	}

	err = ServerStart(cmd.Context(), opts)
	if err != nil {
		log.Fatalln("error:", err)
		return
	}

	log.Println("done")
}

func ServerStart(ctx context.Context, opts *ServerOpts) error {
	log.Println("ServerStart")
	log.Printf("%#v\n", opts)

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
		log.Println(CommonFlagConnect, opts.Endpoint)
		return Connect(ctx, opts.Endpoint, handler)

	case CommonFlagListen:
		log.Println(CommonFlagListen, opts.Endpoint)
		return Listen(ctx, opts.Endpoint, handler, opts.MaxConnIfListen)

	default:
		panic(internal.Unexpected)
	}
}

func ServerHandler(ctx context.Context, rw io.ReadWriter) error {

	// handlers := map[uint32]MsgHandler{}
	// return RunMsgLoop(ctx, rw, handlers)

	_ = ctx

	time.Sleep(time.Duration(2) * time.Second)

	data := []byte("ServerMessage\n")
	log.Println("Write:", string(data))
	n, err := rw.Write(data)
	if err != nil {
		return err
	}

	buf := make([]byte, 1024)
	n, err = rw.Read(buf)
	if err != nil {
		return err
	}
	log.Println("Read:", n, err, string(buf))
	log.Println("End")

	return nil
}
