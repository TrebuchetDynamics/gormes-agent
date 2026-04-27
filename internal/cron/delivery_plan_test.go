package cron

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestCronDeliveryPlan_ParseTargets(t *testing.T) {
	origin := &DeliveryOrigin{
		Platform: "Telegram",
		ChatID:   "-100777",
		ThreadID: "99",
	}
	directory := staticDeliveryDirectory{
		targets: map[string]DeliveryTarget{
			"discord": {Platform: "discord", ChatID: "home-channel", ThreadID: "thread-7"},
		},
	}

	tests := []struct {
		name string
		opts DeliveryPlanOptions
		want []string
	}{
		{
			name: "omitted deliver defaults local",
			opts: DeliveryPlanOptions{},
			want: []string{"local"},
		},
		{
			name: "origin target",
			opts: DeliveryPlanOptions{Deliver: "origin", Origin: origin},
			want: []string{"telegram:-100777:99"},
		},
		{
			name: "local target",
			opts: DeliveryPlanOptions{Deliver: " local "},
			want: []string{"local"},
		},
		{
			name: "explicit telegram thread",
			opts: DeliveryPlanOptions{Deliver: "telegram:-100123:42"},
			want: []string{"telegram:-100123:42"},
		},
		{
			name: "comma separated targets",
			opts: DeliveryPlanOptions{
				Deliver:   "telegram:-100123:42, local, discord",
				Directory: directory,
			},
			want: []string{"telegram:-100123:42", "local", "discord:home-channel:thread-7"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plan := PlanCronDelivery(tt.opts)
			if got := normalizedDeliveryTargets(plan.Targets); !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("targets = %#v, want %#v; evidence=%#v", got, tt.want, plan.Evidence)
			}
			if len(plan.Evidence) != 0 {
				t.Fatalf("evidence = %#v, want none", plan.Evidence)
			}
		})
	}
}

func TestCronDeliveryPlan_InvalidTargetsReturnEvidence(t *testing.T) {
	for _, raw := range []string{"telegram:", ":42", "telegram::42", "telegram:chat:", "telegram:chat:thread:extra"} {
		t.Run(raw, func(t *testing.T) {
			plan := PlanCronDelivery(DeliveryPlanOptions{Deliver: raw})
			if len(plan.Targets) != 0 {
				t.Fatalf("targets = %#v, want none", plan.Targets)
			}
			assertDeliveryEvidence(t, plan.Evidence, DeliveryEvidenceTargetParseFailed)
		})
	}

	plan := PlanCronDelivery(DeliveryPlanOptions{Deliver: "discord"})
	if len(plan.Targets) != 0 {
		t.Fatalf("targets = %#v, want none without channel directory", plan.Targets)
	}
	assertDeliveryEvidence(t, plan.Evidence, DeliveryEvidenceChannelDirectoryMissing)
}

func TestCronDeliveryPlan_MediaTags(t *testing.T) {
	content := PrepareCronDeliveryContent("Report ready [MEDIA:outputs/chart.png]\nUnsafe [MEDIA:../../secret.txt]\nStill text.")

	if got, want := content.Text, "Report ready\nUnsafe [MEDIA:redacted]\nStill text."; got != want {
		t.Fatalf("cleaned text = %q, want %q", got, want)
	}
	if got, want := mediaPaths(content.Media), []string{"outputs/chart.png"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("media paths = %#v, want %#v", got, want)
	}
	if strings.Contains(content.Text, "secret") || strings.Contains(content.Text, "../") {
		t.Fatalf("cleaned text leaked traversal path: %q", content.Text)
	}
	if strings.Contains(formatDeliveryEvidence(content.Evidence), "secret") ||
		strings.Contains(formatDeliveryEvidence(content.Evidence), "../") {
		t.Fatalf("evidence leaked traversal path: %#v", content.Evidence)
	}
	assertDeliveryEvidence(t, content.Evidence, DeliveryEvidenceMediaIgnored)
}

