package gormescli

import (
	appwebcontent "github.com/TrebuchetDynamics/gormes-agent/internal/app/webcontent"
	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools"
)

func NewWebContentProcessor(client llm.Client, model string) tools.WebContentProcessor {
	return appwebcontent.NewProcessor(client, model)
}
