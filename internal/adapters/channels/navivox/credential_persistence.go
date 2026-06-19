package navivox

import (
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// credentialPersistenceFile is the on-disk JSON format for device credentials.
// Only the SHA-256 hash is stored — the raw secret is never written to disk.
type credentialPersistenceFile struct {
	Version     int                           `json:"version"`
	Credentials []credentialPersistenceRecord `json:"credentials"`
}

type credentialPersistenceRecord struct {
	CredentialID  string   `json:"credential_id"`
	AppInstallID  string   `json:"app_install_id"`
	GatewayID     string   `json:"gateway_id"`
	Scopes        []string `json:"scopes"`
	CreatedAt     string   `json:"created_at"`
	Revoked       bool     `json:"revoked"`
	SecretHashHex string   `json:"secret_hash"`
}

func loadCredentialsFromDisk(path string) (map[string]*deviceCredentialRecord, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]*deviceCredentialRecord{}, nil
		}
		return nil, err
	}
	var file credentialPersistenceFile
	if err := json.Unmarshal(data, &file); err != nil {
		// A malformed file is not fatal: start with an empty map.
		return map[string]*deviceCredentialRecord{}, nil
	}
	out := make(map[string]*deviceCredentialRecord, len(file.Credentials))
	for _, rec := range file.Credentials {
		hashBytes, err := hex.DecodeString(rec.SecretHashHex)
		if err != nil || len(hashBytes) != 32 {
			continue
		}
		var hash [32]byte
		copy(hash[:], hashBytes)
		createdAt, _ := time.Parse(time.RFC3339, rec.CreatedAt)
		out[rec.CredentialID] = &deviceCredentialRecord{
			CredentialID: rec.CredentialID,
			AppInstallID: rec.AppInstallID,
			GatewayID:    rec.GatewayID,
			Scopes:       rec.Scopes,
			CreatedAt:    createdAt,
			Revoked:      rec.Revoked,
			secretHash:   hash,
		}
	}
	return out, nil
}

func saveCredentialsToDisk(path string, credentials map[string]*deviceCredentialRecord) error {
	records := make([]credentialPersistenceRecord, 0, len(credentials))
	for _, rec := range credentials {
		records = append(records, credentialPersistenceRecord{
			CredentialID:  rec.CredentialID,
			AppInstallID:  rec.AppInstallID,
			GatewayID:     rec.GatewayID,
			Scopes:        rec.Scopes,
			CreatedAt:     rec.CreatedAt.UTC().Format(time.RFC3339),
			Revoked:       rec.Revoked,
			SecretHashHex: hex.EncodeToString(rec.secretHash[:]),
		})
	}
	file := credentialPersistenceFile{Version: 1, Credentials: records}
	data, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	// Atomic write: write to a temp file then rename so a crash can't corrupt the existing file.
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// persistCredentialsToDisk snapshots the current credential map under the lock
// and writes it to disk outside the lock. Non-fatal: logs on failure.
func (c *Channel) persistCredentialsToDisk() {
	if c.credentialsPath == "" {
		return
	}
	c.mu.Lock()
	snapshot := make(map[string]*deviceCredentialRecord, len(c.deviceCredentials))
	for k, v := range c.deviceCredentials {
		rec := *v
		snapshot[k] = &rec
	}
	c.mu.Unlock()
	if err := saveCredentialsToDisk(c.credentialsPath, snapshot); err != nil {
		if c.log != nil {
			c.log.Warn("navivox: failed to persist device credentials", "error", err)
		}
	}
}
