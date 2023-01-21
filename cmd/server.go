/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"github.com/djosix/med/internal/handler"
	"github.com/spf13/cobra"
)

var serverCmd = &cobra.Command{
	Use:     "server",
	Short:   "Start a med server",
	Run:     handler.ServerMain,
	PreRunE: handler.CheckServerFlags,
}

func init() {
	clientCmd.AddCommand(serverCmd)
	handler.InitServerFlags(serverCmd)
}
