package cron

import (
	"reflect"
	"testing"
)

func TestCuratorCronSkillRewriteAndRestore(t *testing.T) {
	store, cleanup := newTestStore(t)
	defer cleanup()

	jobA := NewJob("daily", "@daily", "report")
	jobA.Skills = []string{"old-skill", "keep", "dead-skill"}
	if err := store.Create(jobA); err != nil {
		t.Fatalf("Create jobA: %v", err)
	}
	jobB := NewJob("weekly", "@weekly", "report")
	jobB.Skills = []string{"keep"}
	if err := store.Create(jobB); err != nil {
		t.Fatalf("Create jobB: %v", err)
	}

	snapshot, err := store.SnapshotSkillRefs()
	if err != nil {
		t.Fatalf("SnapshotSkillRefs: %v", err)
	}
	report, err := store.RewriteSkillRefs(map[string]string{"old-skill": "umbrella"}, []string{"dead-skill"})
	if err != nil {
		t.Fatalf("RewriteSkillRefs: %v", err)
	}
	if report.UpdatedJobs != 1 || report.Replaced != 1 || report.Removed != 1 {
		t.Fatalf("rewrite report = %+v, want one job, one replace, one remove", report)
	}
	gotA, err := store.Get(jobA.ID)
	if err != nil {
		t.Fatalf("Get jobA: %v", err)
	}
	if !reflect.DeepEqual(gotA.Skills, []string{"umbrella", "keep"}) {
		t.Fatalf("jobA skills after rewrite = %v, want [umbrella keep]", gotA.Skills)
	}

	restored, err := store.RestoreSkillRefs(snapshot)
	if err != nil {
		t.Fatalf("RestoreSkillRefs: %v", err)
	}
	if restored != 1 {
		t.Fatalf("restored = %d, want 1 changed job", restored)
	}
	gotA, err = store.Get(jobA.ID)
	if err != nil {
		t.Fatalf("Get jobA restored: %v", err)
	}
	if !reflect.DeepEqual(gotA.Skills, jobA.Skills) {
		t.Fatalf("jobA skills after restore = %v, want %v", gotA.Skills, jobA.Skills)
	}
	gotB, err := store.Get(jobB.ID)
	if err != nil {
		t.Fatalf("Get jobB restored: %v", err)
	}
	if !reflect.DeepEqual(gotB.Skills, jobB.Skills) {
		t.Fatalf("jobB skills after restore = %v, want %v", gotB.Skills, jobB.Skills)
	}
}
