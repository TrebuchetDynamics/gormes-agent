package tokens

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
)

// State labels the operator-visible state of an MCP OAuth token slot without
// leaking secret material.
type State = string

const (
	StateAbsent                 State = "absent"
	StateValid                  State = "valid"
	StateExpired                State = "expired"
	StateNoninteractiveRequired State = "noninteractive_required"
)

const (
	evidenceNoToken                = "no_token"
	evidenceOK                     = "ok"
	evidenceTokenExpired           = "token_expired"
	evidenceNoninteractiveRequired = "noninteractive_auth_unavailable"
)

// ErrNoninteractiveRequired is returned when a token cannot be recovered
// without user interaction.
var ErrNoninteractiveRequired = errors.New("mcp oauth: noninteractive auth unavailable")

// Token is the in-memory credential record for a single MCP server.
type Token struct {
	AccessToken  string
	RefreshToken string
	Scope        string
	Issuer       string
	ExpiresAt    time.Time
}

// Status is the redacted, operator-visible read of one server's OAuth state.
type Status struct {
	Server   string
	State    State
	Evidence string
}

func (s Status) String() string {
	parts := []string{
		"server=" + s.Server,
		"state=" + s.State,
	}
	if s.Evidence != "" {
		parts = append(parts, "evidence="+s.Evidence)
	}
	return "mcp_oauth " + strings.Join(parts, " ")
}

// Store is a pure in-memory state store for MCP OAuth tokens.
type Store struct {
	mu             sync.RWMutex
	tokens         map[string]Token
	noninteractive bool
}

// NewStore returns an empty store in interactive mode.
func NewStore() *Store {
	return &Store{tokens: map[string]Token{}}
}

// WithNoninteractive toggles the noninteractive policy and returns the store so
// call sites can chain configuration on construction.
func (s *Store) WithNoninteractive(enabled bool) *Store {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	s.noninteractive = enabled
	s.mu.Unlock()
	return s
}

// Get returns the stored token for server, if any.
func (s *Store) Get(server string) (Token, bool) {
	if s == nil {
		return Token{}, false
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	tok, ok := s.tokens[server]
	return tok, ok
}

// Set stores tok under server.
func (s *Store) Set(server string, tok Token) error {
	if s == nil {
		return fmt.Errorf("mcp oauth: nil store")
	}
	if strings.TrimSpace(server) == "" {
		return fmt.Errorf("mcp oauth: server name required")
	}
	s.mu.Lock()
	s.tokens[server] = tok
	s.mu.Unlock()
	return nil
}

// Clear removes any stored token for server.
func (s *Store) Clear(server string) {
	if s == nil {
		return
	}
	s.mu.Lock()
	delete(s.tokens, server)
	s.mu.Unlock()
}

// StatusFor returns the redacted status of server at the given instant.
func (s *Store) StatusFor(server string, now time.Time) Status {
	status := Status{Server: server, State: StateAbsent, Evidence: evidenceNoToken}
	if s == nil {
		return status
	}
	s.mu.RLock()
	tok, ok := s.tokens[server]
	noninteractive := s.noninteractive
	s.mu.RUnlock()

	if !ok {
		if noninteractive {
			status.State = StateNoninteractiveRequired
			status.Evidence = evidenceNoninteractiveRequired
		}
		return status
	}
	if strings.TrimSpace(tok.AccessToken) == "" {
		if strings.TrimSpace(tok.RefreshToken) == "" {
			if noninteractive {
				status.State = StateNoninteractiveRequired
				status.Evidence = evidenceNoninteractiveRequired
			}
			return status
		}
		status.State = StateExpired
		status.Evidence = evidenceTokenExpired
		return status
	}
	if !tok.ExpiresAt.IsZero() && !now.Before(tok.ExpiresAt) {
		if strings.TrimSpace(tok.RefreshToken) == "" && noninteractive {
			status.State = StateNoninteractiveRequired
			status.Evidence = evidenceNoninteractiveRequired
			return status
		}
		status.State = StateExpired
		status.Evidence = evidenceTokenExpired
		return status
	}
	status.State = StateValid
	status.Evidence = evidenceOK
	return status
}
