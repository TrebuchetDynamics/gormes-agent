package persisted

import "testing"

func TestSpecAppliesDefaultFileMetadataWhilePreservingRoot(t *testing.T) {
	spec := Spec{Name: "channel_directory.json", TmpPattern: ".channel_directory-*.tmp", Label: "channel directory"}

	file := spec.Apply(File{Root: NewRoot(" /tmp/gormes ")})

	if file.Root.String() != "/tmp/gormes" {
		t.Fatalf("Root = %q, want trimmed root", file.Root.String())
	}
	if file.Name != "channel_directory.json" || file.TmpPattern != ".channel_directory-*.tmp" || file.Label != "channel directory" {
		t.Fatalf("file metadata = %+v, want spec metadata", file)
	}
}

func TestSpecApplyKeepsConfiguredFile(t *testing.T) {
	spec := Spec{Name: "default.json", TmpPattern: ".default-*.tmp", Label: "default"}
	configured := File{Root: NewRoot("/tmp/custom"), Name: "custom.json", TmpPattern: ".custom-*.tmp", Label: "custom"}

	if got := spec.Apply(configured); got != configured {
		t.Fatalf("Apply configured = %+v, want %+v", got, configured)
	}
}
