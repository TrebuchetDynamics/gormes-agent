package matrix

import (
	"os"
	"strings"
)

type Config struct {
	Homeserver string
	UserID     string
	Password   string
	DeviceID   string
	Encryption bool
	AutoThread bool
}

func EnvConfig() Config {
	return Config{
		Homeserver: strings.TrimSpace(os.Getenv("MATRIX_HOMESERVER")),
		UserID:     strings.TrimSpace(os.Getenv("MATRIX_USER_ID")),
		Password:   strings.TrimSpace(os.Getenv("MATRIX_PASSWORD")),
		DeviceID:   strings.TrimSpace(os.Getenv("MATRIX_DEVICE_ID")),
		Encryption: strings.ToLower(os.Getenv("MATRIX_ENCRYPTION")) == "true",
		AutoThread: strings.ToLower(os.Getenv("MATRIX_AUTO_THREAD")) != "false",
	}
}

func (c Config) IsAvailable() bool {
	return c.Homeserver != "" && c.UserID != "" && c.Password != ""
}

type Evidence string

const (
	EvidenceMissingCredentials Evidence = "matrix_missing_credentials"
	EvidenceDisabled           Evidence = "matrix_disabled"
	EvidenceNotConfigured      Evidence = "matrix_not_configured"
)

type Channel struct {
	cfg Config
}

func New(cfg Config) *Channel {
	return &Channel{cfg: cfg}
}

func (ch *Channel) Available() bool {
	if !ch.cfg.IsAvailable() {
		return false
	}
	return true
}

func (ch *Channel) Status() map[string]any {
	available := ch.cfg.IsAvailable()
	status := map[string]any{
		"platform":   "matrix",
		"available":  available,
		"encryption": ch.cfg.Encryption,
		"auto_thread": ch.cfg.AutoThread,
	}
	if !available {
		status["evidence"] = string(EvidenceNotConfigured)
	}
	return status
}
