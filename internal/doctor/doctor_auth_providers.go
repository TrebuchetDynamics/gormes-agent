package doctor

import (
	"fmt"
	"strings"
)

const (
	AuthProviderLoggedIn  = "logged_in"
	AuthProviderLoggedOut = "logged_out"
	AuthProviderError     = "error"
	AuthProviderSkipped   = "skipped"
)

// AuthProviderStatus is the redacted, local auth state for one provider in
// the Hermes-compatible Auth Providers doctor section.
type AuthProviderStatus struct {
	Name            string
	Provider        string
	AuthType        string
	Status          string
	Reason          string
	Authenticated   bool
	CredentialCount int
	Redacted        bool
}

// CheckAuthProviders renders one Hermes-compatible Auth Providers check with
// a per-provider item list. It never runs network probes; callers pass local
// credential-store state that has already been redacted.
func CheckAuthProviders(providers []AuthProviderStatus) CheckResult {
	items := make([]ItemInfo, 0, len(providers))
	overall := StatusPass
	skipped := 0
	for _, provider := range providers {
		item := authProviderItem(provider)
		items = append(items, item)
		switch item.Status {
		case StatusFail, StatusWarn:
			if overall != StatusFail {
				overall = StatusWarn
			}
		case StatusSkip:
			skipped++
		}
	}
	if len(items) == 0 || skipped == len(items) {
		overall = StatusSkip
	}

	summary := "provider auth ready"
	switch overall {
	case StatusWarn:
		summary = "some auth providers are not logged in"
	case StatusFail:
		summary = "provider auth failed"
	case StatusSkip:
		summary = "no auth providers configured"
	}
	return CheckResult{
		Name:    "Auth Providers",
		Status:  overall,
		Summary: summary,
		Items:   items,
	}
}

func authProviderItem(provider AuthProviderStatus) ItemInfo {
	name := strings.TrimSpace(provider.Name)
	if name == "" {
		name = strings.TrimSpace(provider.Provider)
	}
	if name == "" {
		name = "provider"
	}
	status := StatusWarn
	displayStatus := "not logged in"
	switch {
	case provider.Authenticated || provider.Status == AuthProviderLoggedIn:
		status = StatusPass
		displayStatus = "logged in"
	case provider.Status == AuthProviderSkipped:
		status = StatusSkip
		displayStatus = "skipped"
	case provider.Status == AuthProviderError:
		status = StatusWarn
		displayStatus = "not logged in"
	case provider.Status == AuthProviderLoggedOut:
		status = StatusWarn
		displayStatus = "not logged in"
	}
	return ItemInfo{
		Name:   name,
		Status: status,
		Note:   authProviderNote(provider, displayStatus),
	}
}

func authProviderNote(provider AuthProviderStatus, displayStatus string) string {
	parts := []string{
		"provider=" + authProviderField(provider.Provider, "unknown"),
		"status=" + displayStatus,
	}
	if authType := strings.TrimSpace(provider.AuthType); authType != "" {
		parts = append(parts, "auth_type="+authType)
	}
	if provider.CredentialCount > 0 {
		parts = append(parts, fmt.Sprintf("credentials=%d", provider.CredentialCount))
	}
	if reason := sanitizeAuthProviderReason(provider.Reason); reason != "" {
		parts = append(parts, "reason="+reason)
	}
	parts = append(parts, fmt.Sprintf("redacted=%t", provider.Redacted))
	return strings.Join(parts, " ")
}

func authProviderField(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}

func sanitizeAuthProviderReason(reason string) string {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return ""
	}
	lower := strings.ToLower(reason)
	for _, marker := range []string{
		"access_token",
		"refresh_token",
		"authorization",
		"bearer ",
		"api_key",
		"apikey",
		"jwt",
		"token=",
		"_token",
	} {
		if strings.Contains(lower, marker) {
			return "[redacted]"
		}
	}
	if len(reason) > 160 {
		return reason[:157] + "..."
	}
	return strings.Join(strings.Fields(reason), " ")
}
