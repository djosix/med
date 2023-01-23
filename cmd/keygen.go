package cmd

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"

	"github.com/djosix/med/internal/logger"
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
			fmt.Printf("private: %s\n", hex.EncodeToString(privateKey))
			fmt.Printf("public: %s\n", hex.EncodeToString(publicKey))

			return
		}

		for i, path := range args {
			publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
			if err != nil {
				panic(err)
			}

			privateKeyPath := fmt.Sprintf("%s.key", path)
			fmt.Printf("privateKey[%d] = %s\n", i, privateKeyPath)
			if err := os.WriteFile(privateKeyPath, []byte(hex.EncodeToString(privateKey)+"\n"), 0600); err != nil {
				logger.Print("error:", err)
			}

			publicKeyPath := fmt.Sprintf("%s.pub", path)
			fmt.Printf("publicKey[%d] = %s\n", i, publicKeyPath)
			if err := os.WriteFile(publicKeyPath, []byte(hex.EncodeToString(publicKey)+"\n"), 0622); err != nil {
				logger.Print("error:", err)
			}
		}

	},
}

func init() {
	clientCmd.AddCommand(keygenCmd)
}
