package cmd

import (
	"fmt"

	"github.com/djosix/med/internal/logger"
	"github.com/spf13/cobra"
)

// DevCmd represents the test command
var DevCmd = &cobra.Command{
	Use: "dev",
	Run: func(cmd *cobra.Command, args []string) {
		logger.Print("error:", devMain(cmd, args))
	},
}

func devMain(cmd *cobra.Command, args []string) error {
	_, _ = cmd, args

	fmt.Println(logger.GetFirstNonLoggerCaller())

	// b := make([]byte, 1024)
	// fmt.Println(len(b))
	// fmt.Println(cap(b))
	// fmt.Println(len(b[100:]))
	// fmt.Println(cap(b[100:]))
	// fmt.Println(len(b[:200]))
	// fmt.Println(cap(b[:200]))
	// fmt.Println(len(b[100:200]))
	// fmt.Println(cap(b[100:200]))

	// r.Cancel()
	// r.Ctx, r.Cancel = context.WithCancel(ctx)
	// testRead()
	// r.Cancel()
	// testRead()
	// testRead()

	// done := make(chan struct{})
	// close(done)
	// close(done)

	// select {
	// case <-done:
	// 	logger.Show("done")
	// default:
	// 	logger.Show("default")
	// }

	// // Create arbitrary command.
	// c := exec.Command("bash")

	// // Start the command with a pty.
	// ptmx, err := pty.Start(c)
	// if err != nil {
	// 	return err
	// }
	// // Make sure to close the pty at the end.
	// defer func() { _ = ptmx.Close() }() // Best effort.

	// // Handle pty size.
	// ch := make(chan os.Signal, 1)
	// signal.Notify(ch, syscall.SIGWINCH)
	// go func() {
	// 	for range ch {
	// 		if err := pty.InheritSize(os.Stdin, ptmx); err != nil {
	// 			log.Printf("error resizing pty: %s", err)
	// 		}
	// 	}
	// }()
	// ch <- syscall.SIGWINCH                        // Initial resize.
	// defer func() { signal.Stop(ch); close(ch) }() // Cleanup signals when done.

	// // Set stdin in raw mode.
	// oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	// if err != nil {
	// 	panic(err)
	// }
	// defer func() { _ = term.Restore(int(os.Stdin.Fd()), oldState) }() // Best effort.

	// // Copy stdin to the pty and the pty to stdout.
	// // NOTE: The goroutine will keep reading until the next keystroke before returning.
	// go func() { _, _ = io.Copy(ptmx, os.Stdin) }()
	// _, _ = io.Copy(os.Stdout, ptmx)

	// return nil

	// message := pb.Message{
	// 	Content: &pb.Message_C1{
	// 		C1: &pb.Content1{
	// 			// Name: "test",
	// 		},
	// 	},
	// }
	// data, err := proto.Marshal(&message)
	// if err != nil {
	// 	logger.Show("error:", err)
	// 	return
	// }
	// logger.Show("Message:", data)
	// data := []byte{}
	// for len(data) < 1000 {
	// 	data = append(data, byte(len(data)%256))
	// }

	// encoded := []byte{}
	// encoded = snappy.Encode(encoded, data)

	// logger.Show("src:", len(data))
	// logger.Show("dst:", len(encoded))

	return nil
}
