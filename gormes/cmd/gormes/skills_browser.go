package main

import (
	"context"

	"github.com/TrebuchetDynamics/gormes-agent/gormes/internal/skills"
)

// hubSkillsBrowser implements gateway.SkillsBrowser by reading both the local
// active skill store and the hub lock file through skills.Hub. The gateway
// handler already bounds page/perPage, so this implementation only needs to
// defer to the hub and render the shared summary text.
type hubSkillsBrowser struct {
	hub *skills.Hub
}

func newHubSkillsBrowser(hub *skills.Hub) *hubSkillsBrowser {
	return &hubSkillsBrowser{hub: hub}
}

func (b *hubSkillsBrowser) Browse(ctx context.Context, page, perPage int) (skills.BrowseView, string, error) {
	if b == nil || b.hub == nil {
		view := skills.BuildBrowseView(nil, nil, page, perPage)
		return view, skills.FormatBrowseSummary(view), nil
	}
	if err := ctx.Err(); err != nil {
		return skills.BrowseView{}, "", err
	}

	installed, err := b.hub.ListInstalled()
	if err != nil {
		return skills.BrowseView{}, "", err
	}
	available, err := b.hub.ListHub()
	if err != nil {
		return skills.BrowseView{}, "", err
	}

	view := skills.BuildBrowseView(installed, available, page, perPage)
	return view, skills.FormatBrowseSummary(view), nil
}
