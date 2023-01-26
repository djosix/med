package main

import (
	"os"
	"strings"

	"github.com/djosix/med/cmd"
	"github.com/spf13/cobra"
)

var RootCmd = cmd.ClientCmd

func init() {
	RootCmd.Use = "med " + strings.SplitN(RootCmd.Use, " ", 2)[1]
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
