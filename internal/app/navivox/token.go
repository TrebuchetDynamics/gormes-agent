package navivox

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
)

// GenerateSetupToken creates a Navivox bootstrap token.
func GenerateSetupToken() (string, error) {
	var raw [32]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", fmt.Errorf("setup navivox: generate token: %w", err)
	}
	return "nvbx_" + base64.RawURLEncoding.EncodeToString(raw[:]), nil
}

func generateNavivoxSetupToken() (string, error) { return GenerateSetupToken() }
