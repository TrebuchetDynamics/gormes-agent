package cron

import (
	"context"
	"errors"
	"fmt"
	"path"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
)

const (
	DeliveryEvidenceTargetParseFailed       = "target_parse_failed"
	DeliveryEvidenceChannelDirectoryMissing = "channel_directory_missing"
	DeliveryEvidenceMediaIgnored            = "media_ignored"
	DeliveryEvidenceLiveAdapterUnavailable  = "live_adapter_unavailable"
	DeliveryEvidenceStandaloneSenderUsed    = "standalone_sender_used"
	DeliveryEvidenceStandaloneSenderFailed  = "standalone_sender_failed"
	DeliveryEvidenceFallbackSinkUsed        = "fallback_sink_used"
)

var ErrLiveAdapterUnavailable = errors.New("cron: live adapter unavailable")
var ErrStandaloneSenderUnavailable = errors.New("cron: standalone sender unavailable")

type DeliveryOrigin struct {
	Platform string
	ChatID   string
	ThreadID string
}

type DeliveryPlanOptions struct {
	Deliver   string
	Origin    *DeliveryOrigin
	Directory DeliveryTargetDirectory
}

type DeliveryTargetDirectory interface {
	HomeDeliveryTarget(platform string) (DeliveryTarget, bool)
}

type DeliveryAllTargetDirectory interface {
	HomeDeliveryTargets() []DeliveryTarget
}

type DeliveryPlan struct {
	Targets  []DeliveryTarget
	Evidence []DeliveryEvidence
}

type DeliveryTarget struct {
	Platform string
	ChatID   string
	ThreadID string
	Local    bool
	Origin   bool
	Explicit bool
}

func (t DeliveryTarget) Normalized() string {
	if t.Local || (strings.EqualFold(strings.TrimSpace(t.Platform), "local") && strings.TrimSpace(t.ChatID) == "") {
		return "local"
	}
	platform := strings.ToLower(strings.TrimSpace(t.Platform))
	chatID := strings.TrimSpace(t.ChatID)
	threadID := strings.TrimSpace(t.ThreadID)
	if chatID == "" {
		return platform
	}
	if threadID == "" {
		return platform + ":" + chatID
	}
	return platform + ":" + chatID + ":" + threadID
}

type MediaAttachment struct {
	Path string
}

type DeliveryContent struct {
	Text     string
	Media    []MediaAttachment
	Evidence []DeliveryEvidence
}

type DeliveryEvidence struct {
	Code   string
	Target string
	Detail string
}

type LiveDeliveryAdapter interface {
	DeliverCron(ctx context.Context, target DeliveryTarget, text string, media []MediaAttachment) error
}

type StandaloneDeliveryAdapter interface {
	DeliverCronStandalone(ctx context.Context, target DeliveryTarget, text string, media []MediaAttachment) error
}

type DeliveryOutcome struct {
	Delivered bool
	Evidence  []DeliveryEvidence
	Err       error
}

func PlanCronDelivery(opts DeliveryPlanOptions) DeliveryPlan {
	deliver := strings.TrimSpace(opts.Deliver)
	if deliver == "" {
		deliver = "local"
	}

	var plan DeliveryPlan
	seen := map[string]struct{}{}
	for _, raw := range strings.Split(deliver, ",") {
		token := strings.TrimSpace(raw)
		if token == "" {
			plan.Evidence = append(plan.Evidence, DeliveryEvidence{
				Code:   DeliveryEvidenceTargetParseFailed,
				Target: "[empty]",
				Detail: "empty delivery target",
			})
			continue
		}

		targets, evidence := resolveCronDeliveryTargets(token, opts)
		plan.Evidence = append(plan.Evidence, evidence...)
		for _, target := range targets {
			key := target.Normalized()
			if _, exists := seen[key]; exists {
				continue
			}
			seen[key] = struct{}{}
			plan.Targets = append(plan.Targets, target)
		}
	}
	return plan
}

func PlanCronDeliveryForJob(job any, directory DeliveryTargetDirectory) DeliveryPlan {
	opts := DeliveryPlanOptions{Directory: directory}
	v := reflect.ValueOf(job)
	if !v.IsValid() {
		return PlanCronDelivery(opts)
	}
	if v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return PlanCronDelivery(opts)
		}
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return PlanCronDelivery(opts)
	}
	if f := v.FieldByName("Deliver"); f.IsValid() && f.Kind() == reflect.String {
		opts.Deliver = f.String()
	}
	if origin, ok := deliveryOriginFromValue(v.FieldByName("Origin")); ok {
		opts.Origin = &origin
	}
	return PlanCronDelivery(opts)
}

