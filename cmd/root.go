package cmd

import (
	"os"
)

func Execute() {
	err := clientCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}
