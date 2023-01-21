package cmd

import (
	"github.com/djosix/med/internal/handler"
	"github.com/spf13/cobra"
)

var clientCmd = &cobra.Command{
	Use:     "med",
	Run:     handler.ClientMain,
	PreRunE: handler.CheckClientFlags,
}

func init() {
	handler.InitClientFlags(clientCmd)
}
