package toolsview

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestUsage(t *testing.T) {
	got := Usage("enable")
	for _, want := range []string{"usage: /tools enable <name> [name ...]", "/tools enable web", "/tools enable github:create_issue"} {
		if !strings.Contains(got, want) {
			t.Fatalf("Usage missing %q: %q", want, got)
		}
	}
}

func TestLines(t *testing.T) {
	got := Lines("disable", Result{Changed: []string{"web"}, Unknown: []string{"missing"}, MissingServers: []string{"github"}, Reset: true})
	want := []string{"disabled: web", "unknown toolsets: missing", "missing MCP servers: github", "session reset. new tool configuration is active."}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Lines = %#v, want %#v", got, want)
	}
}

func TestHandleSlash(t *testing.T) {
	if res := HandleSlash("/tools", nil); !res.Fallback {
		t.Fatalf("HandleSlash(/tools) = %+v, want fallback", res)
	}
	if res := HandleSlash("/tools list", nil); !res.Fallback {
		t.Fatalf("HandleSlash(/tools list) = %+v, want fallback", res)
	}
	if res := HandleSlash("/tools enable", nil); res.Status != Usage("enable") || res.Open {
		t.Fatalf("HandleSlash(missing names) = %+v", res)
	}
	if res := HandleSlash("/tools enable web", nil); res.Status != "tools: configuration unavailable" || res.Open {
		t.Fatalf("HandleSlash(no configure) = %+v", res)
	}

	var gotAction string
	var gotNames []string
	res := HandleSlash("/tools disable web github:create_issue", func(action string, names []string) (Result, error) {
		gotAction = action
		gotNames = append([]string(nil), names...)
		return Result{Changed: []string{"web"}}, nil
	})
	if gotAction != "disable" || !reflect.DeepEqual(gotNames, []string{"web", "github:create_issue"}) || !res.Open || res.Title != "Tools" || res.Status != "disabled: web" || res.Body != "disabled: web" {
		t.Fatalf("HandleSlash(success) = %+v action=%q names=%v", res, gotAction, gotNames)
	}
	if res := HandleSlash("/tools enable web", func(string, []string) (Result, error) { return Result{}, errors.New("locked") }); res.Status != "tools: locked" || res.Open {
		t.Fatalf("HandleSlash(error) = %+v", res)
	}
}
