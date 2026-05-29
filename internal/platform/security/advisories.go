package security

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// Security advisory subsystem for Gormes.
//
// Parity intent: hermes_cli/security_advisories.py@55c9f3206 (detect_compromised
// / filter_unacked / full_remediation_text / get_acked_ids / ack_advisory).
//
// OWNED DIVERGENCE: upstream detect_compromised scans the active Python venv
// via importlib.metadata. Gormes is a single Go binary with no Python venv, so
// that scan CANNOT be faithfully reproduced and is NOT fabricated. Detection
// here is an injectable seam (PackageVersionFunc). The default seam reports no
// package as installed, so the catalog produces no hits — exactly Hermes'
// "loud when it matters, silent otherwise" behavior, not a stub. The advisory
// catalog itself is carried verbatim as DATA so the contract, remediation copy,
// and ack flow are real and a future Gormes-relevant detector can be wired
// without re-architecting. The ack store is Gormes-owned under ~/.gormes (a
// JSON file), never the Python config.yaml security.acked_advisories list.

// CompromisedPackage names a package and the version set known to be
// compromised. An empty Versions slice means "any installed version is
// suspect" (rare; only when the maintainer namespace itself is compromised).
type CompromisedPackage struct {
	Package  string
	Versions []string
}

// Advisory mirrors the upstream Advisory dataclass shape.
type Advisory struct {
	ID          string
	Title       string
	Summary     string
	URL         string
	Compromised []CompromisedPackage
	Remediation []string
	Published   string
	Severity    string // low / medium / high / critical
}

// AdvisoryHit is one package-version match against an advisory.
type AdvisoryHit struct {
	Advisory         Advisory
	Package          string
	InstalledVersion string
}

// PackageVersionFunc reports the installed version of pkg, or "" if it is not
// present. This is the injectable detector seam (see the owned-divergence note
// above): in a pure-Go runtime the default returns "" for everything.
type PackageVersionFunc func(pkg string) string

// NoInstalledPackages is the default detector seam for a pure-Go Gormes
// runtime: nothing is reported installed, so the catalog yields no hits.
func NoInstalledPackages(string) string { return "" }

// DefaultCatalog carries the upstream advisory data verbatim. The
// remediation text (including any `pip uninstall ...` line) is upstream
// advisory CONTENT, not Gormes scaffolding wording. Do not remove old
// advisories — older releases with the compromised dependency still warn.
func DefaultCatalog() []Advisory {
	return []Advisory{
		{
			ID:    "shai-hulud-2026-05",
			Title: "Mini Shai-Hulud worm — mistralai 2.4.6 compromised on PyPI",
			Summary: "PyPI quarantined the mistralai package on 2026-05-12 after a " +
				"malicious 2.4.6 release. The worm steals credentials from " +
				"environment variables and credential files (~/.npmrc, ~/.pypirc, " +
				"~/.aws/credentials, GitHub PATs, cloud SDK tokens) and exfils " +
				"them to a hardcoded webhook. If you ran any process that " +
				"imported mistralai 2.4.6, assume those credentials are exposed.",
			URL: "https://socket.dev/blog/mini-shai-hulud-worm-pypi",
			Compromised: []CompromisedPackage{
				{Package: "mistralai", Versions: []string{"2.4.6"}},
			},
			Remediation: []string{
				"Run: pip uninstall -y mistralai  (or: uv pip uninstall mistralai)",
				"Rotate API keys in your credential store (OpenRouter, Anthropic, " +
					"OpenAI, Nous, GitHub, AWS, Google, Mistral, etc.).",
				"Audit ~/.npmrc, ~/.pypirc, ~/.aws/credentials, ~/.config/gh/hosts.yml, " +
					"and any other credential files for tokens that may have been read.",
				"Check GitHub for unexpected new SSH keys, deploy keys, or webhook " +
					"additions on repos you have admin on.",
				"After cleanup: gormes doctor --ack shai-hulud-2026-05  to dismiss " +
					"this warning.",
			},
			Published: "2026-05-12",
			Severity:  "critical",
		},
	}
}

