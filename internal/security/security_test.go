package security

import (
	"crypto/ed25519"
	"encoding/base64"
	"fmt"
	"testing"
)

func TestVerifyChecksum(t *testing.T) {
	data := []byte("package")
	checksums := fmt.Sprintf("%s  forge-connect_darwin_arm64.tar.gz\n", SHA256(data))
	if err := VerifyChecksum(data, checksums, "forge-connect_darwin_arm64.tar.gz"); err != nil {
		t.Fatal(err)
	}
	if err := VerifyChecksum([]byte("bad"), checksums, "forge-connect_darwin_arm64.tar.gz"); err == nil {
		t.Fatal("expected checksum mismatch")
	}
}

func TestVerifyEd25519(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	msg := []byte("checksums")
	sig := ed25519.Sign(priv, msg)
	if err := VerifyEd25519(base64.StdEncoding.EncodeToString(pub), msg, []byte(base64.StdEncoding.EncodeToString(sig))); err != nil {
		t.Fatal(err)
	}
	if err := VerifyEd25519(base64.StdEncoding.EncodeToString(pub), []byte("other"), []byte(base64.StdEncoding.EncodeToString(sig))); err == nil {
		t.Fatal("expected signature failure")
	}
}
