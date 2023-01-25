//go:build keygen
// +build keygen

package cmd

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var keygenCmd = &cobra.Command{
	Use:   "keygen",
	Short: "Generate med key pairs",
	Run: func(cmd *cobra.Command, args []string) {
		_, _ = cmd, args

		if len(args) == 0 {
			publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
			if err != nil {
				panic(err)
			}
			fmt.Printf("key = %s\n", hex.EncodeToString(privateKey))
			fmt.Printf("pub = %s\n", hex.EncodeToString(publicKey))

			return
		}

		for i, path := range args {
			pub, key, err := ed25519.GenerateKey(rand.Reader)
			if err != nil {
				panic(err)
			}

			{
				keyPath := fmt.Sprintf("%s.key", path)
				data := hex.EncodeToString(key) + "\n"
				err := os.WriteFile(keyPath, []byte(data), 0600)
				if err != nil {
					fmt.Println("error:", err)
					return
				}
				fmt.Printf("key[%d] = %s\n", i, keyPath)
			}

			{
				pubPath := fmt.Sprintf("%s.pub", path)
				data := hex.EncodeToString(pub) + "\n"
				err := os.WriteFile(pubPath, []byte(data), 0622)
				if err != nil {
					fmt.Printf("pub[%d] = %s\n", i, pubPath)
					return
				}
				fmt.Printf("pub[%d] = %s\n", i, pubPath)
			}
		}
	},
}

func init() {
	RootCmd.AddCommand(keygenCmd)
}
