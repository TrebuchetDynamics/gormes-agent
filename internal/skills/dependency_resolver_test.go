package skills

import (
	"strings"
	"testing"
)

func TestDependencyGraph_Linear(t *testing.T) {
	dg := NewDependencyGraph()
	dg.AddSkill("A", nil)
	dg.AddSkill("B", []string{"A"})
	dg.AddSkill("C", []string{"B"})

	order, err := dg.Resolve()
	if err != nil {
		t.Fatal(err)
	}
	if len(order) != 3 {
		t.Fatalf("order length = %d, want 3", len(order))
	}
}

func TestDependencyGraph_Circular(t *testing.T) {
	dg := NewDependencyGraph()
	dg.AddSkill("A", []string{"B"})
	dg.AddSkill("B", []string{"A"})

	_, err := dg.Resolve()
	if err == nil {
		t.Fatal("circular dependency should error")
	}
	if !strings.Contains(err.Error(), "circular") {
		t.Fatalf("error should mention circular: %v", err)
	}
}

func TestDependencyGraph_Missing(t *testing.T) {
	dg := NewDependencyGraph()
	dg.AddSkill("A", []string{"nonexistent"})

	_, err := dg.Resolve()
	if err == nil {
		t.Fatal("missing dependency should error")
	}
}

func TestSkillComposer_Compose(t *testing.T) {
	sc := NewSkillComposer()
	order, _, err := sc.Compose([]SkillDependency{
		{Name: "base", Dependencies: nil},
		{Name: "mid", Dependencies: []string{"base"}},
		{Name: "top", Dependencies: []string{"mid"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(order) != 3 {
		t.Fatalf("order length = %d, want 3", len(order))
	}
}

func TestSkillComposer_ValidateChain(t *testing.T) {
	sc := NewSkillComposer()
	sc.graph.AddSkill("base", nil)
	sc.graph.AddSkill("mid", []string{"base"})

	if err := sc.ValidateChain([]string{"base", "mid"}); err != nil {
		t.Fatal(err)
	}
	if err := sc.ValidateChain([]string{"mid", "base"}); err == nil {
		t.Fatal("wrong order should fail validation")
	}
}
