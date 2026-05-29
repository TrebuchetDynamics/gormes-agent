package cli

import (
	"errors"
	"testing"
)

func TestUpdateReleasePlannerReleaseInstallAvailable(t *testing.T) {
	plan := BuildUpdateReleasePlan(UpdateReleasePlanOptions{
		InstallKind:  UpdateInstallKindRelease,
		Channel:      UpdateReleaseChannelStable,
		Current:      UpdateBuildIdentity{Version: "0.2.12", GitCommit: "oldsha"},
		Target:       UpdateReleaseMetadata{Version: "0.2.13", Tag: "v0.2.13", GitCommit: "newsha"},
		GOOS:         "linux",
		GOARCH:       "amd64",
		SnapshotPath: "/home/xel/.gormes/snapshots/20260515-201545-pre-update",
	})

	if plan.InstallKind != UpdateInstallKindRelease || plan.Source != UpdateSourceGitHubRelease {
		t.Fatalf("layout = %s/%s, want release/github_release", plan.InstallKind, plan.Source)
	}
	if !plan.UpdateAvailable {
		t.Fatalf("UpdateAvailable = false, want true; plan=%+v", plan)
	}
	if plan.Current.Version != "0.2.12" || plan.Target.Version != "0.2.13" || plan.Target.GitCommit != "newsha" {
		t.Fatalf("identity = current=%+v target=%+v, want current and target build identities", plan.Current, plan.Target)
	}
	if plan.ArtifactName != "gormes-0.2.13-linux-amd64.tar.gz" {
		t.Fatalf("ArtifactName = %q, want linux-amd64 release asset", plan.ArtifactName)
	}
	if plan.SnapshotPath != "/home/xel/.gormes/snapshots/20260515-201545-pre-update" {
		t.Fatalf("SnapshotPath = %q, want planned pre-update snapshot path", plan.SnapshotPath)
	}
	for _, want := range []UpdateReleaseComponent{
		UpdateReleaseComponentSnapshot,
		UpdateReleaseComponentBinary,
		UpdateReleaseComponentChecksum,
	} {
		if !updateReleasePlanHasComponent(plan, want) {
			t.Fatalf("components = %+v, missing %s", plan.Components, want)
		}
	}
	if len(plan.Blockers) != 0 {
		t.Fatalf("Blockers = %+v, want none", plan.Blockers)
	}

	current := BuildUpdateReleasePlan(UpdateReleasePlanOptions{
		InstallKind: UpdateInstallKindRelease,
		Channel:     UpdateReleaseChannelStable,
		Current:     UpdateBuildIdentity{Version: "0.2.12", GitCommit: "oldsha"},
		Target:      UpdateReleaseMetadata{Version: "0.2.12", Tag: "v0.2.12", GitCommit: "main"},
		GOOS:        "linux",
		GOARCH:      "amd64",
	})
	if current.UpdateAvailable {
		t.Fatalf("UpdateAvailable = true for same release version with non-SHA target_commitish; plan=%+v", current)
	}
}

func TestUpdateReleasePlannerClassifiesLayoutsAndBlockers(t *testing.T) {
	managed := BuildUpdateReleasePlan(UpdateReleasePlanOptions{
		InstallKind: UpdateInstallKindManagedSource,
		Channel:     UpdateReleaseChannelDevelopment,
		Current:     UpdateBuildIdentity{Version: "0.2.12", GitCommit: "oldsha"},
		Target:      UpdateReleaseMetadata{Version: "0.2.12", GitCommit: "newsha"},
		GOOS:        "linux",
		GOARCH:      "amd64",
	})
	if managed.Source != UpdateSourceManagedSource {
		t.Fatalf("managed source plan source = %q, want managed_source", managed.Source)
	}
	if !updateReleasePlanHasComponent(managed, UpdateReleaseComponentSourceCheckout) {
		t.Fatalf("managed source components = %+v, want source_checkout", managed.Components)
	}

	unknown := BuildUpdateReleasePlan(UpdateReleasePlanOptions{
		InstallKind: UpdateInstallKindUnknown,
		Channel:     UpdateReleaseChannelStable,
		Current:     UpdateBuildIdentity{Version: "0.2.12", GitCommit: "oldsha"},
		GOOS:        "linux",
		GOARCH:      "amd64",
	})
	if unknown.Source != UpdateSourceUnknown || !updateReleasePlanHasBlocker(unknown, UpdateReleaseBlockerUnknownInstallLayout) {
		t.Fatalf("unknown plan = %+v, want unknown source and unknown_install_layout blocker", unknown)
	}

	blocked := BuildUpdateReleasePlan(UpdateReleasePlanOptions{
		InstallKind:   UpdateInstallKindUnmanagedSource,
		Channel:       UpdateReleaseChannelStable,
		Current:       UpdateBuildIdentity{Version: "0.2.12", GitCommit: "oldsha"},
		Target:        UpdateReleaseMetadata{Version: "0.2.13", Tag: "v0.2.13"},
		GOOS:          "plan9",
		GOARCH:        "mips",
		MetadataError: errors.New("github API unavailable"),
		DirtySource:   true,
	})
	for _, want := range []UpdateReleaseBlockerKind{
		UpdateReleaseBlockerChannelMismatch,
		UpdateReleaseBlockerDirtyUnmanagedSource,
		UpdateReleaseBlockerUnsupportedPlatform,
		UpdateReleaseBlockerMissingReleaseMetadata,
	} {
		if !updateReleasePlanHasBlocker(blocked, want) {
			t.Fatalf("blocked plan = %+v, missing blocker %s", blocked, want)
		}
	}
}

func updateReleasePlanHasComponent(plan UpdateReleasePlan, want UpdateReleaseComponent) bool {
	for _, got := range plan.Components {
		if got == want {
			return true
		}
	}
	return false
}

func updateReleasePlanHasBlocker(plan UpdateReleasePlan, want UpdateReleaseBlockerKind) bool {
	for _, got := range plan.Blockers {
		if got.Kind == want {
			return true
		}
	}
	return false
}
