//go:build client && server
// +build client,server

package cmd

var RootCmd = clientCmd

func init() {
	RootCmd.AddCommand(serverCmd)
}
