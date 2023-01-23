/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"context"

	"github.com/djosix/med/internal/logger"
	"github.com/spf13/cobra"
)

// testCmd represents the test command
var testCmd = &cobra.Command{
	Use: "test",
	Run: func(cmd *cobra.Command, args []string) {
		if err := mainForTest(cmd, args); err != nil {
			logger.Log("error:", err)
		}
	},
}

func init() {
	clientCmd.AddCommand(testCmd)
}

type MyReader struct {
	Ctx    context.Context
	Cancel context.CancelFunc
	idx    int
}

func NewMyReader() *MyReader {
	r := MyReader{}
	return &r
}

func (r *MyReader) Read(p []byte) (int, error) {
	<-r.Ctx.Done()
	for i := range p {
		p[i] = byte(r.idx % 256)
		r.idx++
	}
	return len(p), nil
}

func mainForTest(cmd *cobra.Command, args []string) error {
	_, _ = cmd, args
	// ctx := context.Background()

	// r := NewMyReader()
	// r.Ctx, r.Cancel = context.WithCancel(ctx)

	// brCtx, brCancel := context.WithCancel(context.Background())
	// br := helper.NewBreakableReader(brCtx, r, 4)
	// _ = brCancel

	// go func() {
	// 	logger.Log("wait")
	// 	time.Sleep(1 * time.Second)
	// 	logger.Log("br.BreakRead()")
	// 	br.BreakRead()
	// 	logger.Log("wait")
	// 	time.Sleep(1 * time.Second)
	// 	logger.Log("br.BreakRead()")
	// 	br.BreakRead()
	// }()

	// buf := make([]byte, 32)

	// testRead := func() {
	// 	if n, err := br.Read(buf); true {
	// 		logger.Log(fmt.Sprintf("br.Read => n=%v, err=%v", buf[:n], err))
	// 	}
	// }
	// testRead()
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
	// 	logger.Log("done")
	// default:
	// 	logger.Log("default")
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
	// 	logger.Log("error:", err)
	// 	return
	// }
	// logger.Log("Message:", data)
	// data := []byte{}
	// for len(data) < 1000 {
	// 	data = append(data, byte(len(data)%256))
	// }

	// encoded := []byte{}
	// encoded = snappy.Encode(encoded, data)

	// logger.Log("src:", len(data))
	// logger.Log("dst:", len(encoded))

	return nil
}