func resolveCronDeliveryTargets(raw string, opts DeliveryPlanOptions) ([]DeliveryTarget, []DeliveryEvidence) {
	if strings.EqualFold(strings.TrimSpace(raw), "all") {
		return expandCronDeliveryAllTargets(opts)
	}
	target, evidence, ok := resolveCronDeliveryTarget(raw, opts)
	if evidence.Code != "" {
		if ok {
			return []DeliveryTarget{target}, []DeliveryEvidence{evidence}
		}
		return nil, []DeliveryEvidence{evidence}
	}
	if !ok {
		return nil, nil
	}
	return []DeliveryTarget{target}, nil
}

func expandCronDeliveryAllTargets(opts DeliveryPlanOptions) ([]DeliveryTarget, []DeliveryEvidence) {
	directory, ok := opts.Directory.(DeliveryAllTargetDirectory)
	if !ok || directory == nil {
		return nil, []DeliveryEvidence{{
			Code:   DeliveryEvidenceChannelDirectoryMissing,
			Target: "all",
			Detail: "home delivery targets unavailable",
		}}
	}
	targets := make([]DeliveryTarget, 0)
	for _, target := range directory.HomeDeliveryTargets() {
		target.Platform = strings.ToLower(strings.TrimSpace(target.Platform))
		target.ChatID = strings.TrimSpace(target.ChatID)
		target.ThreadID = strings.TrimSpace(target.ThreadID)
		target.Local = false
		target.Origin = false
		target.Explicit = false
		if target.Platform == "" || strings.EqualFold(target.Platform, "local") || target.ChatID == "" {
			continue
		}
		targets = append(targets, target)
	}
	sort.SliceStable(targets, func(i, j int) bool {
		return targets[i].Normalized() < targets[j].Normalized()
	})
	if len(targets) == 0 {
		return nil, []DeliveryEvidence{{
			Code:   DeliveryEvidenceChannelDirectoryMissing,
			Target: "all",
			Detail: "no home delivery targets configured",
		}}
	}
	return targets, nil
}

func resolveCronDeliveryTarget(raw string, opts DeliveryPlanOptions) (DeliveryTarget, DeliveryEvidence, bool) {
	if strings.EqualFold(raw, "local") {
		return DeliveryTarget{Platform: "local", Local: true}, DeliveryEvidence{}, true
	}
	if strings.EqualFold(raw, "origin") {
		if target, ok := targetFromOrigin(opts.Origin); ok {
			return target, DeliveryEvidence{}, true
		}
		return DeliveryTarget{}, DeliveryEvidence{
			Code:   DeliveryEvidenceChannelDirectoryMissing,
			Target: "origin",
			Detail: "origin target unavailable",
		}, false
	}

	if strings.Contains(raw, ":") {
		parsed, err := gateway.ParseDeliveryTarget(raw, nil)
		if err != nil {
			return DeliveryTarget{}, DeliveryEvidence{
				Code:   DeliveryEvidenceTargetParseFailed,
				Target: raw,
				Detail: err.Error(),
			}, false
		}
		return targetFromGateway(parsed), DeliveryEvidence{}, true
	}

	platform := strings.ToLower(strings.TrimSpace(raw))
	if platform == "" {
		return DeliveryTarget{}, DeliveryEvidence{
			Code:   DeliveryEvidenceTargetParseFailed,
			Target: "[empty]",
			Detail: "empty delivery platform",
		}, false
	}
	if opts.Origin != nil && strings.EqualFold(strings.TrimSpace(opts.Origin.Platform), platform) {
		if target, ok := targetFromOrigin(opts.Origin); ok {
			return target, DeliveryEvidence{}, true
		}
	}
	if opts.Directory != nil {
		if target, ok := opts.Directory.HomeDeliveryTarget(platform); ok {
			target.Platform = strings.ToLower(strings.TrimSpace(firstNonEmpty(target.Platform, platform)))
			return target, DeliveryEvidence{}, true
		}
	}
	return DeliveryTarget{}, DeliveryEvidence{
		Code:   DeliveryEvidenceChannelDirectoryMissing,
		Target: platform,
		Detail: "home delivery target unavailable",
	}, false
}

