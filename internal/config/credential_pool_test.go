package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestCredentialPoolLoadRoundTrip(t *testing.T) {
	hermesHome := t.TempDir()
	now := time.Unix(1_700_000_000, 0).UTC()
	entries := []PooledCredential{
		{ID: "alpha", Label: "Alpha", AuthType: CredentialAuthAPIKey, Priority: 1, Source: "fixture", AccessToken: "plain-existing-token", RefreshToken: "plain-refresh-token", RequestCount: 2},
		{ID: "beta", Label: "Beta", AuthType: CredentialAuthOAuth, Priority: 0, Source: "fixture", AccessToken: "second-existing-token", LastStatus: CredentialStatusOK},
	}
	if err := SaveCredentialPoolEntries(CredentialPoolOptions{HermesHome: hermesHome, Provider: "fixture-provider"}, entries); err != nil {
		t.Fatal(err)
	}

	pool, evidence, err := LoadCredentialPool(CredentialPoolOptions{HermesHome: hermesHome, Provider: "fixture-provider", Now: func() time.Time { return now }})
	if err != nil {
		t.Fatal(err)
	}
	if evidence.Code != CredentialPoolEvidenceLoaded || evidence.Count != 2 || !evidence.Redacted {
		t.Fatalf("evidence = %#v", evidence)
	}
	loaded := pool.Entries()
	if len(loaded) != 2 || loaded[0].ID != "beta" || loaded[1].ID != "alpha" {
		t.Fatalf("loaded priority order = %#v", loaded)
	}
	status := pool.RedactedStatus()
	blob, err := json.Marshal(status)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(blob), "plain-existing-token") || strings.Contains(string(blob), "plain-refresh-token") || strings.Contains(string(blob), "second-existing-token") {
		t.Fatalf("redacted status leaked token fields: %s", blob)
	}
	if !strings.Contains(string(blob), "fixture-provider") || !strings.Contains(string(blob), "Alpha") {
		t.Fatalf("redacted status omitted safe metadata: %s", blob)
	}
}

func TestCredentialPoolLoadFiltersByOwnerProfile(t *testing.T) {
	hermesHome := t.TempDir()
	entries := []PooledCredential{
		{ID: "main-provider", Label: "Main provider", AuthType: CredentialAuthAPIKey, Priority: 0, Source: "fixture", OwnerProfile: DefaultProfileID, AccessToken: "main-token"},
		{ID: "alpha-provider", Label: "Alpha provider", AuthType: CredentialAuthAPIKey, Priority: 1, Source: "fixture", OwnerProfile: "alpha", AccessToken: "alpha-token"},
		{ID: "beta-provider", Label: "Beta provider", AuthType: CredentialAuthAPIKey, Priority: 2, Source: "fixture", OwnerProfile: "beta", AccessToken: "beta-token"},
		{ID: "legacy-provider", Label: "Legacy provider", AuthType: CredentialAuthAPIKey, Priority: 3, Source: "fixture", AccessToken: "legacy-token"},
	}
	if err := SaveCredentialPoolEntries(CredentialPoolOptions{HermesHome: hermesHome, Provider: "fixture-provider"}, entries); err != nil {
		t.Fatal(err)
	}

	pool, evidence, err := LoadCredentialPool(CredentialPoolOptions{HermesHome: hermesHome, Provider: "fixture-provider", ProfileID: "alpha"})
	if err != nil {
		t.Fatal(err)
	}
	loaded := pool.Entries()
	if evidence.Count != 4 || evidence.FilteredCount != 3 {
		t.Fatalf("evidence = %#v, want count=4 filtered=3", evidence)
	}
	if ids := credentialEntryIDs(loaded); strings.Join(ids, ",") != "main-provider,alpha-provider,legacy-provider" {
		t.Fatalf("alpha profile entries = %v, want shared main, alpha-owned, and legacy entries only", ids)
	}
	for _, entry := range loaded {
		if entry.ID == "beta-provider" || strings.Contains(entry.AccessToken, "beta-token") {
			t.Fatalf("profile-filtered pool leaked beta-owned credential: %#v", loaded)
		}
	}

	unfiltered, evidence, err := LoadCredentialPool(CredentialPoolOptions{HermesHome: hermesHome, Provider: "fixture-provider"})
	if err != nil {
		t.Fatal(err)
	}
	if len(unfiltered.Entries()) != 4 || evidence.FilteredCount != 0 {
		t.Fatalf("unfiltered entries=%v evidence=%#v, want all entries and no filtered count", credentialEntryIDs(unfiltered.Entries()), evidence)
	}
}

