package main

import (
	"os"

	"github.com/djosix/med/cmd"
	"github.com/spf13/cobra"
)

var RootCmd = cmd.ClientCmd

func init() {
	RootCmd.AddCommand(cmd.ServerCmd)
	RootCmd.AddCommand(cmd.KeygenCmd)
	RootCmd.AddCommand(cmd.DevCmd)
	RootCmd.CompletionOptions = cobra.CompletionOptions{
		DisableDefaultCmd: true,
	}
}

func main() {

	if RootCmd.Execute() != nil {
		os.Exit(1)
	}
}
