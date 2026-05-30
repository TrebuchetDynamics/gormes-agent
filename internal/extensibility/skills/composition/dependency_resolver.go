package composition

import (
	"fmt"
	"sort"
)

type SkillDependency struct {
	Name         string
	Dependencies []string
}

type DependencyGraph struct {
	skills map[string]*SkillDependency
}

func NewDependencyGraph() *DependencyGraph {
	return &DependencyGraph{skills: make(map[string]*SkillDependency)}
}

func (dg *DependencyGraph) AddSkill(name string, deps []string) {
	dg.skills[name] = &SkillDependency{
		Name:         name,
		Dependencies: deps,
	}
}

func (dg *DependencyGraph) Resolve() ([]string, error) {
	visited := make(map[string]bool)
	resolved := make(map[string]bool)
	var order []string

	var visit func(name string) error
	visit = func(name string) error {
		if visited[name] {
			if !resolved[name] {
				return fmt.Errorf("circular dependency detected at %q", name)
			}
			return nil
		}
		visited[name] = true

		skill, ok := dg.skills[name]
		if !ok {
			return fmt.Errorf("missing dependency: %q", name)
		}

		for _, dep := range skill.Dependencies {
			if err := visit(dep); err != nil {
				return err
			}
		}

		resolved[name] = true
		order = append(order, name)
		return nil
	}

	names := make([]string, 0, len(dg.skills))
	for n := range dg.skills {
		names = append(names, n)
	}
	sort.Strings(names)

	for _, name := range names {
		if !resolved[name] {
			if err := visit(name); err != nil {
				return nil, err
			}
		}
	}

	return order, nil
}

type SkillComposer struct {
	graph *DependencyGraph
}

func NewSkillComposer() *SkillComposer {
	return &SkillComposer{graph: NewDependencyGraph()}
}

func (sc *SkillComposer) Compose(skills []SkillDependency) ([]string, []string, error) {
	for _, s := range skills {
		sc.graph.AddSkill(s.Name, s.Dependencies)
	}
	order, err := sc.graph.Resolve()
	if err != nil {
		return nil, nil, err
	}
	var cycleFree []string
	for _, s := range order {
		cycleFree = append(cycleFree, s)
	}
	return order, cycleFree, nil
}

func (sc *SkillComposer) ValidateChain(chain []string) error {
	return ValidateChain(sc.graph, chain)
}

func ValidateChain(graph *DependencyGraph, chain []string) error {
	for i, name := range chain {
		skill, ok := graph.skills[name]
		if !ok {
			return fmt.Errorf("skill %q not found in graph", name)
		}
		for _, dep := range skill.Dependencies {
			found := false
			for j := 0; j < i; j++ {
				if chain[j] == dep {
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("skill %q depends on %q which is not before it in chain", name, dep)
			}
		}
	}
	return nil
}
