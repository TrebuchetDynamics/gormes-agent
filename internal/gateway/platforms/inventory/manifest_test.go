package inventory

import (
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

func TestHermesGatewayPlatformManifestCoversUpstream(t *testing.T) {
	manifest := HermesGatewayPlatformManifest()
	if len(manifest) == 0 {
		t.Fatal("manifest is empty")
	}
	byID := make(map[string]PlatformManifestEntry, len(manifest))
	for _, entry := range manifest {
		if entry.ID == "" {
			t.Fatalf("manifest entry has empty id: %+v", entry)
		}
		if _, exists := byID[entry.ID]; exists {
			t.Fatalf("duplicate platform manifest id %q", entry.ID)
		}
		byID[entry.ID] = entry
		if entry.DisplayName == "" || entry.HermesSource == "" || entry.GormesSurface == "" || entry.BacklogOwner == "" {
			t.Fatalf("manifest entry %q missing source/evidence fields: %+v", entry.ID, entry)
		}
		if !validPlatformImplementationStatus(entry.Status) {
			t.Fatalf("manifest entry %q has invalid status %q", entry.ID, entry.Status)
		}
		for field, status := range map[string]PlatformSurfaceStatus{
			"inbound": entry.Inbound, "outbound": entry.Outbound, "media": entry.Media, "commands": entry.Commands,
			"toolset": entry.Toolset, "config": entry.Config, "pairing": entry.Pairing, "delivery": entry.Delivery,
		} {
			if !validPlatformSurfaceStatus(status) {
				t.Fatalf("manifest entry %q has invalid %s status %q", entry.ID, field, status)
			}
		}
	}

	upstreamPlatforms, err := readHermesPlatformEnumIDs()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			t.Skipf("upstream hermes-agent checkout not present: %v", err)
		}
		t.Fatal(err)
	}
	for _, id := range upstreamPlatforms {
		if _, ok := byID[id]; !ok {
			t.Fatalf("manifest missing upstream Platform enum value %q; have %v", id, sortedKeys(byID))
		}
	}
	if len(byID) != len(upstreamPlatforms) {
		extras := manifestEntriesOutsideUpstream(byID, upstreamPlatforms)
		if len(extras) == 0 {
			t.Fatalf("manifest has %d entries, upstream Platform enum has %d entries: manifest=%v upstream=%v", len(byID), len(upstreamPlatforms), sortedKeys(byID), upstreamPlatforms)
		}
		for _, id := range extras {
			entry := byID[id]
			if !strings.HasPrefix(entry.HermesSource, "plugins/platforms/") {
				t.Fatalf("manifest extra %q must be source-backed by plugins/platforms, got %q", id, entry.HermesSource)
			}
			if err := assertHermesPluginPlatformSourceExists(entry.HermesSource); err != nil {
				t.Fatalf("manifest extra %q has invalid plugin source: %v", id, err)
			}
		}
	}

	connectors, err := readHermesGatewayConnectorIDs()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			t.Skipf("upstream hermes-agent gateway/platforms not present: %v", err)
		}
		t.Fatal(err)
	}
	for _, id := range connectors {
		entry, ok := byID[id]
		if !ok {
			t.Fatalf("manifest missing connector %q from gateway/platforms", id)
		}
		if entry.HermesSource == "gateway/config.py:Platform."+strings.ToUpper(id) {
			t.Fatalf("manifest connector %q should name its connector source file, got %q", id, entry.HermesSource)
		}
	}
}

func TestHermesGatewayPlatformManifestCoversBundledPluginPlatforms(t *testing.T) {
	manifest := HermesGatewayPlatformManifest()
	byID := make(map[string]PlatformManifestEntry, len(manifest))
	for _, entry := range manifest {
		byID[entry.ID] = entry
	}

	plugins, err := readHermesBundledPlatformPluginIDs()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			t.Skipf("upstream hermes-agent bundled platform plugins not present: %v", err)
		}
		t.Fatal(err)
	}
	if len(plugins) == 0 {
		t.Fatal("no bundled platform plugins discovered")
	}

	for _, id := range plugins {
		entry, ok := byID[id]
		if !ok {
			t.Fatalf("manifest missing bundled platform plugin %q; have %v", id, sortedKeys(byID))
		}
		if entry.Kind != PlatformKindChannel {
			t.Fatalf("plugin platform %q kind = %q, want %q", id, entry.Kind, PlatformKindChannel)
		}
		if !entry.RequiresLiveCredentials {
			t.Fatalf("plugin platform %q must require live credentials", id)
		}
		wantSource := "plugins/platforms/" + id + "/adapter.py"
		if entry.HermesSource != wantSource {
			t.Fatalf("plugin platform %q HermesSource = %q, want %q", id, entry.HermesSource, wantSource)
		}
		if entry.Status == PlatformStatusImplemented {
			t.Fatalf("plugin platform %q must not be marked implemented without adapter fixtures", id)
		}
	}
}

func manifestEntriesOutsideUpstream(byID map[string]PlatformManifestEntry, upstream []string) []string {
	upstreamSet := map[string]bool{}
	for _, id := range upstream {
		upstreamSet[id] = true
	}
	var extras []string
	for id := range byID {
		if !upstreamSet[id] {
			extras = append(extras, id)
		}
	}
	sort.Strings(extras)
	return extras
}