func TestCredentialPoolProfileFilteredPersistPreservesOtherOwners(t *testing.T) {
	hermesHome := t.TempDir()
	entries := []PooledCredential{
		{ID: "alpha", Label: "Alpha", AuthType: CredentialAuthAPIKey, Source: "fixture", OwnerProfile: "alpha", AccessToken: "alpha-token"},
		{ID: "beta", Label: "Beta", AuthType: CredentialAuthAPIKey, Source: "fixture", OwnerProfile: "beta", AccessToken: "beta-token"},
	}
	if err := SaveCredentialPoolEntries(CredentialPoolOptions{HermesHome: hermesHome, Provider: "fixture-provider"}, entries); err != nil {
		t.Fatal(err)
	}
	pool, _, err := LoadCredentialPool(CredentialPoolOptions{HermesHome: hermesHome, Provider: "fixture-provider", ProfileID: "alpha", Strategy: CredentialPoolStrategyLeastUsed})
	if err != nil {
		t.Fatal(err)
	}
	selected, _ := pool.Select()
	if selected == nil || selected.ID != "alpha" {
		t.Fatalf("selected = %#v, want alpha", selected)
	}

	unfiltered, _, err := LoadCredentialPool(CredentialPoolOptions{HermesHome: hermesHome, Provider: "fixture-provider"})
	if err != nil {
		t.Fatal(err)
	}
	loaded := unfiltered.Entries()
	if ids := credentialEntryIDs(loaded); strings.Join(ids, ",") != "alpha,beta" {
		t.Fatalf("entries after profile-filtered persist = %v, want alpha,beta", ids)
	}
	if beta := entryByID(loaded, "beta"); beta == nil || beta.AccessToken != "beta-token" || beta.OwnerProfile != "beta" {
		t.Fatalf("beta entry was not preserved: %#v", loaded)
	}
	if alpha := entryByID(loaded, "alpha"); alpha == nil || alpha.RequestCount != 1 {
		t.Fatalf("alpha request count not persisted: %#v", loaded)
	}
}

func TestCredentialPoolStatusIncludesRedactedOwnerProfile(t *testing.T) {
	hermesHome := t.TempDir()
	entries := []PooledCredential{
		{ID: "alpha", Label: "Alpha", AuthType: CredentialAuthAPIKey, Source: "fixture", OwnerProfile: "alpha", AccessToken: "alpha-token"},
	}
	if err := SaveCredentialPoolEntries(CredentialPoolOptions{HermesHome: hermesHome, Provider: "fixture-provider"}, entries); err != nil {
		t.Fatal(err)
	}

	pool, _, err := LoadCredentialPool(CredentialPoolOptions{HermesHome: hermesHome, Provider: "fixture-provider", ProfileID: "alpha"})
	if err != nil {
		t.Fatal(err)
	}
	status := pool.RedactedStatus()
	if len(status.Entries) != 1 || status.Entries[0].OwnerProfile != "alpha" || !status.Entries[0].SecretsRedacted {
		t.Fatalf("status = %#v, want redacted alpha owner evidence", status)
	}
	blob, err := json.Marshal(status)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(blob), "alpha-token") {
		t.Fatalf("redacted status leaked token: %s", blob)
	}
}

