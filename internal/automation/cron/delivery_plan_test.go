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

func TestCronDeliveryPlanForJob_UsesStoredDeliverAndOrigin(t *testing.T) {
	job := Job{
		ID:      "job-1",
		Name:    "briefing",
		Deliver: "origin, discord",
		Origin: &DeliveryOrigin{
			Platform: "Telegram",
			ChatID:   "-100777",
			ThreadID: "99",
		},
	}
	directory := staticDeliveryDirectory{
		targets: map[string]DeliveryTarget{
			"discord": {Platform: "discord", ChatID: "home-channel", ThreadID: "thread-7"},
		},
	}

	plan := PlanCronDeliveryForJob(job, directory)
	if got, want := normalizedDeliveryTargets(plan.Targets), []string{"telegram:-100777:99", "discord:home-channel:thread-7"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("targets = %#v, want %#v; evidence=%#v", got, want, plan.Evidence)
	}
	if len(plan.Evidence) != 0 {
		t.Fatalf("evidence = %#v, want none", plan.Evidence)
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

func TestCronDeliveryPlan_RoutingIntentAllExpandsHomeTargets(t *testing.T) {
	directory := staticDeliveryDirectory{
		targets: map[string]DeliveryTarget{
			"telegram": {Platform: "telegram", ChatID: "-100777", ThreadID: "99"},
			"discord":  {Platform: "discord", ChatID: "home-channel", ThreadID: "thread-7"},
			"slack":    {Platform: "slack", ChatID: "C123"},
		},
	}

	plan := PlanCronDelivery(DeliveryPlanOptions{Deliver: "all", Directory: directory})

	want := []string{"discord:home-channel:thread-7", "slack:C123", "telegram:-100777:99"}
	if got := normalizedDeliveryTargets(plan.Targets); !reflect.DeepEqual(got, want) {
		t.Fatalf("targets = %#v, want %#v; evidence=%#v", got, want, plan.Evidence)
	}
	if len(plan.Evidence) != 0 {
		t.Fatalf("evidence = %#v, want none", plan.Evidence)
	}
}

func TestCronDeliveryPlan_RoutingIntentAllComposesAndDedupes(t *testing.T) {
	origin := &DeliveryOrigin{Platform: "telegram", ChatID: "-100777", ThreadID: "99"}
	directory := staticDeliveryDirectory{
		targets: map[string]DeliveryTarget{
			"telegram": {Platform: "telegram", ChatID: "-100777", ThreadID: "99"},
			"discord":  {Platform: "discord", ChatID: "home-channel", ThreadID: "thread-7"},
		},
	}

	plan := PlanCronDelivery(DeliveryPlanOptions{
		Deliver:   "origin,all,telegram:-100777:99,local",
		Origin:    origin,
		Directory: directory,
	})

	want := []string{"telegram:-100777:99", "discord:home-channel:thread-7", "local"}
	if got := normalizedDeliveryTargets(plan.Targets); !reflect.DeepEqual(got, want) {
		t.Fatalf("targets = %#v, want %#v; evidence=%#v", got, want, plan.Evidence)
	}
	if len(plan.Evidence) != 0 {
		t.Fatalf("evidence = %#v, want none", plan.Evidence)
	}
}

func TestCronDeliveryPlan_RoutingIntentAllMissingDirectoryReturnsEvidence(t *testing.T) {
	tests := []struct {
		name      string
		directory DeliveryTargetDirectory
	}{
		{name: "nil directory"},
		{name: "empty directory", directory: staticDeliveryDirectory{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plan := PlanCronDelivery(DeliveryPlanOptions{Deliver: "all", Directory: tt.directory})
			if len(plan.Targets) != 0 {
				t.Fatalf("targets = %#v, want none", plan.Targets)
			}
			assertDeliveryEvidence(t, plan.Evidence, DeliveryEvidenceChannelDirectoryMissing)
			if got := plan.Evidence[0].Target; got != "all" {
				t.Fatalf("evidence target = %q, want all", got)
			}
		})
	}
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

func TestCronDeliveryPlan_StandaloneSenderFallbackUsesRegisteredSender(t *testing.T) {
	plan := PlanCronDelivery(DeliveryPlanOptions{Deliver: "google_chat:spaces/AAA:spaces/AAA/threads/thread-1"})
	content := PrepareCronDeliveryContent("standalone cron response [MEDIA:outputs/report.txt]")
	standalone := &fakeCronStandaloneSender{}
	fallback := &fakeCronDeliverySink{}

	outcome := DeliverCronDeliveryPlanWithStandalone(context.Background(), plan, content, nil, standalone, fallback)

	if !outcome.Delivered || outcome.Err != nil {
		t.Fatalf("outcome = %+v, want delivered by standalone sender", outcome)
	}
	if got, want := len(standalone.calls), 1; got != want {
		t.Fatalf("standalone calls = %d, want %d", got, want)
	}
	call := standalone.calls[0]
	if call.target.Normalized() != "google_chat:spaces/AAA:spaces/AAA/threads/thread-1" {
		t.Fatalf("standalone target = %q", call.target.Normalized())
	}
	if call.text != "standalone cron response" {
		t.Fatalf("standalone text = %q", call.text)
	}
	if got, want := mediaPaths(call.media), []string{"outputs/report.txt"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("standalone media = %#v, want %#v", got, want)
	}
	if len(fallback.deliveries) != 0 {
		t.Fatalf("fallback deliveries = %#v, want none", fallback.deliveries)
	}
	assertDeliveryEvidence(t, outcome.Evidence, DeliveryEvidenceLiveAdapterUnavailable)
	assertDeliveryEvidence(t, outcome.Evidence, DeliveryEvidenceStandaloneSenderUsed)
	assertNoDeliveryEvidence(t, outcome.Evidence, DeliveryEvidenceFallbackSinkUsed)
}

func TestCronDeliveryPlan_StandaloneSenderFallbackPreservesLiveAdapterFirst(t *testing.T) {
	plan := PlanCronDelivery(DeliveryPlanOptions{Deliver: "teams:conversation-1"})
	content := PrepareCronDeliveryContent("live response")
	live := &fakeCronLiveAdapter{}
	standalone := &fakeCronStandaloneSender{}
	fallback := &fakeCronDeliverySink{}

	outcome := DeliverCronDeliveryPlanWithStandalone(context.Background(), plan, content, live, standalone, fallback)

	if !outcome.Delivered || outcome.Err != nil {
		t.Fatalf("outcome = %+v, want delivered by live adapter", outcome)
	}
	if got := len(live.calls); got != 1 {
		t.Fatalf("live calls = %d, want 1", got)
	}
	if got := len(standalone.calls); got != 0 {
		t.Fatalf("standalone calls = %d, want 0", got)
	}
	if got := len(fallback.deliveries); got != 0 {
		t.Fatalf("fallback deliveries = %d, want 0", got)
	}
	assertNoDeliveryEvidence(t, outcome.Evidence, DeliveryEvidenceStandaloneSenderUsed)
	assertNoDeliveryEvidence(t, outcome.Evidence, DeliveryEvidenceFallbackSinkUsed)
}

func TestCronDeliveryPlan_StandaloneSenderFallbackFailureFallsBack(t *testing.T) {
	plan := PlanCronDelivery(DeliveryPlanOptions{Deliver: "irc:#ops"})
	content := PrepareCronDeliveryContent("fallback response")
	standalone := &fakeCronStandaloneSender{err: errors.New("irc dial failed")}
	fallback := &fakeCronDeliverySink{}

	outcome := DeliverCronDeliveryPlanWithStandalone(context.Background(), plan, content, nil, standalone, fallback)

	if !outcome.Delivered || outcome.Err != nil {
		t.Fatalf("outcome = %+v, want delivered by fallback after standalone failure", outcome)
	}
	if got := len(standalone.calls); got != 1 {
		t.Fatalf("standalone calls = %d, want 1", got)
	}
	if got, want := fallback.deliveries, []string{"fallback response"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("fallback deliveries = %#v, want %#v", got, want)
	}
	assertDeliveryEvidence(t, outcome.Evidence, DeliveryEvidenceLiveAdapterUnavailable)
	assertDeliveryEvidence(t, outcome.Evidence, DeliveryEvidenceStandaloneSenderFailed)
	assertDeliveryEvidence(t, outcome.Evidence, DeliveryEvidenceFallbackSinkUsed)
}

type staticDeliveryDirectory struct {
	targets map[string]DeliveryTarget
}

func (d staticDeliveryDirectory) HomeDeliveryTarget(platform string) (DeliveryTarget, bool) {
	target, ok := d.targets[strings.ToLower(strings.TrimSpace(platform))]
	return target, ok
}

func (d staticDeliveryDirectory) HomeDeliveryTargets() []DeliveryTarget {
	out := make([]DeliveryTarget, 0, len(d.targets))
	for platform, target := range d.targets {
		if target.Platform == "" {
			target.Platform = platform
		}
		out = append(out, target)
	}
	return out
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

type fakeCronStandaloneSender struct {
	err   error
	calls []fakeCronStandaloneCall
}

type fakeCronStandaloneCall struct {
	target DeliveryTarget
	text   string
	media  []MediaAttachment
}

func (s *fakeCronStandaloneSender) DeliverCronStandalone(ctx context.Context, target DeliveryTarget, text string, media []MediaAttachment) error {
	_ = ctx
	s.calls = append(s.calls, fakeCronStandaloneCall{
		target: target,
		text:   text,
		media:  append([]MediaAttachment(nil), media...),
	})
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
