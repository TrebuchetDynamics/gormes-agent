package setup

import jsoncparser "github.com/TrebuchetDynamics/gormes-agent/internal/tui/terminal/setup/jsonc"

func StripJSONComments(input string) string {
	return jsoncparser.StripJSONComments(input)
}

func parseKeybindings(body []byte) ([]map[string]any, error) {
	return jsoncparser.ParseKeybindings(body)
}
