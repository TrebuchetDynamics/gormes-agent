package skills

import "github.com/TrebuchetDynamics/gormes-agent/internal/extensibility/skills/composition"

type SkillDependency = composition.SkillDependency

type DependencyGraph = composition.DependencyGraph

func NewDependencyGraph() *DependencyGraph {
	return composition.NewDependencyGraph()
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
	return composition.ValidateChain(sc.graph, chain)
}
