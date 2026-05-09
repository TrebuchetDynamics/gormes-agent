package install_test

import (
	"encoding/json"
	"regexp"
	"strings"
	"testing"
)

func TestLandingReleaseMetadataCarriesDateAlias(t *testing.T) {
	versionGo := readRepoFileRelease(t, "cmd/gormes/version.go")
	version := goStringVar(t, versionGo, "Version")
	dateAlias := goStringVar(t, versionGo, "VersionDateAlias")

	var release struct {
		Version   string `json:"version"`
		Tag       string `json:"tag"`
		DateAlias string `json:"date_alias"`
		URL       string `json:"url"`
		Source    string `json:"source"`
	}
	rawRelease := readRepoFileRelease(t, "webpages/landing/src/data/release.json")
	if err := json.Unmarshal([]byte(rawRelease), &release); err != nil {
		t.Fatalf("release.json must be valid JSON: %v\n%s", err, rawRelease)
	}
	if release.Version != version {
		t.Fatalf("release.json version = %q, want %q", release.Version, version)
	}
	if release.Tag != "v"+version {
		t.Fatalf("release.json tag = %q, want v%s", release.Tag, version)
	}
	if release.DateAlias != dateAlias {
		t.Fatalf("release.json date_alias = %q, want %q from cmd/gormes/version.go", release.DateAlias, dateAlias)
	}
	if release.Source != "cmd/gormes/version.go" {
		t.Fatalf("release.json source = %q, want cmd/gormes/version.go", release.Source)
	}

	syncAssets := readRepoFileRelease(t, "webpages/landing/scripts/sync-assets.mjs")
	if !strings.Contains(syncAssets, "VersionDateAlias") || !strings.Contains(syncAssets, "date_alias") {
		t.Fatalf("sync-assets.mjs must parse VersionDateAlias and write date_alias")
	}

	landingData := readRepoFileRelease(t, "webpages/landing/src/data/landing.js")
	for _, want := range []string{"release?.date_alias", "releaseDateAlias", "Current scout release:"} {
		if !strings.Contains(landingData, want) {
			t.Fatalf("landing.js missing %q; release label must be derived from release.json date_alias", want)
		}
	}
}

func goStringVar(t *testing.T, source, name string) string {
	t.Helper()
	re := regexp.MustCompile(`var\s+` + regexp.QuoteMeta(name) + `\s*=\s*"([^"]+)"`)
	match := re.FindStringSubmatch(source)
	if len(match) != 2 {
		t.Fatalf("could not find Go string var %s", name)
	}
	return match[1]
}
