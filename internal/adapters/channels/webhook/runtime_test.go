package webhook

import (
	"testing"
	"time"
)

func TestWebhookSignatureBeforeRateLimit_InvalidSignaturesDoNotConsumeQuota(t *testing.T) {
	clock := fakeWebhookClock{now: time.Unix(1700000000, 0)}
	rt := NewRuntime(RuntimeConfig{
		Routes: NewDynamicRouteSet(t.TempDir(), map[string]RouteConfig{
			"alerts": {
				Secret: "route-secret",
				Events: []string{"push"},
				Prompt: "Push {ref}",
			},
		}),
		RateLimitPerMinute: 2,
		MaxBodyBytes:       1024,
		Now:                clock.Now,
	})
	body := []byte(`{"ref":"main"}`)

	for i := 0; i < 2; i++ {
		resp := rt.Handle("alerts", InboundRequest{
			Headers: map[string]string{
				"X-GitHub-Event":      "push",
				"X-GitHub-Delivery":   "bad",
				"X-Hub-Signature-256": "sha256=invalid",
			},
			Body:          body,
			ContentLength: int64(len(body)),
		})
		if resp.StatusCode != 401 || resp.Status != "error" {
			t.Fatalf("bad signature response = %+v, want 401 error", resp)
		}
	}

	resp := rt.Handle("alerts", InboundRequest{
		Headers: map[string]string{
			"X-GitHub-Event":      "push",
			"X-GitHub-Delivery":   "good-1",
			"X-Hub-Signature-256": githubSignature(body, "route-secret"),
		},
		Body:          body,
		ContentLength: int64(len(body)),
	})
	if resp.StatusCode != 202 || resp.Status != "accepted" {
		t.Fatalf("valid signed response after bad signatures = %+v, want 202 accepted", resp)
	}
	if len(rt.Accepted()) != 1 {
		t.Fatalf("Accepted count = %d, want 1", len(rt.Accepted()))
	}
}

func TestWebhookSignatureBeforeRateLimit_ValidSignedRequestsStillRateLimit(t *testing.T) {
	clock := fakeWebhookClock{now: time.Unix(1700000000, 0)}
	rt := NewRuntime(RuntimeConfig{
		Routes: NewDynamicRouteSet(t.TempDir(), map[string]RouteConfig{
			"alerts": {
				Secret: "route-secret",
				Events: []string{"push"},
				Prompt: "Push {ref}",
			},
		}),
		RateLimitPerMinute: 2,
		MaxBodyBytes:       1024,
		Now:                clock.Now,
	})
	body := []byte(`{"ref":"main"}`)

	for _, deliveryID := range []string{"good-1", "good-2"} {
		resp := rt.Handle("alerts", signedWebhookRequest(body, "route-secret", deliveryID))
		if resp.StatusCode != 202 {
			t.Fatalf("Handle(%s) = %+v, want 202", deliveryID, resp)
		}
	}
	resp := rt.Handle("alerts", signedWebhookRequest(body, "route-secret", "good-3"))
	if resp.StatusCode != 429 || resp.Status != "error" {
		t.Fatalf("third valid response = %+v, want 429 error", resp)
	}
}

func TestWebhookRuntime_ReloadsDynamicRoutesAndBuildsAcceptedEnvelope(t *testing.T) {
	home := t.TempDir()
	writeDynamicRoutes(t, home+"/"+DynamicRoutesFilename, `{
		"deploy": {
			"secret": "dynamic-secret",
			"events": ["deployment"],
			"prompt": "Deploy {environment} from {sender.login}",
			"deliver": "telegram",
			"deliver_extra": {"chat_id": "-100123", "thread_id": "{environment}"}
		}
	}`)
	rt := NewRuntime(RuntimeConfig{
		Routes:             NewDynamicRouteSet(home, nil),
		RateLimitPerMinute: 5,
		MaxBodyBytes:       1024,
		Now:                func() time.Time { return time.Unix(1700000000, 0) },
	})
	body := []byte(`{"event_type":"deployment","environment":"prod","sender":{"login":"alice"}}`)

	resp := rt.Handle("deploy", InboundRequest{
		Headers: map[string]string{
			"X-Request-ID":        "deploy-1",
			"X-Webhook-Signature": genericSignature(body, "dynamic-secret"),
		},
		Body:          body,
		ContentLength: int64(len(body)),
	})
	if resp.StatusCode != 202 || resp.Status != "accepted" {
		t.Fatalf("dynamic route response = %+v, want 202 accepted", resp)
	}
	if resp.Route != "deploy" || resp.Event != "deployment" || resp.DeliveryID != "deploy-1" {
		t.Fatalf("response envelope = %+v, want route/event/delivery evidence", resp)
	}
	accepted := rt.Accepted()
	if len(accepted) != 1 {
		t.Fatalf("Accepted count = %d, want 1", len(accepted))
	}
	if accepted[0].Prompt != "Deploy prod from alice" {
		t.Fatalf("Prompt = %q, want rendered prompt", accepted[0].Prompt)
	}
	if accepted[0].Target.ChatID != "-100123" || accepted[0].Target.ThreadID != "prod" {
		t.Fatalf("Target = %+v, want rendered Telegram target", accepted[0].Target)
	}

	duplicate := rt.Handle("deploy", InboundRequest{
		Headers: map[string]string{
			"X-Request-ID":        "deploy-1",
			"X-Webhook-Signature": genericSignature(body, "dynamic-secret"),
		},
		Body:          body,
		ContentLength: int64(len(body)),
	})
	if duplicate.StatusCode != 200 || duplicate.Status != "duplicate" {
		t.Fatalf("duplicate response = %+v, want duplicate envelope", duplicate)
	}
}

