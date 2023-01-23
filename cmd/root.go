package cmd

import (
	"os"

	"github.com/djosix/med/internal/logger"
)

func Execute() {
	logger.SetLevel(logger.LevelDebug)
	err := clientCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}
