package doctor

import (
	"fmt"
	"sort"
	"strings"
)

// DoctorProfileInventory is the small caller-facing seam the ◆ Profiles
// check needs. cmd/gormes wires it from the real profileCommandSeams
// (ListKnownProfiles / ReadActiveProfileName / ResolveProfileRoot /
// ReadDistributionManifest); tests inject a hermetic fake. This keeps
// internal/doctor free of cli/cmd profile dependencies.
//
// Owned divergence (parity intent hermes_cli/doctor.py@55c9f3206:1768):
// Gormes profiles are ~/.gormes/profiles/<name> subdirs with profile-local
// config, optional recorded gateway state, an optional distribution manifest,
// and an active marker. There is intentionally NO shell-wrapper alias or
// orphan-alias fabrication here.
type DoctorProfileInventory struct {
	// Known are the profile names from profileCommandSeams.ListKnownProfiles().
	// "default" is added automatically if absent.
	Known []string
	// Active is the active profile name ("" or "default" → the default).
	Active string
	// RootExists reports whether a profile's resolved root directory
	// exists. nil → treated as unknown-but-present (no false WARN).
	RootExists func(name string) bool
	// HasManifest reports whether a profile has a distribution manifest.
	// Absent is informational, never a WARN.
	HasManifest func(name string) bool
	// Distribution reports optional distribution manifest summary/error.
	// Present manifests are informational; invalid manifests warn for that
	// profile.
	Distribution func(name string) DoctorProfileDistribution
	// Config reports the effective local provider/model read from the
	// profile's own config files and defaults only. Missing provider/model
	// warns for named profiles; default-only stays a clean orientation PASS.
	Config func(name string) DoctorProfileConfig
	// Gateway reports the profile's recorded gateway_state.json state only.
	// It is not live PID proof.
	Gateway func(name string) DoctorProfileGateway
}

type DoctorProfileConfig struct {
	Present  bool
	Provider string
	Model    string
	Error    string
}

type DoctorProfileGateway struct {
	Present bool
	State   string
	Error   string
}

type DoctorProfileDistribution struct {
	Present bool
	Summary string
	Error   string
}

// CheckProfiles renders the Gormes-owned ◆ Profiles section content. With no
// named profiles it is a single clean PASS ("default profile only") — never
// WARN, since Gormes always has a usable default. With named profiles, the
// inventory includes default plus each named profile. Missing profile roots,
// unreadable config, missing provider/model, invalid manifests, and corrupt
// recorded gateway state warn for that item only. An absent distribution
// manifest and absent gateway state are informational PASS details.
func CheckProfiles(inv DoctorProfileInventory) CheckResult {
	active := strings.TrimSpace(inv.Active)
	if active == "" {
		active = "default"
	}

	profiles := normalizeDoctorProfiles(inv.Known)
	defaultOnly := len(profiles) == 1 && profiles[0] == "default"
	if defaultOnly {
		note := "default profile only (create more with `gormes profile create`)"
		if detail := doctorProfileDetail("default", inv, true); detail != "" {
			note += "; " + detail
		}
		return CheckResult{
			Name:    "Profiles",
			Status:  StatusPass,
			Summary: "default profile only",
			Items: []ItemInfo{
				{Name: "default", Status: StatusPass, Note: note},
			},
		}
	}

	worst := StatusPass
	items := make([]ItemInfo, 0, len(profiles)+1)
	items = append(items, ItemInfo{
		Name:   "summary",
		Status: StatusPass,
		Note:   fmt.Sprintf("%d profile(s) found (active: %s)", len(profiles), active),
	})
	for _, name := range profiles {
		label := name
		if name == active {
			label = name + " (active)"
		}
		status, note := doctorProfileItemStatus(name, inv, false)
		if status == StatusWarn {
			worst = StatusWarn
		}
		items = append(items, ItemInfo{Name: label, Status: status, Note: note})
	}

	summary := fmt.Sprintf("%d profile(s) found", len(profiles))
	if worst == StatusWarn {
		summary += "; some profile details need attention"
	}
	return CheckResult{Name: "Profiles", Status: worst, Summary: summary, Items: items}
}

func normalizeDoctorProfiles(known []string) []string {
	seen := map[string]struct{}{"default": {}}
	named := make([]string, 0, len(known))
	for _, n := range known {
		n = strings.TrimSpace(n)
		if n == "" || n == "default" {
			continue
		}
		if _, ok := seen[n]; ok {
			continue
		}
		seen[n] = struct{}{}
		named = append(named, n)
	}
	sort.Strings(named)
	return append([]string{"default"}, named...)
}

func doctorProfileItemStatus(name string, inv DoctorProfileInventory, defaultOnly bool) (Status, string) {
	status := StatusPass
	parts := []string{}

	rootMissing := inv.RootExists != nil && !inv.RootExists(name)
	if rootMissing {
		status = StatusWarn
		parts = append(parts, "root missing — recreate with `gormes profile create`")
	} else {
		parts = append(parts, "root=set")
	}

	if inv.Config != nil && !rootMissing {
		cfg := inv.Config(name)
		switch {
		case cfg.Error != "":
			status = StatusWarn
			parts = append(parts, "config unreadable")
		case !cfg.Present:
			if !defaultOnly {
				status = StatusWarn
				parts = append(parts, "config missing")
			}
		case strings.TrimSpace(cfg.Provider) == "" || strings.TrimSpace(cfg.Model) == "":
			if !defaultOnly {
				status = StatusWarn
			}
			parts = append(parts, "provider/model missing")
		default:
			parts = append(parts, "provider="+strings.TrimSpace(cfg.Provider), "model="+strings.TrimSpace(cfg.Model))
		}
	}

	if inv.Gateway != nil && !rootMissing {
		gw := inv.Gateway(name)
		switch {
		case gw.Error != "":
			status = StatusWarn
			parts = append(parts, "gateway=unreadable")
		case gw.Present:
			state := strings.TrimSpace(gw.State)
			if state == "" {
				state = "unknown"
			}
			parts = append(parts, "gateway=recorded "+state)
		default:
			parts = append(parts, "gateway=not recorded")
		}
	}

	dist := doctorProfileDistribution(name, inv)
	switch {
	case dist.Error != "":
		status = StatusWarn
		parts = append(parts, "distribution manifest invalid")
	case dist.Present:
		summary := strings.TrimSpace(dist.Summary)
		if summary == "" {
			summary = "present"
		}
		parts = append(parts, "distribution="+summary)
	default:
		parts = append(parts, "no distribution manifest")
	}

	return status, strings.Join(parts, "; ")
}

func doctorProfileDetail(name string, inv DoctorProfileInventory, defaultOnly bool) string {
	_, detail := doctorProfileItemStatus(name, inv, defaultOnly)
	return detail
}

func doctorProfileDistribution(name string, inv DoctorProfileInventory) DoctorProfileDistribution {
	if inv.Distribution != nil {
		return inv.Distribution(name)
	}
	if inv.HasManifest != nil && inv.HasManifest(name) {
		return DoctorProfileDistribution{Present: true, Summary: "present"}
	}
	return DoctorProfileDistribution{}
}
