package tools

import "github.com/TrebuchetDynamics/gormes-agent/internal/tools/vision"

type VisionAnalyzeTool = vision.AnalyzeTool

func NewVisionAnalyzeTool() *VisionAnalyzeTool { return vision.NewAnalyzeTool() }