func TestCredentialPoolStatusSanitizesTokenFields(t *testing.T) {
	resetCredentialSanitizerWarningsForTest()
	hermesHome := t.TempDir()
	entries := []PooledCredential{
		{ID: "alpha", Label: "Alpha", AuthType: CredentialAuthOAuth, Source: "fixture", AccessToken: "tok\u028ben", RefreshToken: "ref\u00e9resh"},
	}
	if err := SaveCredentialPoolEntries(CredentialPoolOptions{HermesHome: hermesHome, Provider: "fixture-provider"}, entries); err != nil {
		t.Fatal(err)
	}

	pool, _, err := LoadCredentialPool(CredentialPoolOptions{HermesHome: hermesHome, Provider: "fixture-provider"})
	if err != nil {
		t.Fatal(err)
	}
	loaded := pool.Entries()
	if len(loaded) != 1 {
		t.Fatalf("entries len = %d, want 1", len(loaded))
	}
	if loaded[0].AccessToken != "token" || loaded[0].RefreshToken != "refresh" {
		t.Fatalf("loaded tokens = access %q refresh %q, want sanitized", loaded[0].AccessToken, loaded[0].RefreshToken)
	}

	status := pool.RedactedStatus()
	blob, err := json.Marshal(status)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(blob), "tok\u028ben") || strings.Contains(string(blob), "ref\u00e9resh") {
		t.Fatalf("redacted status leaked raw non-ASCII token bytes: %s", blob)
	}
}

func TestCredentialPoolSelectStrategies(t *testing.T) {
	now := time.Unix(1_700_000_000, 0).UTC()
	base := []PooledCredential{
		{ID: "a", Label: "A", AuthType: CredentialAuthAPIKey, Priority: 0, Source: "fixture", AccessToken: "token-a", RequestCount: 8},
		{ID: "b", Label: "B", AuthType: CredentialAuthAPIKey, Priority: 1, Source: "fixture", AccessToken: "token-b", RequestCount: 1},
		{ID: "c", Label: "C", AuthType: CredentialAuthAPIKey, Priority: 2, Source: "fixture", AccessToken: "token-c", LastStatus: CredentialStatusExhausted, LastStatusAt: now.Unix(), LastErrorResetAt: now.Add(time.Hour).Unix()},
	}

	fill := newFixtureCredentialPool(t, base, CredentialPoolStrategyFillFirst, now)
	selected, evidence := fill.Select()
	if selected == nil || selected.ID != "a" || evidence.Code != CredentialPoolEvidenceSelected {
		t.Fatalf("fill_first selected=%#v evidence=%#v", selected, evidence)
	}

	roundRobin := newFixtureCredentialPool(t, base, CredentialPoolStrategyRoundRobin, now)
	first, _ := roundRobin.Select()
	second, _ := roundRobin.Select()
	if first == nil || second == nil || first.ID != "a" || second.ID != "b" {
		t.Fatalf("round_robin selected first=%#v second=%#v", first, second)
	}

	leastUsed := newFixtureCredentialPool(t, base, CredentialPoolStrategyLeastUsed, now)
	selected, _ = leastUsed.Select()
	if selected == nil || selected.ID != "b" || selected.RequestCount != 2 {
		t.Fatalf("least_used selected=%#v, want b with incremented count", selected)
	}

	randomPool := newFixtureCredentialPool(t, base, CredentialPoolStrategyRandom, now)
	randomPool.SetDeterministicRandom(1)
	selected, _ = randomPool.Select()
	if selected == nil || selected.ID == "c" {
		t.Fatalf("random selected exhausted credential or nil: %#v", selected)
	}

	fallback := newFixtureCredentialPool(t, base, CredentialPoolStrategy("unknown"), now)
	selected, evidence = fallback.Select()
	if selected == nil || selected.ID != "a" || evidence.Strategy != CredentialPoolStrategyFillFirst || evidence.Code != CredentialPoolEvidenceSelected {
		t.Fatalf("invalid strategy fallback selected=%#v evidence=%#v", selected, evidence)
	}
}

