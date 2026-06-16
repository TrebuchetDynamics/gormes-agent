package toolpreview

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
)

func SoulText(call llm.ToolCall) string {
	name := strings.TrimSpace(call.Name)
	if name == "" {
		name = "unknown"
	}
	if preview := Preview(name, call.Arguments); preview != "" {
		return "tool: " + name + ": " + preview
	}
	return "tool: " + name
}

func Preview(name string, raw json.RawMessage) string {
	var args map[string]any
	if len(raw) == 0 || json.Unmarshal(raw, &args) != nil {
		return ""
	}
	if len(args) == 0 {
		return ""
	}
	if name == "process" {
		var parts []string
		if action := scalar(args["action"]); action != "" {
			parts = append(parts, action)
		}
		if sessionID := scalar(args["session_id"]); sessionID != "" {
			parts = append(parts, TruncateToken(sessionID, 16))
		}
		if data := scalar(args["data"]); data != "" {
			parts = append(parts, `"`+TruncateToken(data, 20)+`"`)
		}
		return strings.Join(parts, " ")
	}
	if name == "todo" {
		if todos, ok := args["todos"].([]any); ok {
			if merge, _ := args["merge"].(bool); merge {
				return fmt.Sprintf("updating %d task(s)", len(todos))
			}
			return fmt.Sprintf("planning %d task(s)", len(todos))
		}
		return "reading task list"
	}
	key := PrimaryArg(name)
	if key == "" {
		for _, fallback := range []string{"query", "text", "command", "path", "name", "prompt", "code", "goal", "url"} {
			if _, ok := args[fallback]; ok {
				key = fallback
				break
			}
		}
	}
	if key == "" {
		return ""
	}
	return scalar(args[key])
}

func PrimaryArg(name string) string {
	switch name {
	case "terminal":
		return "command"
	case "execute_code":
		return "code"
	case "web_search":
		return "query"
	case "web_extract":
		return "urls"
	case "web_crawl", "browser_navigate":
		return "url"
	case "read_file", "write_file", "patch":
		return "path"
	case "search_files":
		return "pattern"
	case "browser_click", "browser_type", "browser_scroll", "browser_back", "browser_press", "browser_console", "browser_get_images", "browser_vision", "browser_cdp", "browser_dialog":
		switch name {
		case "browser_click":
			return "ref"
		case "browser_type":
			return "text"
		case "browser_scroll":
			return "direction"
		case "browser_press":
			return "key"
		case "browser_cdp":
			return "method"
		case "browser_dialog":
			return "action"
		default:
			return ""
		}
	case "image_generate":
		return "prompt"
	case "text_to_speech":
		return "text"
	case "vision_analyze":
		return "question"
	case "mixture_of_agents":
		return "user_prompt"
	case "skill_view", "skill_manage":
		return "name"
	case "skills_list":
		return "category"
	case "cronjob":
		return "action"
	case "delegate_task":
		return "goal"
	case "clarify":
		return "question"
	default:
		return ""
	}
}

func scalar(value any) string {
	switch v := value.(type) {
	case nil:
		return ""
	case string:
		return strings.Join(strings.Fields(v), " ")
	case []any:
		if len(v) == 0 {
			return ""
		}
		return scalar(v[0])
	case fmt.Stringer:
		return strings.Join(strings.Fields(v.String()), " ")
	default:
		return strings.Join(strings.Fields(fmt.Sprint(v)), " ")
	}
}

func TruncateToken(s string, n int) string {
	runes := []rune(s)
	if n <= 0 || len(runes) <= n {
		return s
	}
	return string(runes[:n])
}
