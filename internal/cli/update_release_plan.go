package cli

import (
	"fmt"
	"strconv"
	"strings"
)

type UpdateInstallKind string

const (
	UpdateInstallKindRelease         UpdateInstallKind = "release"
	UpdateInstallKindManagedSource   UpdateInstallKind = "managed_source"
	UpdateInstallKindUnmanagedSource UpdateInstallKind = "unmanaged_source"
	UpdateInstallKindUnknown         UpdateInstallKind = "unknown"
)

type UpdateReleaseChannel string

const (
	UpdateReleaseChannelStable      UpdateReleaseChannel = "stable"
	UpdateReleaseChannelDevelopment UpdateReleaseChannel = "development"
)

type UpdateSource string

const (
	UpdateSourceGitHubRelease UpdateSource = "github_release"
	UpdateSourceManagedSource UpdateSource = "managed_source"
	UpdateSourceUnknown       UpdateSource = "unknown"
)

type UpdateBuildIdentity struct {
	Version   string `json:"version"`
	GitCommit string `json:"git_commit,omitempty"`
}

type UpdateReleaseMetadata struct {
	Version   string `json:"version"`
	Tag       string `json:"tag,omitempty"`
	GitCommit string `json:"git_commit,omitempty"`
}

type UpdateReleaseComponent string

const (
	UpdateReleaseComponentSnapshot       UpdateReleaseComponent = "snapshot"
	UpdateReleaseComponentBinary         UpdateReleaseComponent = "binary"
	UpdateReleaseComponentChecksum       UpdateReleaseComponent = "checksum"
	UpdateReleaseComponentManifest       UpdateReleaseComponent = "manifest"
	UpdateReleaseComponentAssets         UpdateReleaseComponent = "assets"
	UpdateReleaseComponentSkills         UpdateReleaseComponent = "skills"
	UpdateReleaseComponentSourceCheckout UpdateReleaseComponent = "source_checkout"
)

type UpdateReleaseBlockerKind string

const (
	UpdateReleaseBlockerUnknownInstallLayout    UpdateReleaseBlockerKind = "unknown_install_layout"
	UpdateReleaseBlockerUnsupportedPlatform     UpdateReleaseBlockerKind = "unsupported_platform"
	UpdateReleaseBlockerMissingReleaseMetadata  UpdateReleaseBlockerKind = "missing_release_metadata"
	UpdateReleaseBlockerChannelMismatch         UpdateReleaseBlockerKind = "channel_mismatch"
	UpdateReleaseBlockerDirtyUnmanagedSource    UpdateReleaseBlockerKind = "dirty_unmanaged_source_checkout"
	UpdateReleaseBlockerUnsupportedInstallState UpdateReleaseBlockerKind = "unsupported_install_state"
)

type UpdateReleaseBlocker struct {
	Kind   UpdateReleaseBlockerKind `json:"kind"`
	Detail string                   `json:"detail,omitempty"`
}

type UpdateReleasePlanOptions struct {
	InstallKind   UpdateInstallKind
	Channel       UpdateReleaseChannel
	Current       UpdateBuildIdentity
	Target        UpdateReleaseMetadata
	MetadataError error
	GOOS          string
	GOARCH        string
	SnapshotPath  string
	DirtySource   bool
}

type UpdateReleasePlan struct {
	InstallKind     UpdateInstallKind        `json:"install_kind"`
	Source          UpdateSource             `json:"source"`
	Channel         UpdateReleaseChannel     `json:"channel"`
	Current         UpdateBuildIdentity      `json:"current"`
	Target          UpdateReleaseMetadata    `json:"target"`
	ArtifactName    string                   `json:"artifact_name,omitempty"`
	SnapshotPath    string                   `json:"snapshot_path,omitempty"`
	Components      []UpdateReleaseComponent `json:"components,omitempty"`
	UpdateAvailable bool                     `json:"update_available"`
	Blockers        []UpdateReleaseBlocker   `json:"blockers,omitempty"`
}

