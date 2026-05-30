package tools

import (
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/toolkit"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/xsearch"
)

const XSearchToolName = xsearch.XSearchToolName

type XSearchConfig = xsearch.XSearchConfig
type XSearchAuthStatus = xsearch.XSearchAuthStatus
type XSearchTool = xsearch.XSearchTool
type XSearchResponse = xsearch.XSearchResponse
type XSearchResult = xsearch.XSearchResult
type XSearchMetrics = xsearch.XSearchMetrics

func NewXSearchTools(cfg XSearchConfig) []toolkit.Tool {
	return xsearch.NewXSearchTools(cfg)
}

func RegisterXSearchTools(r *toolkit.Registry, cfg XSearchConfig) {
	xsearch.RegisterXSearchTools(r, cfg)
}
