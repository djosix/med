//go:build client
// +build client

package cmd

import (
	"github.com/djosix/med/internal/handler"
	"github.com/spf13/cobra"
)

var clientCmd = &cobra.Command{
	Use:     "med",
	Run:     handler.ClientMain,
	Args:    cobra.ArbitraryArgs,
	PreRunE: handler.CheckClientFlags,
}

func init() {
	handler.InitClientFlags(clientCmd)
}
