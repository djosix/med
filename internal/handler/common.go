package handler

import (
	"bytes"
	"crypto/ed25519"
	"encoding/hex"
	"fmt"
	"os"
	"time"

	"github.com/djosix/med/internal"
	"github.com/djosix/med/internal/helper"
	"github.com/djosix/med/internal/initializer"
	"github.com/djosix/med/internal/logger"
	"github.com/spf13/cobra"
)

const (
	CommonFlagConnect         = "connect"
	CommonFlagConnectP        = "c"
	CommonFlagListen          = "listen"
	CommonFlagListenP         = "l"
	CommonFlagMaxConn         = "max-conn"
	CommonFlagMaxConnP        = "C"
	CommonFlagConnDelay       = "conn-delay"
	CommonFlagConnDelayP      = "D"
	CommonFlagPassword        = "passwd"
	CommonFlagPasswordP       = "w"
	CommonFlagPasswordPrompt  = "prompt-passwd"
	CommonFlagPasswordPromptP = "W"
	CommonFlagSecret          = "secret"
	CommonFlagSecretP         = "s"
	CommonFlagSecretPrompt    = "prompt-secret"
	CommonFlagSecretPromptP   = "S"
	CommonFlagTrustPubHex     = "pub"
	CommonFlagTrustPubHexP    = "p"
	CommonFlagTrustPubPath    = "pub-path"
	CommonFlagTrustPubPathP   = "P"
	CommonFlagKeyHex          = "key"
	CommonFlagKeyHexP         = "k"
	CommonFlagKeyPath         = "key-path"
	CommonFlagKeyPathP        = "K"
	CommonFlagUseRaw          = "use-raw"
	CommonFlagUseRawP         = "r"
	CommonFlagVerbose         = "verbose"
	CommonFlagVerboseP        = "v"
)

func InitCommonFlags(cmd *cobra.Command) {
	flags := cmd.Flags()
	flags.SortFlags = false

	flags.StringP(CommonFlagConnect, CommonFlagConnectP, "", "endpoint to connect to, like \"[HOST]:PORT\" or \"unix:PATH\"")
	flags.StringP(CommonFlagListen, CommonFlagListenP, "", "endpoint to listen on, like \"[HOST]:PORT\" or \"unix:PATH\"")
	flags.DurationP(CommonFlagConnDelay, CommonFlagConnDelayP, 0, fmt.Sprintf("reconnection delay if using --%s\nany non-positive duration means connecting once", CommonFlagConnect))
	flags.IntP(CommonFlagMaxConn, CommonFlagMaxConnP, 64, fmt.Sprintf("max number of connections at a time if using --%s", CommonFlagListen))
	flags.StringP(CommonFlagPassword, CommonFlagPasswordP, "", "the password used by the server to authenticate clients")
	flags.BoolP(CommonFlagPasswordPrompt, CommonFlagPasswordPromptP, false, "prompt user for the password")
	flags.StringP(CommonFlagSecret, CommonFlagSecretP, "", "shared secret for symmetric encryption. If specified\nkey exchange handshake will be disabled")
	flags.BoolP(CommonFlagSecretPrompt, CommonFlagSecretPromptP, false, "prompt user for the shared secret")
	flags.StringArrayP(CommonFlagTrustPubHex, CommonFlagTrustPubHexP, []string{}, "hex-encoded trusted public key(s)")
	flags.StringArrayP(CommonFlagTrustPubPath, CommonFlagTrustPubPathP, []string{}, "trusted public key file path(s)")
	flags.StringP(CommonFlagKeyHex, CommonFlagKeyHexP, "", "hex-encoded private key")
	flags.StringP(CommonFlagKeyPath, CommonFlagKeyPathP, "", "private key file path")
	flags.BoolP(CommonFlagUseRaw, CommonFlagUseRawP, false, "disable any encryption or compression")
	flags.IntP(CommonFlagVerbose, CommonFlagVerboseP, int(logger.LevelInfo), "verbosity, 0-"+fmt.Sprint(logger.MaxLogLevel))

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
		return fmt.Errorf("expect either --%s/-%s or --%s/-%s",
			CommonFlagConnect, CommonFlagConnectP,
			CommonFlagListen, CommonFlagListenP,
		)
	}

	// Check crypto-related flags
	{
		hasCustomSecret := flags.Changed(CommonFlagSecret) || flags.Changed(CommonFlagSecretPrompt)
		hasCustomRemotePublicKeys := flags.Changed(CommonFlagTrustPubHex) || flags.Changed(CommonFlagTrustPubPath)
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
	}

	if flags.Changed(CommonFlagMaxConn) && !flags.Changed(CommonFlagListen) {
		return fmt.Errorf("--%s can only be used with --%s", CommonFlagMaxConn, CommonFlagListen)
	}

	if flags.Changed(CommonFlagConnDelay) && !flags.Changed(CommonFlagConnect) {
		return fmt.Errorf("--%s can only be used with --%s", CommonFlagConnDelay, CommonFlagConnect)
	}

	return nil
}

type CommonOpts struct {
	Mode               string
	Endpoint           string
	MaxConnIfListen    int
	ConnDelayIfConnect time.Duration
	PasswordHash       []byte
	SecretHash         []byte
	PrivateKey         ed25519.PrivateKey
	TrustedPublicKeys  []ed25519.PublicKey
	UseRaw             bool
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
		opts.MaxConnIfListen = maxConn
	}

	if connDelay, err := flags.GetDuration(CommonFlagConnDelay); err != nil {
		return nil, err
	} else {
		opts.ConnDelayIfConnect = time.Duration(connDelay)
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

	if flags.Changed(CommonFlagTrustPubHex) {
		hexStrArr, err := flags.GetStringArray(CommonFlagTrustPubHex)
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
	} else if flags.Changed(CommonFlagTrustPubPath) {
		pathArr, err := flags.GetStringArray(CommonFlagTrustPubPath)
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
			initializer.DefaultTrustedPublicKey,
		}
	}

	if flags.Changed(CommonFlagUseRaw) {
		useRaw, err := flags.GetBool(CommonFlagUseRaw)
		if err != nil {
			return nil, err
		}
		opts.UseRaw = useRaw
	}

	if verbosity, err := flags.GetInt(CommonFlagVerbose); err != nil {
		panic(CommonFlagVerbose)
	} else {
		logger.SetLevel(logger.LogLevel(verbosity))
	}

	return &opts, nil
}
