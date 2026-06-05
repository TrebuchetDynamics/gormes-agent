package session

import (
	"context"
	"reflect"
	"testing"
	"time"
)

func TestSessionTreeBuildsLineageLabelsAndFilterProjections(t *testing.T) {
	items := []Metadata{
		{SessionID: "sess-root", Title: "Root Plan", LineageKind: LineageKindPrimary, CreatedAt: 100, UpdatedAt: 120, Labels: []string{"pinned"}},
		{SessionID: "sess-compress", Title: "Compressed", ParentSessionID: "sess-root", LineageKind: LineageKindCompression, CreatedAt: 130, UpdatedAt: 140},
		{SessionID: "sess-fork", Title: "Branch", ParentSessionID: "sess-root", LineageKind: LineageKindFork, CreatedAt: 125, UpdatedAt: 150, Labels: []string{"review"}},
	}
	ledgers := map[string]SessionLedger{
		"sess-root": {SessionID: "sess-root", Messages: []SessionLedgerMessage{
			{ID: 1, Role: "user", Content: "root user prompt", CreatedAtUnix: 101},
			{ID: 2, Role: "tool", Content: "read_file noise", CreatedAtUnix: 102},
			{ID: 3, Role: "assistant", Content: "root answer", CreatedAtUnix: 103},
		}},
		"sess-fork": {SessionID: "sess-fork", Messages: []SessionLedgerMessage{
			{ID: 4, Role: "user", Content: "fork prompt", CreatedAtUnix: 126},
		}},
	}

	tree := BuildSessionTree(items, ledgers, TreeOptions{ActiveSessionID: "sess-fork", Filter: TreeFilterNoTools})
	if gotIDs := treeEntryIDs(tree.Entries); !reflect.DeepEqual(gotIDs, []string{"sess-root", "sess-compress", "sess-fork"}) {
		t.Fatalf("tree IDs = %v, want root/compression/fork order", gotIDs)
	}
	root := tree.Entries[0]
	if root.Depth != 0 || root.Title != "Root Plan" || !reflect.DeepEqual(root.Labels, []string{"pinned"}) {
		t.Fatalf("root entry = %+v, want title, depth, labels", root)
	}
	if tree.Entries[1].Depth != 1 || tree.Entries[1].LineageKind != LineageKindCompression {
		t.Fatalf("compression child = %+v, want depth=1 lineage=compression", tree.Entries[1])
	}
	if !tree.Entries[2].Active || tree.Entries[2].LineageKind != LineageKindFork || !reflect.DeepEqual(tree.Entries[2].Labels, []string{"review"}) {
		t.Fatalf("fork child = %+v, want active fork with label", tree.Entries[2])
	}
	if gotRoles := treeMessageRoles(root.Messages); !reflect.DeepEqual(gotRoles, []string{"user", "assistant"}) {
		t.Fatalf("no-tools message roles = %v, want user+assistant", gotRoles)
	}

	userOnly := BuildSessionTree(items, ledgers, TreeOptions{Filter: TreeFilterUserOnly})
	if gotRoles := treeMessageRoles(userOnly.Entries[0].Messages); !reflect.DeepEqual(gotRoles, []string{"user"}) {
		t.Fatalf("user-only message roles = %v, want only user", gotRoles)
	}

	labeled := BuildSessionTree(items, ledgers, TreeOptions{Filter: TreeFilterLabeledOnly})
	if gotIDs := treeEntryIDs(labeled.Entries); !reflect.DeepEqual(gotIDs, []string{"sess-root", "sess-fork"}) {
		t.Fatalf("labeled-only IDs = %v, want labeled sessions", gotIDs)
	}

	all := BuildSessionTree(items, ledgers, TreeOptions{Filter: TreeFilterAllEquivalent})
	if gotRoles := treeMessageRoles(all.Entries[0].Messages); !reflect.DeepEqual(gotRoles, []string{"user", "tool", "assistant"}) {
		t.Fatalf("all-equivalent roles = %v, want user+tool+assistant", gotRoles)
	}
}

func TestSessionLabelsSetAndClearThroughMetadataStore(t *testing.T) {
	ctx := context.Background()
	m := NewMemMap()
	if err := m.PutMetadata(ctx, Metadata{SessionID: "sess-label", Title: "Label Me", UpdatedAt: 10}); err != nil {
		t.Fatalf("PutMetadata: %v", err)
	}
	labels, err := m.SetLabels(ctx, "sess-label", []string{"review", "review", " Pinned "}, time.Unix(123, 0))
	if err != nil {
		t.Fatalf("SetLabels: %v", err)
	}
	if !reflect.DeepEqual(labels, []string{"Pinned", "review"}) {
		t.Fatalf("labels = %v, want sorted unique labels", labels)
	}
	meta, ok, err := m.GetMetadata(ctx, "sess-label")
	if err != nil || !ok {
		t.Fatalf("GetMetadata ok=%v err=%v", ok, err)
	}
	if meta.Title != "Label Me" || meta.UpdatedAt != 123 || !reflect.DeepEqual(meta.Labels, []string{"Pinned", "review"}) {
		t.Fatalf("metadata after labels = %+v, want title preserved and labels set", meta)
	}

	labels, err = m.SetLabels(ctx, "sess-label", nil, time.Unix(124, 0))
	if err != nil {
		t.Fatalf("clear labels: %v", err)
	}
	if len(labels) != 0 {
		t.Fatalf("cleared labels = %v, want empty", labels)
	}
	meta, _, _ = m.GetMetadata(ctx, "sess-label")
	if len(meta.Labels) != 0 || meta.Title != "Label Me" {
		t.Fatalf("metadata after clear = %+v, want no labels and title preserved", meta)
	}
}

func TestSessionTreeReplayPromptFromLedger(t *testing.T) {
	ledger := SessionLedger{SessionID: "sess-replay", Messages: []SessionLedgerMessage{
		{ID: 1, Role: "assistant", Content: "not editable"},
		{ID: 2, Role: "user", Content: "please edit me"},
	}}
	text, evidence := ReplayPromptFromLedger(ledger, 2)
	if evidence != "" || text != "please edit me" {
		t.Fatalf("ReplayPromptFromLedger user = text %q evidence %q", text, evidence)
	}
	text, evidence = ReplayPromptFromLedger(ledger, 1)
	if text != "" || evidence != ReplayUnavailableNonUserEntry {
		t.Fatalf("ReplayPromptFromLedger assistant = text %q evidence %q", text, evidence)
	}
	_, evidence = ReplayPromptFromLedger(ledger, 99)
	if evidence != ReplayUnavailableEntryMissing {
		t.Fatalf("missing evidence = %q", evidence)
	}
}

func treeEntryIDs(entries []TreeEntry) []string {
	ids := make([]string, 0, len(entries))
	for _, entry := range entries {
		ids = append(ids, entry.SessionID)
	}
	return ids
}

func treeMessageRoles(messages []TreeMessage) []string {
	roles := make([]string, 0, len(messages))
	for _, msg := range messages {
		roles = append(roles, msg.Role)
	}
	return roles
}