func TestCronDeliveryPlan_LiveAdapterFallback(t *testing.T) {
	plan := PlanCronDelivery(DeliveryPlanOptions{Deliver: "telegram:-100123:42"})
	content := PrepareCronDeliveryContent("final response")

	t.Run("live success bypasses fallback", func(t *testing.T) {
		live := &fakeCronLiveAdapter{}
		fallback := &fakeCronDeliverySink{}

		outcome := DeliverCronDeliveryPlan(context.Background(), plan, content, live, fallback)

		if !outcome.Delivered || outcome.Err != nil {
			t.Fatalf("outcome = %+v, want delivered without error", outcome)
		}
		if len(live.calls) != 1 {
			t.Fatalf("live calls = %d, want 1", len(live.calls))
		}
		if len(fallback.deliveries) != 0 {
			t.Fatalf("fallback deliveries = %#v, want none", fallback.deliveries)
		}
		assertNoDeliveryEvidence(t, outcome.Evidence, DeliveryEvidenceFallbackSinkUsed)
	})

	t.Run("live unavailable uses fallback", func(t *testing.T) {
		live := &fakeCronLiveAdapter{err: ErrLiveAdapterUnavailable}
		fallback := &fakeCronDeliverySink{}

		outcome := DeliverCronDeliveryPlan(context.Background(), plan, content, live, fallback)

		if !outcome.Delivered || outcome.Err != nil {
			t.Fatalf("outcome = %+v, want delivered by fallback without terminal error", outcome)
		}
		if got, want := fallback.deliveries, []string{"final response"}; !reflect.DeepEqual(got, want) {
			t.Fatalf("fallback deliveries = %#v, want %#v", got, want)
		}
		assertDeliveryEvidence(t, outcome.Evidence, DeliveryEvidenceLiveAdapterUnavailable)
		assertDeliveryEvidence(t, outcome.Evidence, DeliveryEvidenceFallbackSinkUsed)
	})

	t.Run("live and fallback failures preserve run status", func(t *testing.T) {
		live := &fakeCronLiveAdapter{err: errors.New("live down")}
		fallback := &fakeCronDeliverySink{err: errors.New("fallback down")}

		outcome := DeliverCronDeliveryPlan(context.Background(), plan, content, live, fallback)
		run := applyDeliveryOutcome(Run{Status: "success"}, outcome)

		if outcome.Delivered || outcome.Err == nil {
			t.Fatalf("outcome = %+v, want undelivered terminal error", outcome)
		}
		if run.Status != "success" {
			t.Fatalf("run status = %q, want success preserved", run.Status)
		}
		if run.Delivered {
			t.Fatal("run delivered = true, want false")
		}
		if !strings.Contains(run.ErrorMsg, DeliveryEvidenceLiveAdapterUnavailable) ||
			!strings.Contains(run.ErrorMsg, DeliveryEvidenceFallbackSinkUsed) {
			t.Fatalf("run error evidence = %q, want live/fallback evidence", run.ErrorMsg)
		}
	})
}

type staticDeliveryDirectory struct {
	targets map[string]DeliveryTarget
}

func (d staticDeliveryDirectory) HomeDeliveryTarget(platform string) (DeliveryTarget, bool) {
	target, ok := d.targets[strings.ToLower(strings.TrimSpace(platform))]
	return target, ok
}

type fakeCronLiveAdapter struct {
	err   error
	calls []fakeCronLiveCall
}

type fakeCronLiveCall struct {
	target DeliveryTarget
	text   string
	media  []MediaAttachment
}

func (a *fakeCronLiveAdapter) DeliverCron(ctx context.Context, target DeliveryTarget, text string, media []MediaAttachment) error {
	_ = ctx
	a.calls = append(a.calls, fakeCronLiveCall{
		target: target,
		text:   text,
		media:  append([]MediaAttachment(nil), media...),
	})
	return a.err
}

type fakeCronDeliverySink struct {
	err        error
	deliveries []string
}

func (s *fakeCronDeliverySink) Deliver(ctx context.Context, text string) error {
	_ = ctx
	s.deliveries = append(s.deliveries, text)
	return s.err
}

func normalizedDeliveryTargets(targets []DeliveryTarget) []string {
	out := make([]string, 0, len(targets))
	for _, target := range targets {
		out = append(out, target.Normalized())
	}
	return out
}

func mediaPaths(media []MediaAttachment) []string {
	out := make([]string, 0, len(media))
	for _, item := range media {
		out = append(out, item.Path)
	}
	return out
}

func assertDeliveryEvidence(t *testing.T, evidence []DeliveryEvidence, code string) {
	t.Helper()
	for _, item := range evidence {
		if item.Code == code {
			return
		}
	}
	t.Fatalf("evidence = %#v, want code %q", evidence, code)
}

func assertNoDeliveryEvidence(t *testing.T, evidence []DeliveryEvidence, code string) {
	t.Helper()
	for _, item := range evidence {
		if item.Code == code {
			t.Fatalf("evidence = %#v, did not want code %q", evidence, code)
		}
	}
}
