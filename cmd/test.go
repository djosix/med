/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"github.com/djosix/med/internal/logger"
	"github.com/spf13/cobra"
)

// testCmd represents the test command
var testCmd = &cobra.Command{
	Use: "test",
	Run: func(cmd *cobra.Command, args []string) {
		if err := mainForTest(cmd, args); err != nil {
			logger.Print("error:", err)
		}
	},
}

func init() {
	clientCmd.AddCommand(testCmd)
}

func mainForTest(cmd *cobra.Command, args []string) error {
	_, _ = cmd, args

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
