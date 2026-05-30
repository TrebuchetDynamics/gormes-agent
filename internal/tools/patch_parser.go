package tools

import "github.com/TrebuchetDynamics/gormes-agent/internal/tools/patchparser"

type PatchParserTool = patchparser.PatchParserTool

type patchFileInfo = patchparser.PatchFileInfo

func NewPatchParserTool() *PatchParserTool { return patchparser.NewPatchParserTool() }