func TestCredentialPoolExhaustedCooldownAndRotate(t *testing.T) {
	now := time.Unix(1_700_000_000, 0).UTC()
	pool := newFixtureCredentialPool(t, []PooledCredential{
		{ID: "a", Label: "A", AuthType: CredentialAuthAPIKey, Priority: 0, Source: "fixture", AccessToken: "token-a"},
		{ID: "b", Label: "B", AuthType: CredentialAuthAPIKey, Priority: 1, Source: "fixture", AccessToken: "token-b"},
	}, CredentialPoolStrategyFillFirst, now)

	selected, _ := pool.Select()
	if selected == nil || selected.ID != "a" {
		t.Fatalf("initial selected = %#v", selected)
	}
	resetAt := now.Add(30 * time.Minute)
	next, evidence := pool.MarkExhaustedAndRotate(CredentialExhaustion{StatusCode: 429, Reason: "rate_limited", Message: "provider said retry", ResetAt: resetAt})
	if next == nil || next.ID != "b" || evidence.Code != CredentialPoolEvidenceExhausted {
		t.Fatalf("rotate next=%#v evidence=%#v", next, evidence)
	}
	entryA := entryByID(pool.Entries(), "a")
	if entryA == nil || entryA.LastStatus != CredentialStatusExhausted || entryA.LastErrorCode != 429 || entryA.LastErrorReason != "rate_limited" || entryA.LastErrorResetAt != resetAt.Unix() {
		t.Fatalf("exhausted metadata = %#v", entryA)
	}

	pool.now = func() time.Time { return resetAt.Add(time.Second) }
	selected, _ = pool.Select()
	if selected == nil || selected.ID != "a" {
		t.Fatalf("after cooldown selected = %#v", selected)
	}
	entryA = entryByID(pool.Entries(), "a")
	if entryA == nil || entryA.LastStatus != CredentialStatusOK || entryA.LastErrorCode != 0 || entryA.LastErrorReason != "" {
		t.Fatalf("expired cooldown did not clear status: %#v", entryA)
	}
}

func TestCredentialPoolLeaseAccounting(t *testing.T) {
	now := time.Unix(1_700_000_000, 0).UTC()
	pool := newFixtureCredentialPool(t, []PooledCredential{
		{ID: "a", Label: "A", AuthType: CredentialAuthAPIKey, Priority: 0, Source: "fixture", AccessToken: "token-a"},
		{ID: "b", Label: "B", AuthType: CredentialAuthAPIKey, Priority: 1, Source: "fixture", AccessToken: "token-b"},
	}, CredentialPoolStrategyLeastUsed, now)
	pool.SetMaxConcurrentPerCredential(1)

	leaseA, evidence := pool.AcquireLease("")
	if leaseA != "a" || evidence.Code != CredentialPoolEvidenceLeaseAcquired {
		t.Fatalf("leaseA=%q evidence=%#v", leaseA, evidence)
	}
	leaseB, _ := pool.AcquireLease("")
	if leaseB != "b" {
		t.Fatalf("leaseB=%q, want b below soft cap", leaseB)
	}
	leaseAgain, _ := pool.AcquireLease("")
	if leaseAgain == "" {
		t.Fatal("leaseAgain empty, want least leased credential even when all at soft cap")
	}
	pool.ReleaseLease(leaseAgain)
	leases := pool.ActiveLeases()
	if got := leases[leaseAgain]; got != 1 {
		t.Fatalf("lease %s after one release = %d, leases=%#v", leaseAgain, got, leases)
	}
	pool.ReleaseLease(leaseAgain)
	leases = pool.ActiveLeases()
	if got := leases[leaseAgain]; got != 0 {
		t.Fatalf("lease %s after second release = %d, leases=%#v", leaseAgain, got, leases)
	}
}

func TestCredentialPoolCorruptStoreEvidence(t *testing.T) {
	hermesHome := t.TempDir()
	authPath := filepath.Join(hermesHome, "auth.json")
	if err := os.WriteFile(authPath, []byte(`{"credential_pool":`), 0o600); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(authPath)
	if err != nil {
		t.Fatal(err)
	}

	pool, evidence, err := LoadCredentialPool(CredentialPoolOptions{HermesHome: hermesHome, Provider: "fixture-provider"})
	if err == nil {
		t.Fatal("LoadCredentialPool err = nil, want corrupt-store error")
	}
	if pool == nil || len(pool.Entries()) != 0 || evidence.Code != CredentialPoolEvidenceCorrupt || !evidence.Redacted {
		t.Fatalf("pool=%#v evidence=%#v err=%v", pool, evidence, err)
	}
	after, err := os.ReadFile(authPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Fatalf("corrupt auth store changed: before=%q after=%q", before, after)
	}
	if strings.Contains(evidence.Message, string(before)) || strings.Contains(evidence.Message, hermesHome) {
		t.Fatalf("corrupt evidence leaked file contents or host path: %#v", evidence)
	}
}

