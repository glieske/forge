package main

import (
	"crypto/ed25519"
	"encoding/base64"
	"fmt"
	"os"
)

func main() {
	if len(os.Args) != 3 {
		stderr("usage: sign-file <input> <output.sig>")
		os.Exit(2)
	}
	raw := os.Getenv("FORGE_ED25519_PRIVATE_KEY")
	if raw == "" {
		stderr("FORGE_ED25519_PRIVATE_KEY is required")
		os.Exit(2)
	}
	key, err := base64.StdEncoding.DecodeString(raw)
	if err != nil {
		stderr(err)
		os.Exit(2)
	}
	if len(key) == ed25519.SeedSize {
		key = ed25519.NewKeyFromSeed(key)
	}
	if len(key) != ed25519.PrivateKeySize {
		stderrf("invalid private key length %d; expected %d-byte seed or %d-byte private key\n", len(key), ed25519.SeedSize, ed25519.PrivateKeySize)
		os.Exit(2)
	}
	msg, err := os.ReadFile(os.Args[1])
	if err != nil {
		stderr(err)
		os.Exit(1)
	}
	sig := ed25519.Sign(ed25519.PrivateKey(key), msg)
	if err := os.WriteFile(os.Args[2], []byte(base64.StdEncoding.EncodeToString(sig)+"\n"), 0o644); err != nil {
		stderr(err)
		os.Exit(1)
	}
}

func stderr(args ...interface{}) {
	_, _ = fmt.Fprintln(os.Stderr, args...)
}

func stderrf(format string, args ...interface{}) {
	_, _ = fmt.Fprintf(os.Stderr, format, args...)
}
