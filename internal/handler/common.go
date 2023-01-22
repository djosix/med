package handler

import (
	"bytes"
	"crypto/ed25519"
	"encoding/hex"
	"fmt"
	"os"

	"github.com/djosix/med/internal"
	"github.com/djosix/med/internal/helper"
	"github.com/djosix/med/internal/initializer"
	"github.com/spf13/cobra"
)

const (
	CommonFlagConnect             = "connect"
	CommonFlagListen              = "listen"
	CommonFlagMaxConn             = "maxConn"
	CommonFlagPassword            = "passwd"
	CommonFlagPasswordPrompt      = "promptPasswd"
	CommonFlagSecret              = "secret"
	CommonFlagSecretPrompt        = "promptSecret"
	CommonFlagTrustedPublicKeyHex = "pub"
	CommonFlagPublicKeyPath       = "pubPath"
	CommonFlagKeyHex              = "key"
	CommonFlagKeyPath             = "keyPath"
	CommonFlagUseRaw              = "useRaw"
)

func InitCommonFlags(cmd *cobra.Command) {
	flags := cmd.Flags()
	flags.SortFlags = false

	flags.StringP(CommonFlagConnect, "c", "", "target to connect")
	flags.StringP(CommonFlagListen, "l", "", "endpoint to listen on")
	flags.IntP(CommonFlagMaxConn, "C", 1, "max number of connections")
	flags.StringP(CommonFlagPassword, "w", "", "the server password")
	flags.BoolP(CommonFlagPasswordPrompt, "W", false, "prompt for the server password")
	flags.StringP(CommonFlagSecret, "s", "", "shared secret")
	flags.BoolP(CommonFlagSecretPrompt, "S", false, "prompt for shared secret")
	flags.StringArrayP(CommonFlagTrustedPublicKeyHex, "p", []string{}, "trusted public key (hex)")
	flags.StringArrayP(CommonFlagPublicKeyPath, "P", []string{}, "trusted public key file path")
	flags.StringP(CommonFlagKeyHex, "k", "", "key (hex)")
	flags.StringP(CommonFlagKeyPath, "K", "", "key file path")
	flags.BoolP(CommonFlagUseRaw, "r", false, "disable encryption")

	cmd.MarkFlagsMutuallyExclusive(CommonFlagConnect, CommonFlagListen)
	cmd.MarkFlagsMutuallyExclusive(CommonFlagPassword, CommonFlagPasswordPrompt)
	cmd.MarkFlagsMutuallyExclusive(CommonFlagSecret, CommonFlagSecretPrompt)
	cmd.MarkFlagsMutuallyExclusive(CommonFlagKeyHex, CommonFlagKeyPath)
	cmd.MarkFlagFilename(CommonFlagKeyHex, CommonFlagKeyPath)
}

func CheckCommonFlags(cmd *cobra.Command, args []string) error {
	_ = args
	flags := cmd.Flags()

	if flags.Changed(CommonFlagConnect) == flags.Changed(CommonFlagListen) {
		return fmt.Errorf("expect either --connect/-c or --listen/-l")
	}

	hasCustomSecret := flags.Changed(CommonFlagSecret) || flags.Changed(CommonFlagSecretPrompt)
	hasCustomRemotePublicKeys := flags.Changed(CommonFlagTrustedPublicKeyHex) || flags.Changed(CommonFlagPublicKeyPath)
	hasCustomKeyPair := flags.Changed(CommonFlagKeyHex) || flags.Changed(CommonFlagKeyPath)

	useRaw, err := flags.GetBool(CommonFlagUseRaw)
	if err != nil {
		panic(err)
	}

	if useRaw && (hasCustomSecret || hasCustomRemotePublicKeys || hasCustomKeyPair) {
		return fmt.Errorf("crypto related flags passed but encryption is disabled")
	}

	if hasCustomSecret && (hasCustomRemotePublicKeys || hasCustomKeyPair) {
		return fmt.Errorf("keys are useless when secret is pre-shared")
	}

	if flags.Changed(CommonFlagMaxConn) && !flags.Changed(CommonFlagListen) {
		return fmt.Errorf("--maxConn can only be used with --listen")
	}

	return nil
}

type CommonOpts struct {
	Mode              string
	Endpoint          string
	MaxConnIfListen   int
	PasswordHash      []byte
	SecretHash        []byte
	PrivateKey        ed25519.PrivateKey
	TrustedPublicKeys []ed25519.PublicKey
	UseRaw            bool
}

