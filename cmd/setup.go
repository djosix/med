package cmd

import (
	"github.com/djosix/med/internal/logger"
	"github.com/spf13/cobra"
)

func init() {
	logger.SetLevel(logger.LevelInfo)
	// logger.SetAddPrefix(false)

	RootCmd.CompletionOptions = cobra.CompletionOptions{
		DisableDefaultCmd: true,
	}
}