func targetFromOrigin(origin *DeliveryOrigin) (DeliveryTarget, bool) {
	if origin == nil {
		return DeliveryTarget{}, false
	}
	platform := strings.ToLower(strings.TrimSpace(origin.Platform))
	chatID := strings.TrimSpace(origin.ChatID)
	if platform == "" || chatID == "" {
		return DeliveryTarget{}, false
	}
	return DeliveryTarget{
		Platform: platform,
		ChatID:   chatID,
		ThreadID: strings.TrimSpace(origin.ThreadID),
		Origin:   true,
	}, true
}

func targetFromGateway(target gateway.DeliveryTarget) DeliveryTarget {
	return DeliveryTarget{
		Platform: strings.ToLower(strings.TrimSpace(target.Platform)),
		ChatID:   strings.TrimSpace(target.ChatID),
		ThreadID: strings.TrimSpace(target.ThreadID),
		Local:    strings.EqualFold(target.Platform, "local"),
		Origin:   target.IsOrigin,
		Explicit: target.IsExplicit,
	}
}

var mediaTagRE = regexp.MustCompile(`\[MEDIA:([^\]]*)\]`)

func PrepareCronDeliveryContent(finalText string) DeliveryContent {
	var out DeliveryContent
	cleaned := mediaTagRE.ReplaceAllStringFunc(finalText, func(tag string) string {
		matches := mediaTagRE.FindStringSubmatch(tag)
		if len(matches) != 2 {
			return tag
		}
		mediaPath, ok := cleanMediaPath(matches[1])
		if !ok {
			out.Evidence = append(out.Evidence, DeliveryEvidence{
				Code:   DeliveryEvidenceMediaIgnored,
				Target: "[redacted]",
				Detail: "unsafe media path redacted",
			})
			return "[MEDIA:redacted]"
		}
		out.Media = append(out.Media, MediaAttachment{Path: mediaPath})
		return ""
	})
	out.Text = trimDeliveryText(cleaned)
	return out
}

func cleanMediaPath(raw string) (string, bool) {
	value := strings.TrimSpace(raw)
	if value == "" || strings.ContainsRune(value, 0) || filepath.IsAbs(value) {
		return "", false
	}
	value = strings.ReplaceAll(value, "\\", "/")
	for _, segment := range strings.Split(value, "/") {
		if segment == ".." {
			return "", false
		}
	}
	cleaned := path.Clean(value)
	if cleaned == "." || strings.HasPrefix(cleaned, "../") || cleaned == ".." {
		return "", false
	}
	return cleaned, true
}