func GetCommonOpts(cmd *cobra.Command, args []string) (*CommonOpts, error) {
	_ = args

	opts := CommonOpts{}
	flags := cmd.Flags()

	switch {
	case flags.Changed(CommonFlagConnect):
		endpoint, err := flags.GetString(CommonFlagConnect)
		if err != nil {
			panic(err)
		}

		opts.Mode = CommonFlagConnect
		opts.Endpoint = endpoint

	case flags.Changed(CommonFlagListen):
		endpoint, err := flags.GetString(CommonFlagListen)
		if err != nil {
			panic(err)
		}

		opts.Mode = CommonFlagListen
		opts.Endpoint = endpoint

	default:
		panic(internal.Unexpected)
	}

	if maxConn, err := flags.GetInt(CommonFlagMaxConn); err != nil {
		return nil, err
	} else {
		if maxConn < 0 {
			return nil, fmt.Errorf("--maxConn cannot be negative")
		}
		opts.MaxConnIfListen = maxConn
	}

	if flags.Changed(CommonFlagPassword) {
		password, err := flags.GetString(CommonFlagPassword)
		if err != nil {
			panic(err)
		}
		opts.PasswordHash = helper.Hash256([]byte(password))
	} else if flags.Changed(CommonFlagPasswordPrompt) {
		prompt, err := flags.GetBool(CommonFlagPasswordPrompt)
		if err != nil {
			panic(err)
		}
		if prompt {
			password, err := helper.PromptHiddenInput("Password: ")
			if err != nil {
				return nil, err
			}
			opts.PasswordHash = helper.Hash256(password)
		}
	}

	if flags.Changed(CommonFlagSecret) {
		secret, err := flags.GetString(CommonFlagSecret)
		if err != nil {
			panic(err)
		}
		opts.SecretHash = helper.Hash256([]byte(secret))
	} else if flags.Changed(CommonFlagSecretPrompt) {
		prompt, err := flags.GetBool(CommonFlagSecretPrompt)
		if err != nil {
			panic(err)
		}
		if prompt {
			secret, err := helper.PromptHiddenInput("Secret: ")
			if err != nil {
				return nil, err
			}
			opts.SecretHash = helper.Hash256(secret)
		}
	}

	if flags.Changed(CommonFlagKeyHex) {
		hexStr, err := flags.GetString(CommonFlagKeyHex)
		if err != nil {
			panic(err)
		}
		data, err := hex.DecodeString(hexStr)
		if err != nil {
			return nil, err
		}
		opts.PrivateKey = ed25519.PrivateKey(data)
	} else if flags.Changed(CommonFlagKeyPath) {
		path, err := flags.GetString(CommonFlagKeyPath)
		if err != nil {
			panic(err)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("cannot read %v", path)
		}
		data = bytes.Trim(data, "\n")
		items := bytes.Split(data, []byte{0x0a})
		if len(items) != 1 {
			return nil, fmt.Errorf("expect one private key, got %v", len(items))
		}
		data = items[0]
		if len(data) != 128 {
			return nil, fmt.Errorf("invalid private key length %v", len(data))
		}
		data, err = hex.DecodeString(string(data))
		if err != nil {
			return nil, fmt.Errorf("cannot decode private key: %w", err)
		}
		opts.PrivateKey = ed25519.PrivateKey(data)
	} else {
		opts.PrivateKey = initializer.DefaultPrivateKey
	}

	if flags.Changed(CommonFlagTrustedPublicKeyHex) {
		hexStrArr, err := flags.GetStringArray(CommonFlagTrustedPublicKeyHex)
		if err != nil {
			panic(err)
		}
		for _, hexStr := range hexStrArr {
			data, err := hex.DecodeString(hexStr)
			if err != nil {
				return nil, err
			}
			opts.TrustedPublicKeys = append(opts.TrustedPublicKeys, ed25519.PublicKey(data))
		}
	} else if flags.Changed(CommonFlagPublicKeyPath) {
		pathArr, err := flags.GetStringArray(CommonFlagPublicKeyPath)
		if err != nil {
			panic(err)
		}
		keyHexSet := map[string]struct{}{}
		for _, path := range pathArr {
			data, err := os.ReadFile(path)
			if err != nil {
				return nil, err
			}
			data = bytes.Trim(data, "\n")
			items := bytes.Split(data, []byte{0x0a})
			if len(items) == 0 {
				return nil, fmt.Errorf("cannot find any public key")
			}
			for _, item := range items {
				if len(item) == 0 {
					continue
				} else if len(item) != 64 {
					return nil, fmt.Errorf("invalid public key length %v", len(item))
				}

				itemHex := string(item)
				if _, exists := keyHexSet[itemHex]; exists {
					continue
				}
				keyHexSet[itemHex] = struct{}{}

				if item, err = hex.DecodeString(itemHex); err != nil {
					return nil, fmt.Errorf("cannot decode public key: %w", err)
				}
				key := ed25519.PublicKey(item)
				opts.TrustedPublicKeys = append(opts.TrustedPublicKeys, key)
			}
		}
	} else {
		opts.TrustedPublicKeys = []ed25519.PublicKey{
			initializer.DefaultPublicKey,
		}
	}

	if flags.Changed(CommonFlagUseRaw) {
		useRaw, err := flags.GetBool(CommonFlagUseRaw)
		if err != nil {
			return nil, err
		}
		opts.UseRaw = useRaw
	}

	return &opts, nil
}
