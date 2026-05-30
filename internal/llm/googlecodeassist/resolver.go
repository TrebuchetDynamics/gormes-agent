package googlecodeassist

import (
	"fmt"
	"net/http"
	"strings"
)

type TokenProvider interface {
	Token() (string, error)
}

type Resolver struct {
	project  string
	tier     string
	provider TokenProvider
}

func NewResolver(project, tier string, provider TokenProvider) *Resolver {
	return &Resolver{
		project:  strings.TrimSpace(project),
		tier:     strings.TrimSpace(tier),
		provider: provider,
	}
}

func (r *Resolver) Headers() (http.Header, error) {
	header := make(http.Header)
	header.Set("Content-Type", "application/json")
	header.Set("User-Agent", "gormes-agent/0.0.0")
	if r.provider != nil {
		token, err := r.provider.Token()
		if err != nil {
			return nil, fmt.Errorf("code_assist_token_unavailable: %w", err)
		}
		if token != "" {
			header.Set("Authorization", "Bearer "+token)
		}
	}
	return header, nil
}

func (r *Resolver) ProjectContext() string {
	if r.project != "" {
		return r.project
	}
	return "-"
}

func (r *Resolver) Tier() string {
	if r.tier != "" {
		return r.tier
	}
	return "free"
}

func (r *Resolver) RequiresExplicitProject() bool {
	return r.Tier() == "paid" && r.project == ""
}