func BuildUpdateReleasePlan(opts UpdateReleasePlanOptions) UpdateReleasePlan {
	kind := opts.InstallKind
	if kind == "" {
		kind = UpdateInstallKindUnknown
	}
	channel := opts.Channel
	if channel == "" {
		channel = UpdateReleaseChannelStable
	}
	target := opts.Target
	if target.Version == "" && target.Tag != "" {
		target.Version = strings.TrimPrefix(strings.TrimSpace(target.Tag), "v")
	}

	plan := UpdateReleasePlan{
		InstallKind:  kind,
		Source:       updateSourceForInstallKind(kind),
		Channel:      channel,
		Current:      opts.Current,
		Target:       target,
		SnapshotPath: opts.SnapshotPath,
	}

	switch kind {
	case UpdateInstallKindRelease:
		plan.Components = append(plan.Components,
			UpdateReleaseComponentSnapshot,
			UpdateReleaseComponentBinary,
			UpdateReleaseComponentChecksum,
			UpdateReleaseComponentManifest,
			UpdateReleaseComponentAssets,
			UpdateReleaseComponentSkills,
		)
	case UpdateInstallKindManagedSource:
		plan.Components = append(plan.Components, UpdateReleaseComponentSourceCheckout)
	case UpdateInstallKindUnmanagedSource:
		plan.Components = append(plan.Components, UpdateReleaseComponentSourceCheckout)
	default:
		plan.addBlocker(UpdateReleaseBlockerUnknownInstallLayout, "could not classify this Gormes install layout")
	}

	if plan.Source == UpdateSourceGitHubRelease || channel == UpdateReleaseChannelStable {
		if slug := ReleaseArtifactPlatformSlug(opts.GOOS, opts.GOARCH); slug != "" && target.Version != "" {
			plan.ArtifactName = fmt.Sprintf("gormes-%s-%s.tar.gz", strings.TrimPrefix(target.Version, "v"), slug)
		} else if slug := ReleaseArtifactPlatformSlug(opts.GOOS, opts.GOARCH); slug == "" {
			plan.addBlocker(UpdateReleaseBlockerUnsupportedPlatform, fmt.Sprintf("%s/%s has no published release artifact", opts.GOOS, opts.GOARCH))
		}
	}

	if opts.MetadataError != nil {
		plan.addBlocker(UpdateReleaseBlockerMissingReleaseMetadata, opts.MetadataError.Error())
	} else if plan.Source == UpdateSourceGitHubRelease && target.Version == "" {
		plan.addBlocker(UpdateReleaseBlockerMissingReleaseMetadata, "latest GitHub release metadata did not include a version")
	}

	if channel == UpdateReleaseChannelStable && kind != UpdateInstallKindRelease {
		plan.addBlocker(UpdateReleaseBlockerChannelMismatch, "stable channel updates require a release install; source checkouts use --channel development")
	}
	if channel == UpdateReleaseChannelDevelopment && kind == UpdateInstallKindRelease {
		plan.addBlocker(UpdateReleaseBlockerChannelMismatch, "development channel updates require a managed source checkout")
	}
	if kind == UpdateInstallKindUnmanagedSource && opts.DirtySource {
		plan.addBlocker(UpdateReleaseBlockerDirtyUnmanagedSource, "unmanaged source checkout has local changes")
	}

	plan.UpdateAvailable = updateReleaseTargetNewer(plan.Current, plan.Target)
	return plan
}

func (p *UpdateReleasePlan) addBlocker(kind UpdateReleaseBlockerKind, detail string) {
	p.Blockers = append(p.Blockers, UpdateReleaseBlocker{Kind: kind, Detail: detail})
}

func updateSourceForInstallKind(kind UpdateInstallKind) UpdateSource {
	switch kind {
	case UpdateInstallKindRelease:
		return UpdateSourceGitHubRelease
	case UpdateInstallKindManagedSource, UpdateInstallKindUnmanagedSource:
		return UpdateSourceManagedSource
	default:
		return UpdateSourceUnknown
	}
}

func ReleaseArtifactPlatformSlug(goos, goarch string) string {
	switch goos + "/" + goarch {
	case "linux/amd64":
		return "linux-amd64"
	case "linux/arm64":
		return "linux-arm64"
	case "darwin/amd64":
		return "darwin-amd64"
	case "darwin/arm64":
		return "darwin-arm64"
	case "windows/amd64":
		return "windows-amd64"
	case "windows/arm64":
		return "windows-arm64"
	case "android/arm64":
		return "android-arm64"
	default:
		return ""
	}
}

func updateReleaseTargetNewer(current UpdateBuildIdentity, target UpdateReleaseMetadata) bool {
	cmp := compareUpdateVersions(current.Version, target.Version)
	if cmp < 0 {
		return true
	}
	if cmp > 0 {
		return false
	}
	return false
}

func compareUpdateVersions(left, right string) int {
	left = normalizeUpdateVersion(left)
	right = normalizeUpdateVersion(right)
	if left == right {
		return 0
	}
	leftParts := strings.Split(left, ".")
	rightParts := strings.Split(right, ".")
	max := len(leftParts)
	if len(rightParts) > max {
		max = len(rightParts)
	}
	for i := 0; i < max; i++ {
		l, lok := numericVersionPart(leftParts, i)
		r, rok := numericVersionPart(rightParts, i)
		if lok && rok {
			if l < r {
				return -1
			}
			if l > r {
				return 1
			}
			continue
		}
		ls := versionPart(leftParts, i)
		rs := versionPart(rightParts, i)
		if ls < rs {
			return -1
		}
		if ls > rs {
			return 1
		}
	}
	return 0
}

func normalizeUpdateVersion(value string) string {
	value = strings.TrimSpace(strings.TrimPrefix(value, "v"))
	for _, sep := range []string{"-", "+"} {
		if idx := strings.Index(value, sep); idx >= 0 {
			value = value[:idx]
		}
	}
	return value
}

func numericVersionPart(parts []string, i int) (int, bool) {
	part := versionPart(parts, i)
	if part == "" {
		return 0, true
	}
	n, err := strconv.Atoi(part)
	return n, err == nil
}

func versionPart(parts []string, i int) string {
	if i >= len(parts) {
		return "0"
	}
	return parts[i]
}
