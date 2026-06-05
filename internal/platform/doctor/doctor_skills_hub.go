package doctor

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/textvalue"
)

// SkillsHubOptions contains the local filesystem and auth probe inputs for
// the Skills Hub doctor check.
type SkillsHubOptions struct {
	Home            string
	Env             map[string]string
	RunGHAuthStatus GitHubAuthStatusRunner
	Offline         bool
}

// CheckSkillsHub inspects the Gormes-owned skills hub state under
// ~/.gormes/skills/.hub. It mirrors the upstream Hermes doctor Skills Hub
// section while keeping Gormes paths and command guidance.
func CheckSkillsHub(ctx context.Context, opts SkillsHubOptions) CheckResult {
	if ctx == nil {
		ctx = context.Background()
	}
	status := StatusPass
	items := make([]ItemInfo, 0, 4)
	hubDir := filepath.Join(opts.Home, "skills", ".hub")

	if dirExists(hubDir) {
		items = append(items, ItemInfo{Name: ".hub", Status: StatusPass, Note: "exists at ~/.gormes/skills/.hub"})
		lockItem := skillsHubLockItem(filepath.Join(hubDir, "lock.json"))
		items = append(items, lockItem)
		status = worstDoctorStatus(status, lockItem.Status)
		quarantineItem := skillsHubQuarantineItem(filepath.Join(hubDir, "quarantine"))
		items = append(items, quarantineItem)
		status = worstDoctorStatus(status, quarantineItem.Status)
	} else {
		hubItem := ItemInfo{Name: ".hub", Status: StatusWarn, Note: "not initialized (run: gormes skills list)"}
		items = append(items, hubItem)
		status = worstDoctorStatus(status, hubItem.Status)
	}

	githubItem := skillsHubGitHubItem(ctx, opts)
	items = append(items, githubItem)
	status = worstDoctorStatus(status, githubItem.Status)

	summary := "Skills Hub ready"
	if status == StatusWarn {
		summary = "Skills Hub has local setup warnings"
	}
	return CheckResult{Name: "Skills Hub", Status: status, Summary: summary, Items: items}
}

func skillsHubLockItem(path string) ItemInfo {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return ItemInfo{Name: "lock.json", Status: StatusPass, Note: "not written yet (0 hub-installed skill(s))"}
		}
		return ItemInfo{Name: "lock.json", Status: StatusWarn, Note: "corrupted or unreadable"}
	}
	var lock struct {
		Installed map[string]json.RawMessage `json:"installed"`
	}
	if err := json.Unmarshal(data, &lock); err != nil {
		return ItemInfo{Name: "lock.json", Status: StatusWarn, Note: "corrupted or unreadable"}
	}
	return ItemInfo{Name: "lock.json", Status: StatusPass, Note: fmt.Sprintf("Lock file OK (%d hub-installed skill(s))", len(lock.Installed))}
}

func skillsHubQuarantineItem(path string) ItemInfo {
	entries, err := os.ReadDir(path)
	if err != nil {
		if os.IsNotExist(err) {
			return ItemInfo{Name: "quarantine", Status: StatusPass, Note: "0 skill(s) in quarantine"}
		}
		return ItemInfo{Name: "quarantine", Status: StatusWarn, Note: "unreadable"}
	}
	count := 0
	for _, entry := range entries {
		if entry.IsDir() {
			count++
		}
	}
	if count > 0 {
		return ItemInfo{Name: "quarantine", Status: StatusWarn, Note: fmt.Sprintf("%d skill(s) in quarantine (pending review)", count)}
	}
	return ItemInfo{Name: "quarantine", Status: StatusPass, Note: "0 skill(s) in quarantine"}
}

func skillsHubGitHubItem(ctx context.Context, opts SkillsHubOptions) ItemInfo {
	if opts.Offline && !skillsHubHasEnvToken(opts.Env) {
		return ItemInfo{Name: "github", Status: StatusSkip, Note: "skipped (--offline; set GITHUB_TOKEN/GH_TOKEN for ~/.gormes/.env rate-limit relief)"}
	}
	auth := CheckGitHubAuth(ctx, GitHubAuthOptions{
		Env:             opts.Env,
		RunGHAuthStatus: opts.RunGHAuthStatus,
	})
	evidence := auth.Summary
	switch auth.Status {
	case StatusPass:
		if strings.Contains(auth.Summary, "gh CLI") {
			return ItemInfo{Name: "github", Status: StatusPass, Note: "GitHub authenticated via gh CLI (full API access - no GITHUB_TOKEN needed) " + evidence}
		}
		return ItemInfo{Name: "github", Status: StatusPass, Note: "GitHub token configured (authenticated API access) " + evidence}
	case StatusWarn:
		return ItemInfo{Name: "github", Status: StatusWarn, Note: "No GITHUB_TOKEN/GH_TOKEN (60 req/hr rate limit - set in ~/.gormes/.env for better rates) " + evidence}
	default:
		return ItemInfo{Name: "github", Status: auth.Status, Note: evidence}
	}
}

func skillsHubHasEnvToken(env map[string]string) bool {
	return textvalue.IsNonBlank(env["GITHUB_TOKEN"]) || textvalue.IsNonBlank(env["GH_TOKEN"])
}

func worstDoctorStatus(current, next Status) Status {
	if current == StatusFail || next == StatusFail {
		return StatusFail
	}
	if current == StatusWarn || next == StatusWarn {
		return StatusWarn
	}
	return current
}