func trimDeliveryText(text string) string {
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimRight(line, " \t")
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

func DeliverCronDeliveryPlan(ctx context.Context, plan DeliveryPlan, content DeliveryContent, live LiveDeliveryAdapter, fallback DeliverySink) DeliveryOutcome {
	return DeliverCronDeliveryPlanWithStandalone(ctx, plan, content, live, nil, fallback)
}

func DeliverCronDeliveryPlanWithStandalone(ctx context.Context, plan DeliveryPlan, content DeliveryContent, live LiveDeliveryAdapter, standalone StandaloneDeliveryAdapter, fallback DeliverySink) DeliveryOutcome {
	evidence := append([]DeliveryEvidence{}, plan.Evidence...)
	evidence = append(evidence, content.Evidence...)

	if len(plan.Targets) == 0 {
		if len(evidence) == 0 {
			return DeliveryOutcome{Delivered: false}
		}
		err := fmt.Errorf("cron delivery target unavailable")
		return DeliveryOutcome{Delivered: false, Evidence: evidence, Err: err}
	}

	delivered := true
	var errs []error
	for _, target := range plan.Targets {
		ok, targetEvidence, err := deliverCronTarget(ctx, target, content, live, standalone, fallback)
		evidence = append(evidence, targetEvidence...)
		if !ok {
			delivered = false
		}
		if err != nil {
			errs = append(errs, err)
		}
	}

	return DeliveryOutcome{
		Delivered: delivered,
		Evidence:  evidence,
		Err:       errors.Join(errs...),
	}
}

func deliverCronTarget(ctx context.Context, target DeliveryTarget, content DeliveryContent, live LiveDeliveryAdapter, standalone StandaloneDeliveryAdapter, fallback DeliverySink) (bool, []DeliveryEvidence, error) {
	if target.Normalized() != "local" {
		if live != nil {
			if err := live.DeliverCron(ctx, target, content.Text, content.Media); err == nil {
				return true, nil, nil
			}
		}
		evidence := []DeliveryEvidence{{
			Code:   DeliveryEvidenceLiveAdapterUnavailable,
			Target: target.Normalized(),
			Detail: "live adapter unavailable",
		}}
		if standalone != nil {
			if err := standalone.DeliverCronStandalone(ctx, target, content.Text, content.Media); err == nil {
				evidence = append(evidence, DeliveryEvidence{
					Code:   DeliveryEvidenceStandaloneSenderUsed,
					Target: target.Normalized(),
					Detail: "standalone sender used",
				})
				return true, evidence, nil
			} else if !errors.Is(err, ErrStandaloneSenderUnavailable) {
				evidence = append(evidence, DeliveryEvidence{
					Code:   DeliveryEvidenceStandaloneSenderFailed,
					Target: target.Normalized(),
					Detail: "standalone sender failed",
				})
			}
		}
		ok, fallbackEvidence, err := deliverViaFallbackSink(ctx, target, content, fallback, true)
		evidence = append(evidence, fallbackEvidence...)
		return ok, evidence, err
	}
	return deliverViaFallbackSink(ctx, target, content, fallback, false)
}

func deliverViaFallbackSink(ctx context.Context, target DeliveryTarget, content DeliveryContent, fallback DeliverySink, reportFallback bool) (bool, []DeliveryEvidence, error) {
	var evidence []DeliveryEvidence
	if reportFallback {
		evidence = append(evidence, DeliveryEvidence{
			Code:   DeliveryEvidenceFallbackSinkUsed,
			Target: target.Normalized(),
			Detail: "existing cron delivery sink used",
		})
	}
	if len(content.Media) > 0 {
		evidence = append(evidence, DeliveryEvidence{
			Code:   DeliveryEvidenceMediaIgnored,
			Target: target.Normalized(),
			Detail: "delivery sink supports text only",
		})
	}
	if fallback == nil {
		return false, evidence, errors.New("cron delivery fallback sink unavailable")
	}
	if err := fallback.Deliver(ctx, content.Text); err != nil {
		return false, evidence, fmt.Errorf("cron delivery fallback sink: %w", err)
	}
	return true, evidence, nil
}

func applyDeliveryOutcome(run Run, outcome DeliveryOutcome) Run {
	run.Delivered = outcome.Delivered
	if msg := deliveryOutcomeMessage(outcome); msg != "" {
		if strings.TrimSpace(run.ErrorMsg) != "" {
			run.ErrorMsg += "; " + msg
		} else {
			run.ErrorMsg = msg
		}
	}
	return run
}

func deliveryOutcomeMessage(outcome DeliveryOutcome) string {
	var parts []string
	if outcome.Err != nil {
		parts = append(parts, outcome.Err.Error())
	}
	if evidence := formatDeliveryEvidence(outcome.Evidence); evidence != "" {
		parts = append(parts, "delivery evidence: "+evidence)
	}
	return strings.Join(parts, "; ")
}

func formatDeliveryEvidence(evidence []DeliveryEvidence) string {
	parts := make([]string, 0, len(evidence))
	for _, item := range evidence {
		code := strings.TrimSpace(item.Code)
		if code == "" {
			continue
		}
		part := code
		if target := strings.TrimSpace(item.Target); target != "" {
			part += ":" + target
		}
		if detail := strings.TrimSpace(item.Detail); detail != "" {
			part += "(" + detail + ")"
		}
		parts = append(parts, part)
	}
	return strings.Join(parts, "; ")
}

func deliveryOriginFromValue(v reflect.Value) (DeliveryOrigin, bool) {
	if !v.IsValid() {
		return DeliveryOrigin{}, false
	}
	if v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return DeliveryOrigin{}, false
		}
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return DeliveryOrigin{}, false
	}
	origin := DeliveryOrigin{
		Platform: stringField(v, "Platform"),
		ChatID:   firstNonEmpty(stringField(v, "ChatID"), stringField(v, "ChatId")),
		ThreadID: firstNonEmpty(stringField(v, "ThreadID"), stringField(v, "ThreadId")),
	}
	if strings.TrimSpace(origin.Platform) == "" || strings.TrimSpace(origin.ChatID) == "" {
		return DeliveryOrigin{}, false
	}
	return origin, true
}

func stringField(v reflect.Value, name string) string {
	f := v.FieldByName(name)
	if !f.IsValid() || f.Kind() != reflect.String {
		return ""
	}
	return f.String()
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
