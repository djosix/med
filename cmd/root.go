package cmd

import (
	"os"
)

func Execute() {
	// logger.SetLevel(logger.LevelDebug)
	err := clientCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}
