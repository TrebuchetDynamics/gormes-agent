package discord

import "testing"

func TestDiscordInteractionAuthNoPoliciesAllowsEveryone(t *testing.T) {
	got := EvaluateInteractionAuthorization(DiscordInteractionContext{
		UserID:    "user-1",
		ChannelID: "chan-1",
	}, DiscordInteractionPolicy{})
	if !got.Allowed {
		t.Fatalf("Allowed = false, reason %q", got.Reason)
	}
}

func TestDiscordInteractionAuthUserAllowlist(t *testing.T) {
	policy := DiscordInteractionPolicy{AllowedUserIDs: []string{"user-1"}}
	if got := EvaluateInteractionAuthorization(DiscordInteractionContext{UserID: "user-1"}, policy); !got.Allowed {
		t.Fatalf("allowed user rejected: %+v", got)
	}
	got := EvaluateInteractionAuthorization(DiscordInteractionContext{UserID: "user-2"}, policy)
	if got.Allowed || got.Code != DiscordInteractionUserDenied {
		t.Fatalf("disallowed user = %+v, want user denied", got)
	}
}

func TestDiscordInteractionAuthRoleAllowlistORsWithUsers(t *testing.T) {
	policy := DiscordInteractionPolicy{
		AllowedUserIDs: []string{"user-1"},
		AllowedRoleIDs: []string{"role-ops"},
	}
	if got := EvaluateInteractionAuthorization(DiscordInteractionContext{UserID: "user-2", RoleIDs: []string{"role-ops"}}, policy); !got.Allowed {
		t.Fatalf("role user rejected: %+v", got)
	}
	if got := EvaluateInteractionAuthorization(DiscordInteractionContext{UserID: "user-1", RoleIDs: []string{"role-other"}}, policy); !got.Allowed {
		t.Fatalf("user match rejected: %+v", got)
	}
	got := EvaluateInteractionAuthorization(DiscordInteractionContext{UserID: "user-2", RoleIDs: []string{"role-other"}}, policy)
	if got.Allowed || got.Code != DiscordInteractionUserDenied {
		t.Fatalf("neither user nor role match = %+v, want user denied", got)
	}
}

func TestDiscordInteractionAuthFailClosedWhenPolicyDataMissing(t *testing.T) {
	userPolicy := DiscordInteractionPolicy{AllowedUserIDs: []string{"user-1"}}
	got := EvaluateInteractionAuthorization(DiscordInteractionContext{}, userPolicy)
	if got.Allowed || got.Code != DiscordInteractionMissingUser {
		t.Fatalf("missing user = %+v, want fail-closed missing user", got)
	}

	rolePolicy := DiscordInteractionPolicy{AllowedRoleIDs: []string{"role-ops"}}
	got = EvaluateInteractionAuthorization(DiscordInteractionContext{UserID: "user-1"}, rolePolicy)
	if got.Allowed || got.Code != DiscordInteractionUserDenied {
		t.Fatalf("missing roles = %+v, want denied under role policy", got)
	}

	channelPolicy := DiscordInteractionPolicy{AllowedChannelIDs: []string{"chan-1"}}
	got = EvaluateInteractionAuthorization(DiscordInteractionContext{UserID: "user-1"}, channelPolicy)
	if got.Allowed || got.Code != DiscordInteractionMissingChannel {
		t.Fatalf("missing channel = %+v, want missing channel", got)
	}
}

func TestDiscordInteractionAuthAllowedIgnoredAndThreadParentChannels(t *testing.T) {
	policy := DiscordInteractionPolicy{
		AllowedChannelIDs: []string{"parent-1"},
		IgnoredChannelIDs: []string{"thread-2"},
	}
	if got := EvaluateInteractionAuthorization(DiscordInteractionContext{
		UserID:          "user-1",
		ChannelID:       "thread-1",
		ParentChannelID: "parent-1",
	}, policy); !got.Allowed {
		t.Fatalf("thread parent allow rejected: %+v", got)
	}
	got := EvaluateInteractionAuthorization(DiscordInteractionContext{
		UserID:          "user-1",
		ChannelID:       "thread-2",
		ParentChannelID: "parent-1",
	}, policy)
	if got.Allowed || got.Code != DiscordInteractionChannelIgnored {
		t.Fatalf("ignored thread = %+v, want ignored to win", got)
	}
}

func TestDiscordInteractionAuthWildcardChannelPolicy(t *testing.T) {
	allowedAll := DiscordInteractionPolicy{AllowedChannelIDs: []string{"*"}}
	if got := EvaluateInteractionAuthorization(DiscordInteractionContext{UserID: "user-1", ChannelID: "chan-x"}, allowedAll); !got.Allowed {
		t.Fatalf("allowed wildcard rejected: %+v", got)
	}
	ignoredAll := DiscordInteractionPolicy{AllowedChannelIDs: []string{"*"}, IgnoredChannelIDs: []string{"*"}}
	got := EvaluateInteractionAuthorization(DiscordInteractionContext{UserID: "user-1", ChannelID: "chan-x"}, ignoredAll)
	if got.Allowed || got.Code != DiscordInteractionChannelIgnored {
		t.Fatalf("ignored wildcard = %+v, want ignored", got)
	}
}

func TestDiscordComponentAuthUserOrRoleAndFailClosed(t *testing.T) {
	if !AuthorizeComponent(DiscordInteractionContext{UserID: "user-1"}, []string{"user-1"}, nil) {
		t.Fatal("user allowlist match rejected")
	}
	if !AuthorizeComponent(DiscordInteractionContext{UserID: "user-2", RoleIDs: []string{"role-ops"}}, nil, []string{"role-ops"}) {
		t.Fatal("role allowlist match rejected")
	}
	if AuthorizeComponent(DiscordInteractionContext{UserID: "user-2"}, nil, []string{"role-ops"}) {
		t.Fatal("role policy with no role data allowed")
	}
	if AuthorizeComponent(DiscordInteractionContext{}, []string{"user-1"}, nil) {
		t.Fatal("missing user with allowlist allowed")
	}
	if !AuthorizeComponent(DiscordInteractionContext{UserID: "anyone"}, nil, nil) {
		t.Fatal("empty allowlists should allow everyone")
	}
}

func TestDiscordSkillAutocompleteHidesCatalogWhenUnauthorized(t *testing.T) {
	policy := DiscordInteractionPolicy{AllowedUserIDs: []string{"operator"}}
	names := []string{"deploy-prod", "doctor", "debug-share"}

	got := AuthorizedSkillAutocomplete(DiscordInteractionContext{UserID: "intruder"}, policy, names, "d")
	if len(got) != 0 {
		t.Fatalf("unauthorized autocomplete leaked names: %v", got)
	}
	got = AuthorizedSkillAutocomplete(DiscordInteractionContext{UserID: "operator"}, policy, names, "d")
	if len(got) != 3 {
		t.Fatalf("authorized autocomplete = %v, want matching catalog entries", got)
	}
}
