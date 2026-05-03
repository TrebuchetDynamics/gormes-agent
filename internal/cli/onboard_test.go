package cli

import (
	"strings"
	"testing"
)

func TestOnboardPlanWalksFirstRunWizardStepsWithSkipWarnings(t *testing.T) {
	plan := BuildOnboardPlan(OnboardPlanInput{})

	wantIDs := []string{
		OnboardStepModel,
		OnboardStepProvider,
		OnboardStepAuth,
		OnboardStepGateway,
		OnboardStepBrowser,
		OnboardStepSkills,
		OnboardStepDashboard,
	}
	if len(plan.Steps) != len(wantIDs) {
		t.Fatalf("len(plan.Steps) = %d, want %d: %#v", len(plan.Steps), len(wantIDs), plan.Steps)
	}
	for i, want := range wantIDs {
		if plan.Steps[i].ID != want {
			t.Fatalf("step %d ID = %q, want %q", i, plan.Steps[i].ID, want)
		}
		if plan.Steps[i].SkipWarning == "" {
			t.Fatalf("step %q missing skip warning: %#v", want, plan.Steps[i])
		}
	}

	for _, want := range []string{
		"gormes setup model",
		"gormes setup provider",
		"gormes auth add",
		"gormes setup gateway",
		"gormes doctor --offline",
		"gormes skills list",
		"gormes dashboard",
	} {
		if !planContains(plan, want) {
			t.Fatalf("plan missing command/guidance %q: %#v", want, plan.Steps)
		}
	}
}

func TestOnboardPlanPrefillsConfiguredRuntime(t *testing.T) {
	plan := BuildOnboardPlan(OnboardPlanInput{
		Provider:       "groq",
		Endpoint:       "https://api.groq.com/openai/v1",
		Model:          "llama-3.3-70b-versatile",
		APIKeyPresent:  true,
		GatewayTargets: []string{"telegram"},
		BrowserCDPURL:  "http://127.0.0.1:9222",
		LocalSkills:    2,
		BundledSkills:  3,
	})

	for _, tc := range []struct {
		id     string
		status string
		want   []string
	}{
		{id: OnboardStepModel, status: OnboardStatusConfigured, want: []string{"llama-3.3-70b-versatile", "pre-filled"}},
		{id: OnboardStepProvider, status: OnboardStatusConfigured, want: []string{"groq", "https://api.groq.com/openai/v1", "pre-filled"}},
		{id: OnboardStepAuth, status: OnboardStatusConfigured, want: []string{"credential", "present"}},
		{id: OnboardStepGateway, status: OnboardStatusConfigured, want: []string{"telegram"}},
		{id: OnboardStepBrowser, status: OnboardStatusConfigured, want: []string{"http://127.0.0.1:9222"}},
		{id: OnboardStepSkills, status: OnboardStatusAvailable, want: []string{"2 local", "3 bundled"}},
		{id: OnboardStepDashboard, status: OnboardStatusAvailable, want: []string{"gormes dashboard"}},
	} {
		step, ok := plan.Step(tc.id)
		if !ok {
			t.Fatalf("plan missing step %q: %#v", tc.id, plan.Steps)
		}
		if step.Status != tc.status {
			t.Fatalf("step %q status = %q, want %q: %#v", tc.id, step.Status, tc.status, step)
		}
		haystack := step.Title + "\n" + step.Detail + "\n" + step.NextCommand + "\n" + step.SkipWarning
		for _, want := range tc.want {
			if !strings.Contains(haystack, want) {
				t.Fatalf("step %q missing %q:\n%#v", tc.id, want, step)
			}
		}
	}
}

func planContains(plan OnboardPlan, needle string) bool {
	for _, step := range plan.Steps {
		haystack := step.Title + "\n" + step.Detail + "\n" + step.NextCommand + "\n" + step.SkipWarning
		if strings.Contains(haystack, needle) {
			return true
		}
	}
	return false
}