// DetectCompromised scans the catalog against the injected detector seam and
// returns every hit. A hit means the advisory's package is "installed" (the
// seam returns a non-empty version) AND the version is in the compromised set
// (or the compromised set is empty, meaning any version is suspect).
func DetectCompromised(catalog []Advisory, installed PackageVersionFunc) []AdvisoryHit {
	if installed == nil {
		installed = NoInstalledPackages
	}
	var hits []AdvisoryHit
	for _, adv := range catalog {
		for _, cp := range adv.Compromised {
			ver := strings.TrimSpace(installed(cp.Package))
			if ver == "" {
				continue
			}
			if len(cp.Versions) == 0 || containsString(cp.Versions, ver) {
				hits = append(hits, AdvisoryHit{
					Advisory:         adv,
					Package:          cp.Package,
					InstalledVersion: ver,
				})
			}
		}
	}
	return hits
}

// FilterUnacked returns only hits whose advisory the user has not dismissed.
func FilterUnacked(hits []AdvisoryHit, acked map[string]struct{}) []AdvisoryHit {
	if len(hits) == 0 {
		return nil
	}
	out := make([]AdvisoryHit, 0, len(hits))
	for _, h := range hits {
		if _, ok := acked[h.Advisory.ID]; ok {
			continue
		}
		out = append(out, h)
	}
	return out
}

// FullRemediationText renders an advisory + remediation block. The advisory
// content is upstream DATA and is rendered verbatim.
func FullRemediationText(hit AdvisoryHit) []string {
	a := hit.Advisory
	lines := []string{
		"=== " + a.Title + " ===",
		"ID:        " + a.ID + "    Severity: " + a.Severity + "    Published: " + a.Published,
		"Detected:  " + hit.Package + "==" + hit.InstalledVersion,
		"Reference: " + a.URL,
		"",
		a.Summary,
		"",
		"Remediation:",
	}
	for i, step := range a.Remediation {
		lines = append(lines, "  "+strconv.Itoa(i+1)+". "+step)
	}
	return lines
}

// AckStore persists dismissed advisory IDs under the Gormes-owned home
// (~/.gormes/security/acked_advisories.json). The ID list is the only state —
// no per-host data, no timestamps (mirrors the upstream "the list is the only
// state" design), but Gormes-owned location, never the Python config.yaml.
type AckStore struct {
	dir string
}

type ackFile struct {
	AckedAdvisories []string `json:"acked_advisories"`
}

// NewAckStore returns an ack store rooted at the injected Gormes home dir
// (mirrors CheckDirectoryStructure(home) — internal/security stays
// config-decoupled; cmd/gormes wires config.GormesHome()).
func NewAckStore(gormesHome string) *AckStore {
	return &AckStore{dir: filepath.Join(gormesHome, "security")}
}

func (s *AckStore) path() string {
	return filepath.Join(s.dir, "acked_advisories.json")
}

// AckedIDs returns the dismissed advisory IDs. A missing/unreadable store is
// an empty set with no error (a broken ack file must never block doctor or
// hide an active advisory — it just keeps firing until repaired).
func (s *AckStore) AckedIDs() (map[string]struct{}, error) {
	out := map[string]struct{}{}
	raw, err := os.ReadFile(s.path())
	if err != nil {
		return out, nil
	}
	var f ackFile
	if err := json.Unmarshal(raw, &f); err != nil {
		return out, nil
	}
	for _, id := range f.AckedAdvisories {
		id = strings.TrimSpace(id)
		if id != "" {
			out[id] = struct{}{}
		}
	}
	return out, nil
}

// Ack persists a dismissal for advisoryID. Idempotent — acking an
// already-acked ID is a no-op success.
func (s *AckStore) Ack(advisoryID string) error {
	advisoryID = strings.TrimSpace(advisoryID)
	if advisoryID == "" {
		return nil
	}
	if err := os.MkdirAll(s.dir, 0o700); err != nil {
		return err
	}
	acked, _ := s.AckedIDs()
	if _, ok := acked[advisoryID]; ok {
		return nil
	}
	ids := make([]string, 0, len(acked)+1)
	for id := range acked {
		ids = append(ids, id)
	}
	ids = append(ids, advisoryID)
	sort.Strings(ids)
	body, err := json.MarshalIndent(ackFile{AckedAdvisories: ids}, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path(), body, 0o600)
}

func containsString(haystack []string, needle string) bool {
	for _, h := range haystack {
		if h == needle {
			return true
		}
	}
	return false
}
