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
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("FORGE_ED25519_PUBLIC_KEY=%s\n", base64.StdEncoding.EncodeToString(pub))
	fmt.Printf("FORGE_ED25519_PRIVATE_KEY=%s\n", base64.StdEncoding.EncodeToString(priv))
}