func newFixtureCredentialPool(t *testing.T, entries []PooledCredential, strategy CredentialPoolStrategy, now time.Time) *CredentialPool {
	t.Helper()
	hermesHome := t.TempDir()
	if err := SaveCredentialPoolEntries(CredentialPoolOptions{HermesHome: hermesHome, Provider: "fixture-provider"}, entries); err != nil {
		t.Fatal(err)
	}
	pool, _, err := LoadCredentialPool(CredentialPoolOptions{HermesHome: hermesHome, Provider: "fixture-provider", Strategy: strategy, Now: func() time.Time { return now }})
	if err != nil {
		t.Fatal(err)
	}
	return pool
}

func credentialEntryIDs(entries []PooledCredential) []string {
	ids := make([]string, 0, len(entries))
	for _, entry := range entries {
		ids = append(ids, entry.ID)
	}
	return ids
}

func entryByID(entries []PooledCredential, id string) *PooledCredential {
	for i := range entries {
		if entries[i].ID == id {
			return &entries[i]
		}
	}
	return nil
}

func TestNousOAuthCredentialPoolRoundTrip(t *testing.T) {
	hermesHome := t.TempDir()
	creds := NousOAuthCredentials{
		Label:              "default",
		PortalBaseURL:      "https://portal.nousresearch.com",
		InferenceBaseURL:   "https://inference.nousresearch.com",
		ClientID:           "test-client-id",
		Scope:              "openid profile",
		TokenType:          "Bearer",
		AccessToken:        "test-access-token",
		RefreshToken:       "test-refresh-token",
		ObtainedAt:         "2026-05-08T00:00:00Z",
		ExpiresAt:          "2026-05-09T00:00:00Z",
		ExpiresIn:          86400,
		AgentKey:           "test-agent-key",
		AgentKeyID:         "key-001",
		AgentKeyExpiresAt:  "2026-05-08T06:00:00Z",
		AgentKeyExpiresIn:  21600,
		AgentKeyObtainedAt: "2026-05-08T00:00:00Z",
	}
	saved, err := SaveNousOAuthCredentials(CredentialPoolOptions{HermesHome: hermesHome}, creds)
	if err != nil {
		t.Fatalf("SaveNousOAuthCredentials: %v", err)
	}
	if saved.AccessToken != creds.AccessToken {
		t.Errorf("saved.AccessToken = %q, want %q", saved.AccessToken, creds.AccessToken)
	}

	loaded, err := LoadNousOAuthCredentials(CredentialPoolOptions{HermesHome: hermesHome})
	if err != nil {
		t.Fatalf("LoadNousOAuthCredentials: %v", err)
	}
	if loaded.AccessToken != creds.AccessToken {
		t.Errorf("loaded.AccessToken = %q, want %q", loaded.AccessToken, creds.AccessToken)
	}
	if loaded.RefreshToken != creds.RefreshToken {
		t.Errorf("loaded.RefreshToken = %q, want %q", loaded.RefreshToken, creds.RefreshToken)
	}
	if loaded.AgentKey != creds.AgentKey {
		t.Errorf("loaded.AgentKey = %q, want %q", loaded.AgentKey, creds.AgentKey)
	}
	if loaded.PortalBaseURL != creds.PortalBaseURL {
		t.Errorf("loaded.PortalBaseURL = %q, want %q", loaded.PortalBaseURL, creds.PortalBaseURL)
	}
	if loaded.InferenceBaseURL != creds.InferenceBaseURL {
		t.Errorf("loaded.InferenceBaseURL = %q, want %q", loaded.InferenceBaseURL, creds.InferenceBaseURL)
	}
}

func TestNousOAuthLoadMissingStore(t *testing.T) {
	hermesHome := t.TempDir()
	_, err := LoadNousOAuthCredentials(CredentialPoolOptions{HermesHome: hermesHome})
	if err == nil {
		t.Fatal("expected error for missing auth store")
	}
}
