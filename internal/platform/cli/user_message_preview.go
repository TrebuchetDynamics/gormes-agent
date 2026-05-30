package cli

import "github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/input"

type UserMessagePreviewConfig = input.UserMessagePreviewConfig

func FormatUserMessagePreview(userInput string, cfg UserMessagePreviewConfig) string {
	return input.FormatUserMessagePreview(userInput, cfg)
}
