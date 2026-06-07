package navivox

import (
	"context"
	"net/http"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/adapters/channels/navivox/memoryoverview"
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
)

var navivoxMemoryDBPath = config.MemoryDBPath

type memoryOverviewResponse = memoryoverview.Response

func (c *Channel) handleMemoryOverview(w http.ResponseWriter, r *http.Request, _ string) {
	if r.Method != http.MethodGet {
		writeNavivoxError(w, http.StatusMethodNotAllowed, "", "bad_request", "Method not allowed")
		return
	}
	profileID := strings.TrimSpace(r.URL.Query().Get("profile_id"))
	if profileID == "" || profileID == "default" {
		profileID = "main"
	}
	overview := readMemoryOverview(r.Context(), navivoxMemoryDBPath(), profileID)
	writeNavivoxJSON(w, http.StatusOK, overview)
}

func readMemoryOverview(ctx context.Context, dbPath, profileID string) memoryOverviewResponse {
	return memoryoverview.Read(ctx, dbPath, profileID)
}

func safeMemoryDatabaseLabel(path string) string {
	return memoryoverview.SafeDatabaseLabel(path)
}
