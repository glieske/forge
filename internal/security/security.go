package security

import (
	"bufio"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"
)

func SHA256(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func PublicKeyFingerprint(publicKeyBase64 string) (string, error) {
	pub, err := base64.StdEncoding.DecodeString(publicKeyBase64)
	if err != nil {
		return "", err
	}
	if len(pub) != ed25519.PublicKeySize {
		return "", fmt.Errorf("invalid public key length %d", len(pub))
	}
	sum := sha256.Sum256(pub)
	return hex.EncodeToString(sum[:]), nil
}

func ExpectedChecksum(checksums, filename string) (string, error) {
	scanner := bufio.NewScanner(strings.NewReader(checksums))
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 2 {
			continue
		}
		if strings.TrimPrefix(fields[1], "*") == filename {
			return strings.ToLower(fields[0]), nil
		}
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	return "", fmt.Errorf("checksum for %s not found", filename)
}

func VerifyChecksum(data []byte, checksums, filename string) error {
	expected, err := ExpectedChecksum(checksums, filename)
	if err != nil {
		return err
	}
	actual := SHA256(data)
	if actual != expected {
		return fmt.Errorf("checksum mismatch for %s: got %s, want %s", filename, actual, expected)
	}
	return nil
}

func VerifyEd25519(publicKeyBase64 string, message, signature []byte) error {
	if publicKeyBase64 == "" {
		return fmt.Errorf("missing public key")
	}
	pub, err := base64.StdEncoding.DecodeString(publicKeyBase64)
	if err != nil {
		return err
	}
	if len(pub) != ed25519.PublicKeySize {
		return fmt.Errorf("invalid public key length %d", len(pub))
	}
	sig, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(signature)))
	if err != nil {
		return err
	}
	if !ed25519.Verify(ed25519.PublicKey(pub), message, sig) {
		return fmt.Errorf("signature verification failed")
	}
	return nil
}