func TestWebhookRuntime_RequestAdmissionEnvelope(t *testing.T) {
	rt := NewRuntime(RuntimeConfig{
		Routes: NewDynamicRouteSet(t.TempDir(), map[string]RouteConfig{
			"alerts": {
				Secret: "route-secret",
				Events: []string{"push"},
				Prompt: "Push {ref}",
			},
		}),
		RateLimitPerMinute: 5,
		MaxBodyBytes:       12,
		Now:                func() time.Time { return time.Unix(1700000000, 0) },
	})

	if resp := rt.Handle("missing", signedWebhookRequest([]byte(`{"ref":"main"}`), "route-secret", "missing-1")); resp.StatusCode != 404 {
		t.Fatalf("unknown route response = %+v, want 404", resp)
	}

	oversizedBody := []byte(`{"ref":"longer-than-limit"}`)
	oversized := rt.Handle("alerts", InboundRequest{
		Headers: map[string]string{
			"X-GitHub-Event":      "push",
			"X-GitHub-Delivery":   "oversized-1",
			"X-Hub-Signature-256": githubSignature(oversizedBody, "route-secret"),
		},
		Body:          oversizedBody,
		ContentLength: int64(len(oversizedBody)),
	})
	if oversized.StatusCode != 413 {
		t.Fatalf("oversized response = %+v, want 413", oversized)
	}

	badJSON := []byte(`%zz`)
	parseErr := rt.Handle("alerts", InboundRequest{
		Headers: map[string]string{
			"X-GitHub-Event":      "push",
			"X-GitHub-Delivery":   "bad-json-1",
			"X-Hub-Signature-256": githubSignature(badJSON, "route-secret"),
		},
		Body:          badJSON,
		ContentLength: int64(len(badJSON)),
	})
	if parseErr.StatusCode != 400 {
		t.Fatalf("parse error response = %+v, want 400", parseErr)
	}

	filteredBody := []byte(`{}`)
	filtered := rt.Handle("alerts", InboundRequest{
		Headers: map[string]string{
			"X-GitHub-Event":      "deployment",
			"X-GitHub-Delivery":   "filtered-1",
			"X-Hub-Signature-256": githubSignature(filteredBody, "route-secret"),
		},
		Body:          filteredBody,
		ContentLength: int64(len(filteredBody)),
	})
	if filtered.StatusCode != 200 || filtered.Status != "ignored" || filtered.Event != "deployment" {
		t.Fatalf("filtered response = %+v, want ignored deployment", filtered)
	}
	if got := len(rt.Accepted()); got != 0 {
		t.Fatalf("Accepted count = %d, want 0 for rejected/ignored requests", got)
	}
}

func signedWebhookRequest(body []byte, secret, deliveryID string) InboundRequest {
	return InboundRequest{
		Headers: map[string]string{
			"X-GitHub-Event":      "push",
			"X-GitHub-Delivery":   deliveryID,
			"X-Hub-Signature-256": githubSignature(body, secret),
		},
		Body:          body,
		ContentLength: int64(len(body)),
	}
}

type fakeWebhookClock struct {
	now time.Time
}

func (c fakeWebhookClock) Now() time.Time {
	return c.now
}
