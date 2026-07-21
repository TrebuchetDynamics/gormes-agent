package webhook

import (
	"context"
	"errors"
	"time"
)

type RuntimeConfig struct {
	Routes             *DynamicRouteSet
	ProfileHome        string
	ScriptTimeout      time.Duration
	RateLimitPerMinute int
	MaxBodyBytes       int64
	Now                func() time.Time
	IdempotencyTTL     time.Duration
	OnAccepted         func(AcceptedWebhook)
}

type Runtime struct {
	routes         *DynamicRouteSet
	profileHome    string
	scriptTimeout  time.Duration
	rateLimit      int
	maxBodyBytes   int64
	now            func() time.Time
	idempotencyTTL time.Duration
	onAccepted     func(AcceptedWebhook)

	rateCounts map[string][]time.Time
	seen       map[string]time.Time
	accepted   []AcceptedWebhook
	lastReload error
}

type RuntimeResponse struct {
	StatusCode int
	Status     string
	Reason     string
	Error      string
	Route      string
	Event      string
	DeliveryID string
}

type AcceptedWebhook struct {
	Route      string
	Event      string
	DeliveryID string
	Parsed     ParsedInbound
	PromptDelivery
}

func NewRuntime(cfg RuntimeConfig) *Runtime {
	now := cfg.Now
	if now == nil {
		now = time.Now
	}
	ttl := cfg.IdempotencyTTL
	if ttl <= 0 {
		ttl = 24 * time.Hour
	}
	return &Runtime{
		routes:         cfg.Routes,
		profileHome:    trim(cfg.ProfileHome),
		scriptTimeout:  clampRouteScriptTimeout(cfg.ScriptTimeout),
		rateLimit:      cfg.RateLimitPerMinute,
		maxBodyBytes:   cfg.MaxBodyBytes,
		now:            now,
		idempotencyTTL: ttl,
		onAccepted:     cfg.OnAccepted,
		rateCounts:     map[string][]time.Time{},
		seen:           map[string]time.Time{},
	}
}

func (r *Runtime) Handle(routeName string, req InboundRequest) RuntimeResponse {
	return r.HandleContext(context.Background(), routeName, req)
}

func (r *Runtime) HandleContext(ctx context.Context, routeName string, req InboundRequest) RuntimeResponse {
	if ctx == nil {
		ctx = context.Background()
	}
	routeName = trim(routeName)
	route, ok := r.reloadAndRoute(routeName)
	if !ok {
		return RuntimeResponse{StatusCode: 404, Status: "error", Error: "Unknown route: " + routeName, Route: routeName}
	}

	if exceedsLimit(req, r.maxBodyBytes) {
		return RuntimeResponse{StatusCode: 413, Status: "error", Error: "Payload too large", Route: routeName}
	}

	if secret := trim(route.Secret); secret != "" && secret != InsecureNoAuth {
		if !ValidateSignature(req.Headers, req.Body, secret) {
			return RuntimeResponse{StatusCode: 401, Status: "error", Error: "Invalid signature", Route: routeName}
		}
	}

	if !r.allowRate(routeName) {
		return RuntimeResponse{StatusCode: 429, Status: "error", Error: "Rate limit exceeded", Route: routeName}
	}

	parsed, allowed, err := ParseInbound(req, IngressConfig{
		Secret:       route.Secret,
		Events:       route.Events,
		MaxBodyBytes: r.maxBodyBytes,
	})
	if err != nil {
		return runtimeErrorResponse(routeName, err)
	}
	if !allowed {
		return RuntimeResponse{StatusCode: 200, Status: "ignored", Route: routeName, Event: parsed.EventType}
	}
	if !routeFiltersMatchWithHome(route.Filters, parsed.Payload, parsed.EventType, req.Headers, r.profileHome) {
		return RuntimeResponse{StatusCode: 200, Status: "ignored", Reason: "filter", Route: routeName, Event: parsed.EventType}
	}
	if trim(route.Script) != "" {
		payload, ok := runRouteScript(ctx, r.profileHome, route.Script, parsed.Payload, r.scriptTimeout)
		if !ok {
			return RuntimeResponse{StatusCode: 200, Status: "ignored", Reason: "script", Route: routeName, Event: parsed.EventType}
		}
		parsed.Payload = payload
	}

	if r.seenDelivery(parsed.DeliveryID) {
		return RuntimeResponse{StatusCode: 200, Status: "duplicate", Route: routeName, Event: parsed.EventType, DeliveryID: parsed.DeliveryID}
	}

	delivery, err := BuildPromptDelivery(routeName, parsed.DeliveryID, parsed.EventType, route, parsed.Payload)
	if err != nil {
		return RuntimeResponse{StatusCode: 400, Status: "error", Error: err.Error(), Route: routeName, Event: parsed.EventType, DeliveryID: parsed.DeliveryID}
	}
	accepted := AcceptedWebhook{
		Route:          routeName,
		Event:          parsed.EventType,
		DeliveryID:     parsed.DeliveryID,
		Parsed:         parsed,
		PromptDelivery: delivery,
	}
	r.accepted = append(r.accepted, accepted)
	if r.onAccepted != nil {
		r.onAccepted(accepted)
	}

	return RuntimeResponse{
		StatusCode: 202,
		Status:     "accepted",
		Route:      routeName,
		Event:      parsed.EventType,
		DeliveryID: parsed.DeliveryID,
	}
}

func (r *Runtime) Accepted() []AcceptedWebhook {
	if r == nil {
		return nil
	}
	return append([]AcceptedWebhook(nil), r.accepted...)
}

func (r *Runtime) LastReloadError() error {
	if r == nil {
		return nil
	}
	return r.lastReload
}

func (r *Runtime) reloadAndRoute(name string) (RouteConfig, bool) {
	if r == nil || r.routes == nil {
		return RouteConfig{}, false
	}
	r.lastReload = r.routes.Reload()
	return r.routes.Route(name)
}

func (r *Runtime) allowRate(route string) bool {
	if r.rateLimit <= 0 {
		return true
	}
	now := r.now()
	window := r.rateCounts[route]
	keep := window[:0]
	for _, seen := range window {
		if now.Sub(seen) < time.Minute {
			keep = append(keep, seen)
		}
	}
	if len(keep) >= r.rateLimit {
		r.rateCounts[route] = keep
		return false
	}
	keep = append(keep, now)
	r.rateCounts[route] = keep
	return true
}

func (r *Runtime) seenDelivery(deliveryID string) bool {
	now := r.now()
	for id, seen := range r.seen {
		if now.Sub(seen) >= r.idempotencyTTL {
			delete(r.seen, id)
		}
	}
	if _, ok := r.seen[deliveryID]; ok {
		return true
	}
	r.seen[deliveryID] = now
	return false
}

func runtimeErrorResponse(route string, err error) RuntimeResponse {
	switch {
	case errors.Is(err, ErrPayloadTooLarge):
		return RuntimeResponse{StatusCode: 413, Status: "error", Error: "Payload too large", Route: route}
	case errors.Is(err, ErrInvalidSignature):
		return RuntimeResponse{StatusCode: 401, Status: "error", Error: "Invalid signature", Route: route}
	case errors.Is(err, ErrCannotParseBody):
		return RuntimeResponse{StatusCode: 400, Status: "error", Error: "Cannot parse body", Route: route}
	default:
		return RuntimeResponse{StatusCode: 400, Status: "error", Error: err.Error(), Route: route}
	}
}
