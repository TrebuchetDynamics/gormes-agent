package llm

import "github.com/TrebuchetDynamics/gormes-agent/internal/llm/routing"

type ModelIdentity = routing.ModelIdentity
type DirectAlias = routing.DirectAlias
type ModelSwitchResult = routing.ModelSwitchResult

var ModelAliases = routing.ModelAliases

func ParseModelFlags(rawArgs string) (modelInput string, explicitProvider string, isGlobal bool) {
	return routing.ParseModelFlags(rawArgs)
}

func ModelSortKey(modelID, prefix string) (int, string, string) {
	return routing.ModelSortKey(modelID, prefix)
}

func SortedModelAliases() []string {
	return routing.SortedModelAliases()
}