func assertHermesPluginPlatformSourceExists(source string) error {
	root, err := workspaceRoot()
	if err != nil {
		return err
	}
	info, err := os.Stat(filepath.Join(root, "hermes-agent", filepath.FromSlash(source)))
	if err != nil {
		return err
	}
	if info.IsDir() {
		return os.ErrInvalid
	}
	return nil
}

func readHermesBundledPlatformPluginIDs() ([]string, error) {
	workspace, err := workspaceRoot()
	if err != nil {
		return nil, err
	}
	root := filepath.Join(workspace, "hermes-agent", "plugins", "platforms")
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	var ids []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		id := entry.Name()
		data, err := os.ReadFile(filepath.Join(root, id, "plugin.yaml"))
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return nil, err
		}
		if yamlScalarValue(data, "kind") != "platform" {
			continue
		}
		if _, err := os.Stat(filepath.Join(root, id, "adapter.py")); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids, nil
}

func yamlScalarValue(data []byte, key string) string {
	prefix := key + ":"
	for _, line := range strings.Split(string(data), "\n") {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, prefix) {
			continue
		}
		return strings.Trim(strings.TrimSpace(strings.TrimPrefix(trimmed, prefix)), `"'`)
	}
	return ""
}

func TestHermesGatewayPlatformManifestReturnsCopy(t *testing.T) {
	first := HermesGatewayPlatformManifest()
	second := HermesGatewayPlatformManifest()
	if len(first) == 0 || len(second) == 0 {
		t.Fatal("manifest is empty")
	}
	first[0].ID = "mutated"
	if second[0].ID == "mutated" || HermesGatewayPlatformManifest()[0].ID == "mutated" {
		t.Fatal("HermesGatewayPlatformManifest must return a defensive copy")
	}
}

func readHermesPlatformEnumIDs() ([]string, error) {
	root, err := workspaceRoot()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(filepath.Join(root, "hermes-agent", "gateway", "config.py"))
	if err != nil {
		return nil, err
	}
	lines := strings.Split(string(data), "\n")
	var ids []string
	inEnum := false
	re := regexp.MustCompile(`^\s{4}[A-Z0-9_]+\s*=\s*"([a-z0-9_]+)"`)
	for _, line := range lines {
		if strings.HasPrefix(line, "class Platform(Enum):") {
			inEnum = true
			continue
		}
		if inEnum && strings.HasPrefix(line, "@dataclass") {
			break
		}
		if !inEnum {
			continue
		}
		match := re.FindStringSubmatch(line)
		if len(match) == 2 {
			ids = append(ids, match[1])
		}
	}
	if len(ids) == 0 {
		return nil, os.ErrNotExist
	}
	return ids, nil
}

func readHermesGatewayConnectorIDs() ([]string, error) {
	workspace, err := workspaceRoot()
	if err != nil {
		return nil, err
	}
	root := filepath.Join(workspace, "hermes-agent", "gateway", "platforms")
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	ignore := map[string]bool{
		"__init__":             true,
		"base":                 true,
		"helpers":              true,
		"telegram_network":     true,
		"signal_rate_limit":    true,
		"wecom_crypto":         true,
		"feishu_comment":       true,
		"feishu_comment_rules": true,
		"yuanbao_media":        true,
		"yuanbao_proto":        true,
		"yuanbao_sticker":      true,
	}
	var ids []string
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() {
			if name == "qqbot" {
				ids = append(ids, name)
			}
			continue
		}
		if !strings.HasSuffix(name, ".py") {
			continue
		}
		id := strings.TrimSuffix(name, ".py")
		if strings.HasPrefix(id, "_") {
			continue
		}
		if ignore[id] {
			continue
		}
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids, nil
}

func workspaceRoot() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for dir := wd; dir != filepath.Dir(dir); dir = filepath.Dir(dir) {
		if hasHermesGatewayConfig(dir) {
			return dir, nil
		}
	}
	for _, dir := range []string{
		"/home/xel/git/sages-openclaw/workspace-gormes",
		"/home/xel/git/sages-openclaw/workspace-mineru",
	} {
		if hasHermesGatewayConfig(dir) {
			return dir, nil
		}
	}
	return "", os.ErrNotExist
}

func hasHermesGatewayConfig(dir string) bool {
	info, err := os.Stat(filepath.Join(dir, "hermes-agent", "gateway", "config.py"))
	return err == nil && !info.IsDir()
}

func validPlatformImplementationStatus(status PlatformImplementationStatus) bool {
	switch status {
	case PlatformStatusImplemented, PlatformStatusPartial, PlatformStatusRowBacked, PlatformStatusExcluded, PlatformStatusOwned:
		return true
	default:
		return false
	}
}

func validPlatformSurfaceStatus(status PlatformSurfaceStatus) bool {
	switch status {
	case PlatformSurfaceImplemented, PlatformSurfacePartial, PlatformSurfaceRowBacked, PlatformSurfaceNotApplicable, PlatformSurfaceOwned:
		return true
	default:
		return false
	}
}

func sortedKeys[K ~string, V any](m map[K]V) []string {
	out := make([]string, 0, len(m))
	for key := range m {
		out = append(out, string(key))
	}
	sort.Strings(out)
	return out
}
