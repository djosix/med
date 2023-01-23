package initializer

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	_ "embed"
	"fmt"
	"io"

	"github.com/aead/ecdh"
	"github.com/djosix/med/internal"
	"github.com/djosix/med/internal/helper"
	pb "github.com/djosix/med/internal/protobuf"
	"github.com/djosix/med/internal/readwriter"
	"google.golang.org/protobuf/proto"
)

func InitHandshake(privateKey ed25519.PrivateKey, trustedPublicKeys []ed25519.PublicKey) Initializer {
	return func(ctx context.Context, rw io.ReadWriter) (ctxOut context.Context, rwOut io.ReadWriter, err error) {
		initLogger.Debug("Handshake")

		sharedSecret, err := Handshake(rw, privateKey, trustedPublicKeys)
		if err != nil {
			return
		}

		rwOut = rw
		ctxOut = context.WithValue(ctx, "secret", sharedSecret)

		return
	}
}

var (
	DefaultPublicKey = ed25519.PublicKey{
		0x75, 0x5e, 0x48, 0x52, 0xac, 0x37, 0xc6, 0x23, 0xff, 0xab, 0x6c, 0x75, 0x37, 0x2e, 0x7, 0x25,
		0xbe, 0xd5, 0x1a, 0xdb, 0x79, 0x32, 0x92, 0xf7, 0x37, 0xf4, 0xc3, 0xa, 0xf3, 0x73, 0xa4, 0x60,
	}
	DefaultPrivateKey = ed25519.PrivateKey{
		0xd6, 0x18, 0x85, 0x58, 0x94, 0x12, 0xdf, 0xcb, 0xf5, 0x62, 0x96, 0xc7, 0xe8, 0xb8, 0xf1, 0x6c,
		0xb0, 0xd0, 0x83, 0x3f, 0xc1, 0x3a, 0xe8, 0xed, 0x4d, 0xb9, 0xe6, 0xe3, 0x38, 0xb8, 0xd4, 0x59,
		0x75, 0x5e, 0x48, 0x52, 0xac, 0x37, 0xc6, 0x23, 0xff, 0xab, 0x6c, 0x75, 0x37, 0x2e, 0x7, 0x25,
		0xbe, 0xd5, 0x1a, 0xdb, 0x79, 0x32, 0x92, 0xf7, 0x37, 0xf4, 0xc3, 0xa, 0xf3, 0x73, 0xa4, 0x60,
	}
)

func Handshake(rw io.ReadWriter, privateKey ed25519.PrivateKey, trustedPublicKeys []ed25519.PublicKey) ([]byte, error) {
	f := readwriter.NewPlainFrameReadWriter(rw)

	var localReq pb.KexReq
	{
		localReq.Nonce = make([]byte, 32)
		if _, err := rand.Read(localReq.Nonce); err != nil {
			return nil, err
		}

		localReq.PublicKeyHash = helper.HashSalt256(
			[]byte(privateKey.Public().(ed25519.PublicKey)),
			localReq.Nonce,
		)

		body := []byte{}
		body = append(body, localReq.Nonce...)
		body = append(body, localReq.PublicKeyHash...)
		localReq.Signature = ed25519.Sign(privateKey, body)

		data, err := proto.Marshal(&localReq)
		if err != nil {
			return nil, fmt.Errorf("cannot marshal localReq")
		}

		if err := f.WriteFrame(data); err != nil {
			return nil, fmt.Errorf("cannot write localReq frame")
		}
	}

	var remotePublicKey ed25519.PublicKey = nil
	var remoteReq pb.KexReq
	{
		data, err := f.ReadFrame()
		if err != nil {
			return nil, fmt.Errorf("cannot read remoteReq frame: %w", err)
		}

		if err := proto.Unmarshal(data, &remoteReq); err != nil {
			return nil, fmt.Errorf("cannot unmarshal remoteReq: %w", err)
		}

		var invalid = false
		invalid = invalid || len(remoteReq.Nonce) != len(localReq.Nonce)
		invalid = invalid || len(remoteReq.Signature) != len(localReq.Signature)
		invalid = invalid || len(remoteReq.PublicKeyHash) != len(localReq.PublicKeyHash)
		if invalid {
			return nil, fmt.Errorf("invalid KexReq")
		}

		for _, publicKey := range trustedPublicKeys {
			hash := helper.HashSalt256(publicKey, remoteReq.Nonce)
			if bytes.Equal(hash, remoteReq.PublicKeyHash) {
				remotePublicKey = publicKey
				break
			}
		}
		if remotePublicKey == nil {
			return nil, fmt.Errorf("untrusted remote")
		}

		body := []byte{}
		body = append(body, remoteReq.Nonce...)
		body = append(body, remoteReq.PublicKeyHash...)
		if !ed25519.Verify(remotePublicKey, body, remoteReq.Signature) {
			return nil, fmt.Errorf("invalid KexReq.signature")
		}
	}

	kex := ecdh.X25519()
	kexPrivateKey, kexPublicKey, err := kex.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}

	var localResp pb.KexResp
	{
		kexPublicKey, ok := kexPublicKey.([32]byte)
		if !ok {
			panic(internal.Unexpected)
		}
		localResp.Nonce = remoteReq.Nonce
		localResp.KexPublicKey = kexPublicKey[:]

		var body []byte
		body = append(body, localResp.Nonce...)
		body = append(body, localResp.KexPublicKey...)
		localResp.Signature = ed25519.Sign(privateKey, body)

		data, err := proto.Marshal(&localResp)
		if err != nil {
			return nil, fmt.Errorf("cannot marshal localResp: %w", err)
		}
		if err := f.WriteFrame(data); err != nil {
			return nil, fmt.Errorf("cannot write localResp frame: %w", err)
		}
	}

	var remoteResp pb.KexResp
	{
		data, err := f.ReadFrame()
		if err != nil {
			return nil, fmt.Errorf("cannot read remoteResp frame")
		}
		if err := proto.Unmarshal(data, &remoteResp); err != nil {
			return nil, fmt.Errorf("cannot unmarshal remoteResp: %w", err)
		}

		var invalid = false
		invalid = invalid || len(remoteResp.Nonce) != len(localResp.Nonce)
		invalid = invalid || len(remoteResp.KexPublicKey) != len(localResp.KexPublicKey)
		invalid = invalid || len(remoteResp.Signature) != len(localResp.Signature)
		invalid = invalid || !bytes.Equal(remoteResp.Nonce, localReq.Nonce)
		if invalid {
			return nil, fmt.Errorf("invalid kexResp")
		}

		var body []byte
		body = append(body, remoteResp.Nonce...)
		body = append(body, remoteResp.KexPublicKey...)
		if !ed25519.Verify(remotePublicKey, body, remoteResp.Signature) {
			return nil, fmt.Errorf("invalid kexResp.signature")
		}

		if err := kex.Check(remoteResp.KexPublicKey); err != nil {
			return nil, fmt.Errorf("invalid kexResp.kexPublicKey: %w", err)
		}
	}

	sharedSecret := kex.ComputeSecret(kexPrivateKey, remoteResp.KexPublicKey)

	return sharedSecret, nil
}
