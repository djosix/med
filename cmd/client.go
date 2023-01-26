package cmd

import (
	"github.com/djosix/med/internal/handler"
	"github.com/spf13/cobra"
)

var ClientCmd = &cobra.Command{
	Use:     "client {-c endpoint|-l endpoint} [flags] [args]",
	Run:     handler.ClientMain,
	Args:    cobra.ArbitraryArgs,
	PreRunE: handler.CheckClientFlags,
}

func init() {
	handler.InitClientFlags(ClientCmd)
}
