package matrix

import (
	"os"
	"strings"
)

type Config struct {
	Homeserver        string
	AccessToken       string
	UserID            string
	Password          string
	DeviceID          string
	Encryption        bool
	AutoThread        bool
	RequireMention    bool
	FreeResponseRooms []string
	AllowedRooms      []string
}

func EnvConfig() Config {
	return Config{
		Homeserver:        trimMatrixHomeserver(os.Getenv("MATRIX_HOMESERVER")),
		AccessToken:       strings.TrimSpace(os.Getenv("MATRIX_ACCESS_TOKEN")),
		UserID:            strings.TrimSpace(os.Getenv("MATRIX_USER_ID")),
		Password:          strings.TrimSpace(os.Getenv("MATRIX_PASSWORD")),
		DeviceID:          strings.TrimSpace(os.Getenv("MATRIX_DEVICE_ID")),
		Encryption:        parseMatrixBool(os.Getenv("MATRIX_ENCRYPTION"), false),
		AutoThread:        parseMatrixBool(os.Getenv("MATRIX_AUTO_THREAD"), true),
		RequireMention:    parseMatrixBool(os.Getenv("MATRIX_REQUIRE_MENTION"), true),
		FreeResponseRooms: splitMatrixList(os.Getenv("MATRIX_FREE_RESPONSE_ROOMS")),
		AllowedRooms:      splitMatrixList(os.Getenv("MATRIX_ALLOWED_ROOMS")),
	}
}

func (c Config) IsAvailable() bool {
	return strings.TrimSpace(c.Homeserver) != "" && (strings.TrimSpace(c.AccessToken) != "" || (strings.TrimSpace(c.UserID) != "" && strings.TrimSpace(c.Password) != ""))
}

type Evidence string

const (
	MatrixEvidenceConfigMissing        Evidence = "matrix_config_missing"
	MatrixEvidenceAuthFailed           Evidence = "matrix_auth_failed"
	MatrixEvidenceSyncUnavailable      Evidence = "matrix_sync_unavailable"
	MatrixEvidenceE2EEUnavailable      Evidence = "matrix_e2ee_unavailable"
	MatrixEvidenceTransportUnavailable Evidence = "matrix_transport_unavailable"

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
		"platform":    "matrix",
		"available":   available,
		"encryption":  ch.cfg.Encryption,
		"auto_thread": ch.cfg.AutoThread,
	}
	if !available {
		status["evidence"] = string(EvidenceNotConfigured)
	}
	return status
}
