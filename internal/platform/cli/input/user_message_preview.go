package input

import "github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/input/preview"

type UserMessagePreviewConfig = preview.UserMessagePreviewConfig

func FormatUserMessagePreview(userInput string, cfg UserMessagePreviewConfig) string {
	return preview.FormatUserMessagePreview(userInput, cfg)
}
