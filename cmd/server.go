package cmd

import (
	"github.com/djosix/med/internal/handler"
	"github.com/spf13/cobra"
)

var ServerCmd = &cobra.Command{
	Use:     "server",
	Short:   "Start a med server",
	Run:     handler.ServerMain,
	PreRunE: handler.CheckServerFlags,
}

func init() {
	handler.InitServerFlags(ServerCmd)
}
