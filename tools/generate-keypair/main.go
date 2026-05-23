package main

import (
	"crypto/ed25519"
	"encoding/base64"
	"fmt"
	"os"
)

func main() {
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	_, _ = fmt.Printf("FORGE_ED25519_PUBLIC_KEY=%s\n", base64.StdEncoding.EncodeToString(pub))
	_, _ = fmt.Printf("FORGE_ED25519_PRIVATE_KEY=%s\n", base64.StdEncoding.EncodeToString(priv))
}
