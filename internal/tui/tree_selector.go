package tui

import "github.com/TrebuchetDynamics/gormes-agent/internal/tui/sessiontree"

type SessionTreeRequest = sessiontree.SessionTreeRequest

type SessionTreeResult = sessiontree.SessionTreeResult

type SessionTreeEntry = sessiontree.SessionTreeEntry

type SessionTreeMessage = sessiontree.SessionTreeMessage

func BuildSessionTreePage(tree SessionTreeResult) (TransientPageState, bool) {
	page, ok := sessiontree.BuildPage(tree)
	if !ok {
		return TransientPageState{}, false
	}
	return TransientPageState{Title: page.Title, Body: page.Body}, true
}
