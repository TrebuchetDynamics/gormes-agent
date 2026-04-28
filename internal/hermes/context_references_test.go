package hermes

import (
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/transcript"
)

func TestContextReferences_ParseTypedReferencesIgnoresEmailsAndHandles(t *testing.T) {
	message := "email user@example.com and ping @teammate but include @file:src/main.py:1-2 plus @diff and @git:2 and @url:https://example.com/docs"

	refs := ParseContextReferences(message)

	if got, want := referenceKinds(refs), []string{"file", "diff", "git", "url"}; !equalStrings(got, want) {
		t.Fatalf("kinds = %#v, want %#v", got, want)
	}
	if refs[0].Target != "src/main.py" || refs[0].LineStart != 1 || refs[0].LineEnd != 2 {
		t.Fatalf("file reference = %#v", refs[0])
	}
	if refs[2].Target != "2" {
		t.Fatalf("git reference target = %q, want 2", refs[2].Target)
	}
}

func TestContextReferences_ParseIgnoresPathPrefixedAtTokens(t *testing.T) {
	message := "ignore /@file:secret.txt and docs/@folder:src but keep @file:README.md"

	refs := ParseContextReferences(message)

	if got, want := referenceKinds(refs), []string{"file"}; !equalStrings(got, want) {
		t.Fatalf("kinds = %#v, want %#v", got, want)
	}
	if refs[0].Target != "README.md" {
		t.Fatalf("target = %q, want README.md", refs[0].Target)
	}
}

func TestContextReferences_ParseStripsTrailingPunctuation(t *testing.T) {
	refs := ParseContextReferences("review @file:README.md, then see (@url:https://example.com/docs).")

	if got, want := referenceKinds(refs), []string{"file", "url"}; !equalStrings(got, want) {
		t.Fatalf("kinds = %#v, want %#v", got, want)
	}
	if refs[0].Raw != "@file:README.md," {
		t.Fatalf("raw preserves matched token for removal/audit = %q", refs[0].Raw)
	}
	if refs[0].Target != "README.md" {
		t.Fatalf("file target = %q", refs[0].Target)
	}
	if refs[1].Target != "https://example.com/docs" {
		t.Fatalf("url target = %q", refs[1].Target)
	}
}

func TestContextReferences_ParseQuotedReferencesWithSpacesAndRanges(t *testing.T) {
	refs := ParseContextReferences(`review @file:"C:\Users\Simba\My Project\main.py":7-9 and @folder:"docs and specs" plus @file:src/main.py:1-2`)

	if got, want := referenceKinds(refs), []string{"file", "folder", "file"}; !equalStrings(got, want) {
		t.Fatalf("kinds = %#v, want %#v", got, want)
	}
	if refs[0].Target != `C:\Users\Simba\My Project\main.py` || refs[0].LineStart != 7 || refs[0].LineEnd != 9 {
		t.Fatalf("quoted file reference = %#v", refs[0])
	}
	if refs[1].Target != "docs and specs" {
		t.Fatalf("folder target = %q", refs[1].Target)
	}
	if refs[2].Target != "src/main.py" || refs[2].LineStart != 1 || refs[2].LineEnd != 2 {
		t.Fatalf("unquoted ranged file reference = %#v", refs[2])
	}
}

func TestContextReferences_StableHandlesIgnoreMessagePosition(t *testing.T) {
	store := transcript.NewContextReferenceStore()

	first := AttachContextReferenceHandles("review @file:src/main.py:1-2", store)
	second := AttachContextReferenceHandles("later please reread @file:src/main.py:1-2", store)

	if len(first.Handles) != 1 || len(second.Handles) != 1 {
		t.Fatalf("handle counts = %d/%d", len(first.Handles), len(second.Handles))
	}
	if first.Handles[0].ID != second.Handles[0].ID {
		t.Fatalf("stable IDs differ: %s vs %s", first.Handles[0].ID, second.Handles[0].ID)
	}
	if len(store.Snapshot()) != 1 {
		t.Fatalf("store should coalesce equivalent references, got %d records", len(store.Snapshot()))
	}
}

func TestContextReferences_StableHandlesPreserveAttachmentContract(t *testing.T) {
	store := transcript.NewContextReferenceStore()

	result := AttachContextReferenceHandles("use @diff and @staged plus @url:https://example.com/spec", store)

	if result.OriginalMessage != "use @diff and @staged plus @url:https://example.com/spec" {
		t.Fatalf("original message = %q", result.OriginalMessage)
	}
	if got, want := referenceKinds(result.References), []string{"diff", "staged", "url"}; !equalStrings(got, want) {
		t.Fatalf("kinds = %#v, want %#v", got, want)
	}
	if len(result.Handles) != 3 {
		t.Fatalf("handles = %d, want 3", len(result.Handles))
	}
	for _, handle := range result.Handles {
		if handle.ID == "" {
			t.Fatalf("empty handle ID: %#v", handle)
		}
		if handle.Status != transcript.ContextReferenceStatusPending {
			t.Fatalf("handle status = %q, want %q", handle.Status, transcript.ContextReferenceStatusPending)
		}
	}
}

func referenceKinds(refs []ContextReference) []string {
	out := make([]string, 0, len(refs))
	for _, ref := range refs {
		out = append(out, ref.Kind)
	}
	return out
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
