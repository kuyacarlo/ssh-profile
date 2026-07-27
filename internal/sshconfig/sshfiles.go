package sshconfig

import (
	"encoding/pem"
	"os"
	"strings"
)

// IdentityType classifies an SSH key file.
type IdentityType string

const (
	PrivateKey IdentityType = "privkey"
	PublicKey  IdentityType = "pubkey"
	Unknown    IdentityType = "unknown"
)

// ClassifyKey inspects a file and returns whether it looks like a private or public key.
func ClassifyKey(fp string) (IdentityType, error) {
	data, err := os.ReadFile(fp) //nolint:gosec
	if err != nil {
		return Unknown, err
	}

	if block, _ := pem.Decode(data); block != nil {
		return PrivateKey, nil
	}

	fields := strings.Fields(strings.TrimSpace(string(data)))
	if len(fields) >= 2 && isKnownKeyType(fields[0]) {
		return PublicKey, nil
	}

	return Unknown, nil
}

func isKnownKeyType(t string) bool {
	switch {
	case t == "ssh-rsa",
		t == "ssh-dss",
		t == "ssh-ed25519",
		strings.HasPrefix(t, "ecdsa-sha2-"),
		strings.HasPrefix(t, "sk-ssh-"),
		strings.HasPrefix(t, "sk-ecdsa-"):
		return true
	default:
		return false
	}
}
