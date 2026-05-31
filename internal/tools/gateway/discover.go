package gateway

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"
)

const (
	GatewayBonjourServiceType = "_openclaw-gw._tcp"

	GatewayEndpointSourceBonjour = "bonjour"
	GatewayEndpointSourceManual  = "manual"

	GatewayDiscoverCodeCompleted   = "gateway_discover_completed"
	GatewayDiscoverCodeNoGateways  = "gateway_no_gateways_discovered"
	GatewayDiscoverCodeUnavailable = "gateway_discover_unavailable"

	GatewayProbeCodeCompleted   = "gateway_probe_completed"
	GatewayProbeCodeUnreachable = "gateway_probe_unreachable"

	GatewayUsageCostCodeCompleted   = "gateway_usage_cost_completed"
	GatewayUsageCostCodeUnavailable = "gateway_usage_cost_unavailable"

	GatewayDegradedNoGateways            = "no_gateways_discovered"
	GatewayDegradedBonjourUnavailable    = "bonjour_discovery_unavailable"
	GatewayDegradedProbeTimeout          = "probe_timeout"
	GatewayDegradedHTTPProbeUnavailable  = "http_probe_unavailable"
	GatewayDegradedHTTPUnauthorized      = "http_probe_unauthorized"
	GatewayDegradedCapabilityUnsupported = "http_capability_unsupported"
	GatewayDegradedCapabilityMalformed   = "http_capability_malformed"
	GatewayDegradedUsageDataUnavailable  = "usage_data_unavailable"

	GatewayHealthTCPReachable              = "tcp_reachable"
	GatewayHealthUnreachable               = "unreachable"
	GatewayHealthHTTPHealthy               = "http_healthy"
	GatewayHealthHTTPUnavailable           = "http_unavailable"
	GatewayHealthHTTPUnauthorized          = "http_unauthorized"
	GatewayHealthHTTPCapabilityUnsupported = "capability_unsupported"
	GatewayHealthHTTPCapabilityMalformed   = "capability_malformed"

	GatewayProbeStatusRuntimeRunning        = "runtime_running"
	GatewayProbeStatusUnavailable           = "unavailable"
	GatewayProbeStatusCapabilityReady       = "capability_ready"
	GatewayProbeStatusUnauthorized          = "unauthorized"
	GatewayProbeStatusUnsupportedCapability = "unsupported_capability"
	GatewayProbeStatusMalformedCapabilities = "malformed_capabilities"
)

const (
	defaultGatewayDiscoveryTimeout = 2 * time.Second
	defaultGatewayProbeTimeout     = 1500 * time.Millisecond
	defaultGatewayUsageCostDays    = 30
	defaultGatewayPort             = 18789
)

const dnsSDReachedAtMarker = " can be reached at "

const (
	gatewayEndpointCandidateRejectedMissingAddress    = "missing_address"
	gatewayEndpointCandidateRejectedInvalidPort       = "invalid_port"
	gatewayEndpointCandidateRejectedUnsupportedScheme = "unsupported_scheme"
	gatewayEndpointCandidateRejectedDuplicate         = "duplicate_endpoint"
)

// GatewayEndpoint is the stable gateway discovery beacon model surfaced by
// gateway discover/probe. TXT values are retained as non-authoritative hints;
// Address and Port always represent the resolved endpoint used for routing.
type GatewayEndpoint struct {
	InstanceName string            `json:"instanceName,omitempty"`
	DisplayName  string            `json:"displayName,omitempty"`
	Address      string            `json:"address"`
	Port         int               `json:"port"`
	Scheme       string            `json:"scheme"`
	WSURL        string            `json:"wsUrl"`
	Source       string            `json:"source"`
	Domain       string            `json:"domain,omitempty"`
	TXT          map[string]string `json:"txt,omitempty"`
}

type GatewayDegradedStatus struct {
	Reason   string           `json:"reason"`
	Message  string           `json:"message,omitempty"`
	Endpoint *GatewayEndpoint `json:"endpoint,omitempty"`
}

type GatewayDiscoverResult struct {
	OK       bool                    `json:"ok"`
	Code     string                  `json:"code"`
	Count    int                     `json:"count"`
	Beacons  []GatewayEndpoint       `json:"beacons"`
	Degraded []GatewayDegradedStatus `json:"degraded,omitempty"`
}

type GatewayDiscoverRequest struct {
	Discoverer GatewayDiscoverer
	Endpoints  []GatewayEndpoint
}

type GatewayDiscoverer interface {
	DiscoverGateways(context.Context) ([]GatewayEndpoint, error)
}

type GatewayDiscovererFunc func(context.Context) ([]GatewayEndpoint, error)

func (f GatewayDiscovererFunc) DiscoverGateways(ctx context.Context) ([]GatewayEndpoint, error) {
	return f(ctx)
}

func DiscoverGateways(ctx context.Context, req GatewayDiscoverRequest) GatewayDiscoverResult {
	var (
		endpoints []GatewayEndpoint
		degraded  []GatewayDegradedStatus
	)
	if len(req.Endpoints) > 0 {
		endpoints = append(endpoints, req.Endpoints...)
	}
	if req.Discoverer != nil {
		discovered, err := req.Discoverer.DiscoverGateways(ctx)
		if err != nil {
			degraded = append(degraded, GatewayDegradedStatus{
				Reason:  GatewayDegradedBonjourUnavailable,
				Message: sanitizeGatewayError(err),
			})
		}
		endpoints = append(endpoints, discovered...)
	}
	endpoints = normalizeGatewayEndpoints(endpoints)
	if len(endpoints) == 0 {
		code := GatewayDiscoverCodeNoGateways
		if len(degraded) > 0 {
			code = GatewayDiscoverCodeUnavailable
		}
		degraded = append(degraded, GatewayDegradedStatus{
			Reason:  GatewayDegradedNoGateways,
			Message: "no gateway beacons were discovered",
		})
		return GatewayDiscoverResult{Code: code, Degraded: degraded}
	}
	return GatewayDiscoverResult{
		OK:       true,
		Code:     GatewayDiscoverCodeCompleted,
		Count:    len(endpoints),
		Beacons:  endpoints,
		Degraded: degraded,
	}
}

func NormalizeGatewayEndpoint(endpoint GatewayEndpoint) GatewayEndpoint {
	endpoint.InstanceName = strings.TrimSpace(endpoint.InstanceName)
	endpoint.DisplayName = strings.TrimSpace(endpoint.DisplayName)
	endpoint.Address = strings.TrimSpace(endpoint.Address)
	endpoint.Source = strings.TrimSpace(endpoint.Source)
	endpoint.Domain = strings.TrimSpace(endpoint.Domain)
	endpoint.Scheme = normalizeGatewaySchemeAlias(endpoint.Scheme)
	if endpoint.Source == "" {
		endpoint.Source = GatewayEndpointSourceBonjour
	}
	if endpoint.Scheme == "" {
		endpoint.Scheme = "ws"
	}
	endpoint = applyGatewayTXTMetadataHints(endpoint)
	if endpoint.Port == 0 {
		endpoint.Port = defaultGatewayPort
	}
	endpoint.WSURL = ""
	if wsURL, ok := gatewayEndpointWSURL(endpoint); ok {
		endpoint.WSURL = wsURL
	}
	return endpoint
}

func ParseGatewayEndpoint(raw string, source string) (GatewayEndpoint, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return GatewayEndpoint{}, errors.New("gateway endpoint is empty")
	}
	if !strings.Contains(trimmed, "://") {
		trimmed = "ws://" + trimmed
	}
	u, err := url.Parse(trimmed)
	if err != nil {
		return GatewayEndpoint{}, fmt.Errorf("parse gateway endpoint: %w", err)
	}
	address := strings.TrimSpace(u.Hostname())
	if address == "" {
		return GatewayEndpoint{}, errors.New("gateway endpoint host is empty")
	}
	scheme, err := parseGatewayEndpointScheme(u.Scheme)
	if err != nil {
		return GatewayEndpoint{}, err
	}
	port := defaultGatewayPort
	if rawPort := strings.TrimSpace(u.Port()); rawPort != "" {
		parsed, err := strconv.Atoi(rawPort)
		if err != nil || parsed <= 0 || parsed > 65535 {
			return GatewayEndpoint{}, fmt.Errorf("invalid gateway endpoint port %q", rawPort)
		}
		port = parsed
	}
	return NormalizeGatewayEndpoint(GatewayEndpoint{
		Address: address,
		Port:    port,
		Scheme:  scheme,
		Source:  strings.TrimSpace(source),
	}), nil
}

func applyGatewayTXTMetadataHints(endpoint GatewayEndpoint) GatewayEndpoint {
	if endpoint.TXT == nil {
		return endpoint
	}
	endpoint.TXT = cleanGatewayTXT(endpoint.TXT)
	if endpoint.TXT == nil {
		return endpoint
	}
	if endpoint.DisplayName == "" {
		endpoint.DisplayName = strings.TrimSpace(endpoint.TXT["displayName"])
	}
	// TXT records are unauthenticated metadata. Keep routing host/port sourced
	// from the resolved SRV/manual endpoint; only gatewayTls is applied because
	// mDNS SRV has no TLS bit and callers still need the correct ws/wss scheme.
	if endpoint.Scheme == "ws" && truthyGatewayTXT(endpoint.TXT["gatewayTls"]) {
		endpoint.Scheme = "wss"
	}
	return endpoint
}

func normalizeGatewaySchemeAlias(raw string) string {
	switch scheme := strings.ToLower(strings.TrimSpace(raw)); scheme {
	case "", "ws", "http":
		return "ws"
	case "wss", "https":
		return "wss"
	default:
		return scheme
	}
}

func parseGatewayEndpointScheme(raw string) (string, error) {
	scheme := normalizeGatewaySchemeAlias(raw)
	if supportedGatewayEndpointScheme(scheme) {
		return scheme, nil
	}
	return "", fmt.Errorf("unsupported gateway endpoint scheme %q", raw)
}

func supportedGatewayEndpointScheme(scheme string) bool {
	return scheme == "ws" || scheme == "wss"
}

func gatewayEndpointWSURL(endpoint GatewayEndpoint) (string, bool) {
	address := strings.TrimSpace(endpoint.Address)
	if address == "" || !validGatewayEndpointPort(endpoint.Port) || !supportedGatewayEndpointScheme(endpoint.Scheme) {
		return "", false
	}
	return endpoint.Scheme + "://" + net.JoinHostPort(address, strconv.Itoa(endpoint.Port)), true
}

func normalizeGatewayEndpoints(in []GatewayEndpoint) []GatewayEndpoint {
	flow := evaluateGatewayEndpointCandidates(in)
	out := append([]GatewayEndpoint(nil), flow.Accepted...)
	sort.Slice(out, func(i, j int) bool {
		return lessGatewayEndpoint(out[i], out[j])
	})
	return out
}

type gatewayEndpointCandidateFlow struct {
	Accepted []GatewayEndpoint
	Rejected []gatewayEndpointCandidate
}

func evaluateGatewayEndpointCandidates(in []GatewayEndpoint) gatewayEndpointCandidateFlow {
	flow := gatewayEndpointCandidateFlow{
		Accepted: make([]GatewayEndpoint, 0, len(in)),
	}
	seen := map[string]bool{}
	for _, endpoint := range in {
		candidate := classifyGatewayEndpointCandidate(NormalizeGatewayEndpoint(endpoint), seen)
		if !candidate.Accepted {
			flow.Rejected = append(flow.Rejected, candidate)
			continue
		}
		seen[candidate.Key] = true
		flow.Accepted = append(flow.Accepted, candidate.Endpoint)
	}
	return flow
}

type gatewayEndpointCandidate struct {
	Endpoint  GatewayEndpoint
	Key       string
	Rejection string
	Accepted  bool
}

func classifyGatewayEndpointCandidate(endpoint GatewayEndpoint, seen map[string]bool) gatewayEndpointCandidate {
	key, rejection, ok := gatewayEndpointCandidateKey(endpoint)
	if !ok {
		return gatewayEndpointCandidate{Endpoint: endpoint, Rejection: rejection}
	}
	if seen[key] {
		return gatewayEndpointCandidate{Endpoint: endpoint, Key: key, Rejection: gatewayEndpointCandidateRejectedDuplicate}
	}
	return gatewayEndpointCandidate{Endpoint: endpoint, Key: key, Accepted: true}
}

func gatewayEndpointCandidateKey(endpoint GatewayEndpoint) (string, string, bool) {
	address := canonicalGatewayEndpointKeyAddress(endpoint.Address)
	if address == "" {
		return "", gatewayEndpointCandidateRejectedMissingAddress, false
	}
	if !validGatewayEndpointPort(endpoint.Port) {
		return "", gatewayEndpointCandidateRejectedInvalidPort, false
	}
	if !supportedGatewayEndpointScheme(endpoint.Scheme) {
		return "", gatewayEndpointCandidateRejectedUnsupportedScheme, false
	}
	return endpoint.Scheme + "://" + address + ":" + strconv.Itoa(endpoint.Port), "", true
}

func validGatewayEndpointPort(port int) bool {
	return port > 0 && port <= 65535
}

func canonicalGatewayEndpointKeyAddress(address string) string {
	return strings.ToLower(strings.TrimSuffix(strings.TrimSpace(address), "."))
}

func lessGatewayEndpoint(a, b GatewayEndpoint) bool {
	if a.Source != b.Source {
		return a.Source < b.Source
	}
	if a.Address != b.Address {
		return a.Address < b.Address
	}
	return a.Port < b.Port
}

func cleanGatewayTXT(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		key := strings.TrimSpace(k)
		if key == "" {
			continue
		}
		out[key] = strings.TrimSpace(v)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func truthyGatewayTXT(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func parseGatewayPort(raw string) int {
	port, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || port <= 0 || port > 65535 {
		return 0
	}
	return port
}

// ShellGatewayDiscoverer uses installed platform discovery helpers when they
// exist. It never treats unauthenticated TXT records as authoritative routing:
// parsed SRV/A results populate Address and Port; TXT is only retained as hint
// metadata for operators.
type ShellGatewayDiscoverer struct {
	Timeout  time.Duration
	LookPath func(string) (string, error)
	Run      func(context.Context, string, ...string) (string, error)
}

func NewShellGatewayDiscoverer(timeout time.Duration) ShellGatewayDiscoverer {
	if timeout <= 0 {
		timeout = defaultGatewayDiscoveryTimeout
	}
	return ShellGatewayDiscoverer{Timeout: timeout}
}

func (d ShellGatewayDiscoverer) DiscoverGateways(ctx context.Context) ([]GatewayEndpoint, error) {
	lookPath := d.LookPath
	if lookPath == nil {
		lookPath = exec.LookPath
	}
	run := d.Run
	if run == nil {
		run = runGatewayDiscoveryCommand
	}
	timeout := d.Timeout
	if timeout <= 0 {
		timeout = defaultGatewayDiscoveryTimeout
	}
	if _, err := lookPath("avahi-browse"); err == nil {
		runCtx, cancel := context.WithTimeout(ctx, timeout)
		out, runErr := run(runCtx, "avahi-browse", "-rpt", GatewayBonjourServiceType)
		runTimedOut := errors.Is(runCtx.Err(), context.DeadlineExceeded)
		cancel()
		endpoints := parseAvahiBrowseGateways(out)
		if len(endpoints) > 0 {
			return endpoints, nil
		}
		if runErr != nil && !runTimedOut {
			return nil, runErr
		}
	}
	if _, err := lookPath("dns-sd"); err == nil {
		runCtx, cancel := context.WithTimeout(ctx, timeout)
		out, runErr := run(runCtx, "dns-sd", "-B", GatewayBonjourServiceType, "local.")
		runTimedOut := errors.Is(runCtx.Err(), context.DeadlineExceeded)
		cancel()
		instances := parseDNSSDBrowseInstances(out)
		endpoints := make([]GatewayEndpoint, 0, len(instances))
		for _, instance := range instances {
			resolveCtx, resolveCancel := context.WithTimeout(ctx, timeout)
			resolveOut, _ := run(resolveCtx, "dns-sd", "-L", instance, GatewayBonjourServiceType, "local.")
			resolveCancel()
			endpoints = append(endpoints, parseDNSSDResolveGateway(resolveOut, instance)...)
		}
		if len(endpoints) > 0 {
			return endpoints, nil
		}
		if runErr != nil && !runTimedOut {
			return nil, runErr
		}
	}
	return nil, nil
}

func runGatewayDiscoveryCommand(ctx context.Context, name string, args ...string) (string, error) {
	raw, err := exec.CommandContext(ctx, name, args...).CombinedOutput()
	return string(raw), err
}

func parseAvahiBrowseGateways(stdout string) []GatewayEndpoint {
	var endpoints []GatewayEndpoint
	for _, raw := range strings.Split(stdout, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || !strings.HasPrefix(line, "=") {
			continue
		}
		parts := strings.Split(line, ";")
		if len(parts) < 9 {
			continue
		}
		port := parseGatewayPort(parts[8])
		if port == 0 {
			continue
		}
		txt := parseGatewayTXTTokens(parts[9:])
		endpoints = append(endpoints, GatewayEndpoint{
			InstanceName: unescapeAvahiField(parts[3]),
			Address:      strings.TrimSpace(parts[7]),
			Port:         port,
			Source:       GatewayEndpointSourceBonjour,
			Domain:       strings.TrimSpace(parts[5]),
			TXT:          txt,
		})
	}
	return normalizeGatewayEndpoints(endpoints)
}

func parseDNSSDBrowseInstances(stdout string) []string {
	seen := map[string]bool{}
	var instances []string
	for _, raw := range strings.Split(stdout, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.Contains(line, "DATE:") || strings.Contains(line, "Browsing for") {
			continue
		}
		if !strings.Contains(line, GatewayBonjourServiceType) {
			continue
		}
		idx := strings.Index(line, GatewayBonjourServiceType)
		instance := strings.TrimSpace(line[idx+len(GatewayBonjourServiceType):])
		instance = strings.TrimPrefix(instance, ".")
		instance = strings.TrimSpace(instance)
		if instance == "" {
			fields := strings.Fields(line)
			if len(fields) == 0 {
				continue
			}
			instance = fields[len(fields)-1]
		}
		instance = decodeDNSSDText(instance)
		if instance == "" || seen[instance] {
			continue
		}
		seen[instance] = true
		instances = append(instances, instance)
	}
	sort.Strings(instances)
	return instances
}

func parseDNSSDResolveGateway(stdout string, instance string) []GatewayEndpoint {
	var (
		address string
		port    int
		txt     = map[string]string{}
	)
	for _, raw := range strings.Split(stdout, "\n") {
		line := strings.TrimSpace(raw)
		if endpoint, ok := parseDNSSDReachedAtEndpoint(line); ok {
			address, port = endpoint.Address, endpoint.Port
			continue
		}
		for key, value := range parseGatewayTXTLine(line) {
			txt[key] = value
		}
	}
	if address == "" || port == 0 {
		return nil
	}
	return []GatewayEndpoint{NormalizeGatewayEndpoint(GatewayEndpoint{
		InstanceName: decodeDNSSDText(instance),
		Address:      address,
		Port:         port,
		Source:       GatewayEndpointSourceBonjour,
		Domain:       "local.",
		TXT:          txt,
	})}
}

func parseDNSSDReachedAtEndpoint(line string) (GatewayEndpoint, bool) {
	idx := strings.Index(line, dnsSDReachedAtMarker)
	if idx < 0 {
		return GatewayEndpoint{}, false
	}
	after := line[idx+len(dnsSDReachedAtMarker):]
	fields := strings.Fields(after)
	if len(fields) == 0 {
		return GatewayEndpoint{}, false
	}
	hostPort := strings.TrimSuffix(strings.TrimSpace(fields[0]), ".")
	host, port, ok := splitGatewayHostPort(hostPort)
	if !ok {
		return GatewayEndpoint{}, false
	}
	return GatewayEndpoint{Address: host, Port: port}, true
}

func splitGatewayHostPort(raw string) (string, int, bool) {
	if host, portRaw, err := net.SplitHostPort(raw); err == nil {
		port := parseGatewayPort(portRaw)
		return strings.TrimSuffix(host, "."), port, port > 0
	}
	return splitGatewayHostPortFallback(raw)
}

func splitGatewayHostPortFallback(raw string) (string, int, bool) {
	idx := strings.LastIndex(raw, ":")
	if idx <= 0 || idx == len(raw)-1 || strings.Count(raw, ":") != 1 {
		return "", 0, false
	}
	port := parseGatewayPort(raw[idx+1:])
	if port == 0 {
		return "", 0, false
	}
	return strings.TrimSuffix(raw[:idx], "."), port, true
}

func parseGatewayTXTLine(line string) map[string]string {
	return parseGatewayTXTTokens(splitGatewayTXTFields(line))
}

func splitGatewayTXTFields(line string) []string {
	var fields []string
	var b strings.Builder
	inQuote := false
	escaped := false
	flush := func() {
		if b.Len() == 0 {
			return
		}
		fields = append(fields, b.String())
		b.Reset()
	}
	for _, r := range strings.TrimSpace(line) {
		switch {
		case escaped:
			b.WriteRune(r)
			escaped = false
		case r == '\\' && inQuote:
			b.WriteRune(r)
			escaped = true
		case r == '"':
			inQuote = !inQuote
		case unicode.IsSpace(r) && !inQuote:
			flush()
		default:
			b.WriteRune(r)
		}
	}
	flush()
	return fields
}

func parseGatewayTXTTokens(tokens []string) map[string]string {
	txt := map[string]string{}
	for _, token := range tokens {
		cleaned := strings.Trim(strings.TrimSpace(token), `"`)
		if cleaned == "" {
			continue
		}
		if key, value, ok := strings.Cut(cleaned, "="); ok {
			key = decodeDNSSDText(key)
			if key != "" {
				txt[key] = decodeDNSSDText(value)
			}
		}
	}
	if len(txt) == 0 {
		return nil
	}
	return txt
}

func unescapeAvahiField(value string) string {
	return decodeDNSSDText(value)
}

func decodeDNSSDText(value string) string {
	trimmed := strings.TrimSpace(value)
	if !strings.Contains(trimmed, `\`) {
		return trimmed
	}
	var b strings.Builder
	for i := 0; i < len(trimmed); {
		if trimmed[i] == '\\' && i+3 < len(trimmed) && isDNSSDDecimalDigit(trimmed[i+1]) && isDNSSDDecimalDigit(trimmed[i+2]) && isDNSSDDecimalDigit(trimmed[i+3]) {
			decoded, err := strconv.ParseUint(trimmed[i+1:i+4], 10, 8)
			if err == nil {
				b.WriteByte(byte(decoded))
				i += 4
				continue
			}
		}
		b.WriteByte(trimmed[i])
		i++
	}
	return b.String()
}

func isDNSSDDecimalDigit(b byte) bool {
	return b >= '0' && b <= '9'
}

type GatewayRuntimeSummary struct {
	State          string            `json:"state,omitempty"`
	Validation     string            `json:"validation,omitempty"`
	ValidationLive bool              `json:"validationLive"`
	ActiveAgents   int               `json:"activeAgents"`
	Platforms      map[string]string `json:"platforms,omitempty"`
	Missing        bool              `json:"missing,omitempty"`
}

type GatewayProbeRequest struct {
	Discoverer GatewayDiscoverer
	Endpoints  []GatewayEndpoint
	Prober     GatewayEndpointProber
	Runtime    GatewayRuntimeSummary
}

type GatewayDiscoverySummary struct {
	Count    int                     `json:"count"`
	Beacons  []GatewayEndpoint       `json:"beacons"`
	Degraded []GatewayDegradedStatus `json:"degraded,omitempty"`
}

type GatewayProbeResult struct {
	OK        bool                    `json:"ok"`
	Degraded  bool                    `json:"degraded"`
	Code      string                  `json:"code"`
	Discovery GatewayDiscoverySummary `json:"discovery"`
	Runtime   GatewayRuntimeSummary   `json:"runtime"`
	Targets   []GatewayProbeTarget    `json:"targets"`
	Warnings  []GatewayDegradedStatus `json:"warnings,omitempty"`
}

type GatewayProbeTarget struct {
	Endpoint     GatewayEndpoint             `json:"endpoint"`
	Reachable    bool                        `json:"reachable"`
	Health       string                      `json:"health"`
	Status       string                      `json:"status"`
	LatencyMs    int64                       `json:"latencyMs,omitempty"`
	Error        string                      `json:"error,omitempty"`
	Capabilities *GatewayCapabilitiesSummary `json:"capabilities,omitempty"`
}

type GatewayEndpointProber interface {
	ProbeGatewayEndpoint(context.Context, GatewayEndpoint) GatewayProbeTarget
}

type GatewayHTTPAuth struct {
	Token  string
	Source string
}

type GatewayCapabilitiesSummary struct {
	Object       string   `json:"object,omitempty"`
	Platform     string   `json:"platform,omitempty"`
	Model        string   `json:"model,omitempty"`
	AuthType     string   `json:"authType,omitempty"`
	AuthRequired bool     `json:"authRequired"`
	AuthSource   string   `json:"authSource,omitempty"`
	StatusCode   int      `json:"statusCode,omitempty"`
	Features     []string `json:"features,omitempty"`
	Endpoints    []string `json:"endpoints,omitempty"`
}

type GatewayEndpointProberFunc func(context.Context, GatewayEndpoint) GatewayProbeTarget

func (f GatewayEndpointProberFunc) ProbeGatewayEndpoint(ctx context.Context, endpoint GatewayEndpoint) GatewayProbeTarget {
	return f(ctx, endpoint)
}

func ProbeGateways(ctx context.Context, req GatewayProbeRequest) GatewayProbeResult {
	discovery := GatewayDiscoverResult{Code: GatewayDiscoverCodeNoGateways}
	if req.Discoverer != nil {
		discovery = DiscoverGateways(ctx, GatewayDiscoverRequest{Discoverer: req.Discoverer})
	}
	endpoints := normalizeGatewayEndpoints(append(append([]GatewayEndpoint(nil), req.Endpoints...), discovery.Beacons...))
	warnings := append([]GatewayDegradedStatus(nil), discovery.Degraded...)
	if len(endpoints) == 0 {
		if len(warnings) == 0 {
			warnings = append(warnings, GatewayDegradedStatus{
				Reason:  GatewayDegradedNoGateways,
				Message: "no gateway targets were available to probe",
			})
		}
		return GatewayProbeResult{
			Code: GatewayProbeCodeUnreachable,
			Discovery: GatewayDiscoverySummary{
				Degraded: warnings,
			},
			Runtime:  req.Runtime,
			Warnings: warnings,
			Degraded: true,
		}
	}

	prober := req.Prober
	if prober == nil {
		prober = TCPGatewayProber{Timeout: defaultGatewayProbeTimeout}
	}
	targets := make([]GatewayProbeTarget, 0, len(endpoints))
	reachable := false
	for _, endpoint := range endpoints {
		target := prober.ProbeGatewayEndpoint(ctx, endpoint)
		target.Endpoint = NormalizeGatewayEndpoint(target.Endpoint)
		if target.Endpoint.Address == "" {
			target.Endpoint = endpoint
		}
		if target.Health == "" {
			if target.Reachable {
				target.Health = GatewayHealthTCPReachable
			} else {
				target.Health = GatewayHealthUnreachable
			}
		}
		if target.Status == "" {
			target.Status = gatewayProbeStatusFromRuntime(target.Reachable, req.Runtime)
		}
		if gatewayProbeTargetOK(target) {
			reachable = true
		} else if target.Error != "" {
			endpoint := target.Endpoint
			warnings = append(warnings, GatewayDegradedStatus{
				Reason:   gatewayProbeWarningReason(target),
				Message:  target.Error,
				Endpoint: &endpoint,
			})
		}
		targets = append(targets, target)
	}
	if !reachable {
		return GatewayProbeResult{
			Code:      GatewayProbeCodeUnreachable,
			Discovery: GatewayDiscoverySummary{Count: discovery.Count, Beacons: discovery.Beacons, Degraded: discovery.Degraded},
			Runtime:   req.Runtime,
			Targets:   targets,
			Warnings:  warnings,
			Degraded:  true,
		}
	}
	return GatewayProbeResult{
		OK:        true,
		Code:      GatewayProbeCodeCompleted,
		Discovery: GatewayDiscoverySummary{Count: discovery.Count, Beacons: discovery.Beacons, Degraded: discovery.Degraded},
		Runtime:   req.Runtime,
		Targets:   targets,
		Warnings:  warnings,
		Degraded:  len(warnings) > 0,
	}
}

func gatewayProbeTargetOK(target GatewayProbeTarget) bool {
	if !target.Reachable {
		return false
	}
	switch target.Health {
	case GatewayHealthHTTPUnauthorized, GatewayHealthHTTPCapabilityUnsupported, GatewayHealthHTTPCapabilityMalformed, GatewayHealthHTTPUnavailable:
		return false
	default:
		return true
	}
}

func gatewayProbeWarningReason(target GatewayProbeTarget) string {
	switch target.Health {
	case GatewayHealthHTTPUnauthorized:
		return GatewayDegradedHTTPUnauthorized
	case GatewayHealthHTTPCapabilityUnsupported:
		return GatewayDegradedCapabilityUnsupported
	case GatewayHealthHTTPCapabilityMalformed:
		return GatewayDegradedCapabilityMalformed
	case GatewayHealthHTTPUnavailable:
		return GatewayDegradedHTTPProbeUnavailable
	default:
		return GatewayDegradedProbeTimeout
	}
}

func gatewayProbeStatusFromRuntime(reachable bool, runtime GatewayRuntimeSummary) string {
	if !reachable {
		return GatewayProbeStatusUnavailable
	}
	state := strings.ToLower(strings.TrimSpace(runtime.State))
	if state == "" {
		return "runtime_unknown"
	}
	return "runtime_" + state
}

type TCPGatewayProber struct {
	Timeout     time.Duration
	DialContext func(context.Context, string, string) (net.Conn, error)
	Now         func() time.Time
}

func (p TCPGatewayProber) ProbeGatewayEndpoint(ctx context.Context, endpoint GatewayEndpoint) GatewayProbeTarget {
	endpoint = NormalizeGatewayEndpoint(endpoint)
	timeout := p.Timeout
	if timeout <= 0 {
		timeout = defaultGatewayProbeTimeout
	}
	dial := p.DialContext
	if dial == nil {
		d := net.Dialer{Timeout: timeout}
		dial = d.DialContext
	}
	now := p.Now
	if now == nil {
		now = time.Now
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	started := now()
	conn, err := dial(ctx, "tcp", net.JoinHostPort(endpoint.Address, strconv.Itoa(endpoint.Port)))
	latency := now().Sub(started).Milliseconds()
	if err != nil {
		return GatewayProbeTarget{
			Endpoint:  endpoint,
			Reachable: false,
			Health:    GatewayHealthUnreachable,
			Status:    GatewayProbeStatusUnavailable,
			LatencyMs: latency,
			Error:     sanitizeGatewayError(err),
		}
	}
	_ = conn.Close()
	return GatewayProbeTarget{
		Endpoint:  endpoint,
		Reachable: true,
		Health:    GatewayHealthTCPReachable,
		LatencyMs: latency,
	}
}

type HTTPGatewayProber struct {
	Timeout time.Duration
	Client  *http.Client
	Auth    GatewayHTTPAuth
	Now     func() time.Time
}

func (p HTTPGatewayProber) ProbeGatewayEndpoint(ctx context.Context, endpoint GatewayEndpoint) GatewayProbeTarget {
	endpoint = NormalizeGatewayEndpoint(endpoint)
	timeout := p.Timeout
	if timeout <= 0 {
		timeout = defaultGatewayProbeTimeout
	}
	client := p.Client
	if client == nil {
		client = &http.Client{Timeout: timeout}
	}
	now := p.Now
	if now == nil {
		now = time.Now
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	started := now()

	baseURL := gatewayHTTPBaseURL(endpoint)
	healthStatus, _, err := p.get(ctx, client, baseURL+"/health", false)
	if err != nil {
		return p.httpTarget(endpoint, false, GatewayHealthHTTPUnavailable, GatewayProbeStatusUnavailable, now().Sub(started).Milliseconds(), 0, "health probe: "+p.redact(err.Error()), nil)
	}
	if healthStatus < 200 || healthStatus >= 300 {
		return p.httpTarget(endpoint, false, GatewayHealthHTTPUnavailable, GatewayProbeStatusUnavailable, now().Sub(started).Milliseconds(), healthStatus, fmt.Sprintf("health status=%d", healthStatus), nil)
	}

	detailedStatus, _, err := p.get(ctx, client, baseURL+"/health/detailed", false)
	if err != nil {
		return p.httpTarget(endpoint, false, GatewayHealthHTTPUnavailable, GatewayProbeStatusUnavailable, now().Sub(started).Milliseconds(), 0, "health_detailed probe: "+p.redact(err.Error()), nil)
	}
	if detailedStatus < 200 || detailedStatus >= 300 {
		return p.httpTarget(endpoint, false, GatewayHealthHTTPUnavailable, GatewayProbeStatusUnavailable, now().Sub(started).Milliseconds(), detailedStatus, fmt.Sprintf("health_detailed status=%d", detailedStatus), nil)
	}

	capStatus, capBody, err := p.get(ctx, client, baseURL+"/v1/capabilities", true)
	authSource := p.authSource()
	if err != nil {
		summary := &GatewayCapabilitiesSummary{StatusCode: capStatus, AuthSource: authSource}
		return p.httpTarget(endpoint, false, GatewayHealthHTTPUnavailable, GatewayProbeStatusUnavailable, now().Sub(started).Milliseconds(), capStatus, "capabilities probe: "+p.redact(err.Error()), summary)
	}
	if capStatus == http.StatusUnauthorized || capStatus == http.StatusForbidden {
		summary := &GatewayCapabilitiesSummary{StatusCode: capStatus, AuthSource: authSource}
		return p.httpTarget(endpoint, true, GatewayHealthHTTPUnauthorized, GatewayProbeStatusUnauthorized, now().Sub(started).Milliseconds(), capStatus, fmt.Sprintf("capabilities status=%d auth_source=%s", capStatus, authSource), summary)
	}
	if capStatus < 200 || capStatus >= 300 {
		summary := &GatewayCapabilitiesSummary{StatusCode: capStatus, AuthSource: authSource}
		return p.httpTarget(endpoint, true, GatewayHealthHTTPCapabilityUnsupported, GatewayProbeStatusUnsupportedCapability, now().Sub(started).Milliseconds(), capStatus, fmt.Sprintf("capabilities status=%d auth_source=%s", capStatus, authSource), summary)
	}

	summary, err := summarizeGatewayCapabilities(capBody, capStatus, authSource)
	if err != nil {
		return p.httpTarget(endpoint, true, GatewayHealthHTTPCapabilityMalformed, GatewayProbeStatusMalformedCapabilities, now().Sub(started).Milliseconds(), capStatus, "capabilities malformed: "+p.redact(err.Error()), summary)
	}
	if !gatewayCapabilitiesSupported(summary) {
		return p.httpTarget(endpoint, true, GatewayHealthHTTPCapabilityUnsupported, GatewayProbeStatusUnsupportedCapability, now().Sub(started).Milliseconds(), capStatus, "capabilities missing required Hermes API-server features", summary)
	}
	return p.httpTarget(endpoint, true, GatewayHealthHTTPHealthy, GatewayProbeStatusCapabilityReady, now().Sub(started).Milliseconds(), capStatus, "", summary)
}

func (p HTTPGatewayProber) get(ctx context.Context, client *http.Client, url string, authenticated bool) (int, []byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, nil, err
	}
	if authenticated && strings.TrimSpace(p.Auth.Token) != "" {
		req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(p.Auth.Token))
	}
	resp, err := client.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return resp.StatusCode, nil, err
	}
	return resp.StatusCode, body, nil
}

func (p HTTPGatewayProber) httpTarget(endpoint GatewayEndpoint, reachable bool, health string, status string, latency int64, statusCode int, message string, capabilities *GatewayCapabilitiesSummary) GatewayProbeTarget {
	if capabilities != nil && capabilities.StatusCode == 0 {
		capabilities.StatusCode = statusCode
	}
	if capabilities != nil && capabilities.AuthSource == "" {
		capabilities.AuthSource = p.authSource()
	}
	return GatewayProbeTarget{
		Endpoint:     endpoint,
		Reachable:    reachable,
		Health:       health,
		Status:       status,
		LatencyMs:    latency,
		Error:        p.redact(message),
		Capabilities: capabilities,
	}
}

func (p HTTPGatewayProber) authSource() string {
	source := strings.TrimSpace(p.Auth.Source)
	if source == "" {
		return "none"
	}
	return source
}

func (p HTTPGatewayProber) redact(message string) string {
	return redactGatewayProbeText(message, p.Auth.Token)
}

func gatewayHTTPBaseURL(endpoint GatewayEndpoint) string {
	scheme := "http"
	if endpoint.Scheme == "wss" {
		scheme = "https"
	}
	return scheme + "://" + net.JoinHostPort(endpoint.Address, strconv.Itoa(endpoint.Port))
}

type gatewayCapabilitiesPayload struct {
	Object    string                               `json:"object"`
	Platform  string                               `json:"platform"`
	Model     string                               `json:"model"`
	Auth      gatewayCapAuth                       `json:"auth"`
	Features  map[string]any                       `json:"features"`
	Endpoints map[string]gatewayCapabilityEndpoint `json:"endpoints"`
}

type gatewayCapAuth struct {
	Type     string `json:"type"`
	Required bool   `json:"required"`
}

func summarizeGatewayCapabilities(body []byte, statusCode int, authSource string) (*GatewayCapabilitiesSummary, error) {
	var payload gatewayCapabilitiesPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		return &GatewayCapabilitiesSummary{StatusCode: statusCode, AuthSource: authSource}, err
	}
	features := enabledGatewayFeatureNames(payload.Features)
	endpoints := routableGatewayEndpointNames(payload.Endpoints)
	return &GatewayCapabilitiesSummary{
		Object:       strings.TrimSpace(payload.Object),
		Platform:     strings.TrimSpace(payload.Platform),
		Model:        strings.TrimSpace(payload.Model),
		AuthType:     normalizeGatewayCapabilityAuthType(payload.Auth.Type),
		AuthRequired: payload.Auth.Required,
		AuthSource:   authSource,
		StatusCode:   statusCode,
		Features:     features,
		Endpoints:    endpoints,
	}, nil
}

func enabledGatewayFeatureNames(features map[string]any) []string {
	names := make([]string, 0, len(features))
	for name, value := range features {
		if enabled, ok := value.(bool); ok && enabled {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return names
}

type gatewayCapabilityEndpoint struct {
	Method string `json:"method"`
	Path   string `json:"path"`
}

func routableGatewayEndpointNames(endpoints map[string]gatewayCapabilityEndpoint) []string {
	names := make([]string, 0, len(endpoints))
	for name, endpoint := range endpoints {
		if strings.TrimSpace(endpoint.Method) != "" && strings.TrimSpace(endpoint.Path) != "" {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return names
}

func gatewayCapabilitiesSupported(summary *GatewayCapabilitiesSummary) bool {
	if summary == nil {
		return false
	}
	if summary.Object != "hermes.api_server.capabilities" {
		return false
	}
	if normalizeGatewayCapabilityAuthType(summary.AuthType) != "bearer" {
		return false
	}
	return gatewayCapabilityNamesInclude(summary.Features, requiredGatewayCapabilityFeatures()) &&
		gatewayCapabilityNamesInclude(summary.Endpoints, requiredGatewayCapabilityEndpoints())
}

func normalizeGatewayCapabilityAuthType(raw string) string {
	return strings.ToLower(strings.TrimSpace(raw))
}

func requiredGatewayCapabilityFeatures() []string {
	return []string{
		"chat_completions",
		"responses_api",
		"run_submission",
		"run_status",
		"run_events_sse",
		"run_stop",
		"tool_progress_events",
	}
}

func requiredGatewayCapabilityEndpoints() []string {
	return []string{
		"health",
		"health_detailed",
		"chat_completions",
		"runs",
		"run_status",
		"run_events",
		"run_stop",
	}
}

func gatewayCapabilityNamesInclude(have []string, required []string) bool {
	for _, name := range required {
		if !gatewayProbeStringSliceContains(have, name) {
			return false
		}
	}
	return true
}

func gatewayProbeStringSliceContains(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}

type GatewayUsageSession struct {
	SessionID string    `json:"sessionId"`
	Source    string    `json:"source,omitempty"`
	ChatID    string    `json:"chatId,omitempty"`
	Title     string    `json:"title,omitempty"`
	UpdatedAt time.Time `json:"updatedAt,omitempty"`
	TokensIn  int       `json:"tokensIn"`
	TokensOut int       `json:"tokensOut"`
}

type GatewayUsagePricing struct {
	Source             string  `json:"source"`
	InputCostPer1KUSD  float64 `json:"inputCostPer1kUsd"`
	OutputCostPer1KUSD float64 `json:"outputCostPer1kUsd"`
}

type GatewayUsageCostRequest struct {
	Days     int
	Now      func() time.Time
	Sessions []GatewayUsageSession
	Lister   GatewayUsageSessionLister
	Pricing  GatewayUsagePricing
}

type GatewayUsageSessionLister interface {
	ListGatewayUsageSessions(context.Context) ([]GatewayUsageSession, error)
}

type GatewayUsageSessionListerFunc func(context.Context) ([]GatewayUsageSession, error)

func (f GatewayUsageSessionListerFunc) ListGatewayUsageSessions(ctx context.Context) ([]GatewayUsageSession, error) {
	return f(ctx)
}

type GatewayUsageCostSession struct {
	SessionID        string    `json:"sessionId"`
	Source           string    `json:"source,omitempty"`
	ChatID           string    `json:"chatId,omitempty"`
	Title            string    `json:"title,omitempty"`
	UpdatedAt        time.Time `json:"updatedAt,omitempty"`
	TokensIn         int       `json:"tokensIn"`
	TokensOut        int       `json:"tokensOut"`
	TotalTokens      int       `json:"totalTokens"`
	EstimatedCostUSD float64   `json:"estimatedCostUsd"`
	Priced           bool      `json:"priced"`
}

type GatewayUsageCostTotals struct {
	Sessions         int     `json:"sessions"`
	TokensIn         int     `json:"tokensIn"`
	TokensOut        int     `json:"tokensOut"`
	TotalTokens      int     `json:"totalTokens"`
	EstimatedCostUSD float64 `json:"estimatedCostUsd"`
	Priced           bool    `json:"priced"`
}

type GatewayUsageCostResult struct {
	OK       bool                      `json:"ok"`
	Code     string                    `json:"code"`
	Days     int                       `json:"days"`
	Since    time.Time                 `json:"since"`
	Pricing  GatewayUsagePricing       `json:"pricing"`
	Totals   GatewayUsageCostTotals    `json:"totals"`
	Sessions []GatewayUsageCostSession `json:"sessions"`
	Degraded []GatewayDegradedStatus   `json:"degraded,omitempty"`
}

func SummarizeGatewayUsageCost(ctx context.Context, req GatewayUsageCostRequest) GatewayUsageCostResult {
	days := req.Days
	if days <= 0 {
		days = defaultGatewayUsageCostDays
	}
	now := req.Now
	if now == nil {
		now = func() time.Time { return time.Now().UTC() }
	}
	pricing := normalizeGatewayUsagePricing(req.Pricing)
	sessions := append([]GatewayUsageSession(nil), req.Sessions...)
	anchor := now()
	since := gatewayUsageSince(anchor, days)
	if req.Lister != nil {
		listed, err := req.Lister.ListGatewayUsageSessions(ctx)
		if err != nil {
			return gatewayUsageUnavailable(days, since, pricing, sanitizeGatewayError(err))
		}
		sessions = append(sessions, listed...)
	}
	if err := ctx.Err(); err != nil {
		return gatewayUsageUnavailable(days, since, pricing, sanitizeGatewayError(err))
	}

	rows := make([]GatewayUsageCostSession, 0, len(sessions))
	for _, session := range sessions {
		session = normalizeGatewayUsageSession(session)
		if session.SessionID == "" {
			continue
		}
		if !session.UpdatedAt.IsZero() && session.UpdatedAt.Before(since) {
			continue
		}
		total, hasUsage := gatewayUsageTokenTotal(session.TokensIn, session.TokensOut)
		if !hasUsage {
			continue
		}
		cost, priced := estimateGatewayUsageCost(session.TokensIn, session.TokensOut, pricing)
		rows = append(rows, GatewayUsageCostSession{
			SessionID:        session.SessionID,
			Source:           session.Source,
			ChatID:           session.ChatID,
			Title:            session.Title,
			UpdatedAt:        session.UpdatedAt,
			TokensIn:         session.TokensIn,
			TokensOut:        session.TokensOut,
			TotalTokens:      total,
			EstimatedCostUSD: cost,
			Priced:           priced,
		})
	}
	sort.Slice(rows, func(i, j int) bool {
		if !rows[i].UpdatedAt.Equal(rows[j].UpdatedAt) {
			return rows[i].UpdatedAt.After(rows[j].UpdatedAt)
		}
		return rows[i].SessionID < rows[j].SessionID
	})
	if len(rows) == 0 {
		return gatewayUsageUnavailable(days, since, pricing, "no usage-bearing session metadata found")
	}
	result := GatewayUsageCostResult{
		OK:       true,
		Code:     GatewayUsageCostCodeCompleted,
		Days:     days,
		Since:    since,
		Pricing:  pricing,
		Sessions: rows,
	}
	for _, row := range rows {
		result.Totals.Sessions++
		result.Totals.TokensIn = addGatewayUsageTokens(result.Totals.TokensIn, row.TokensIn)
		result.Totals.TokensOut = addGatewayUsageTokens(result.Totals.TokensOut, row.TokensOut)
		result.Totals.TotalTokens = addGatewayUsageTokens(result.Totals.TotalTokens, row.TotalTokens)
		result.Totals.EstimatedCostUSD += row.EstimatedCostUSD
		result.Totals.Priced = result.Totals.Priced || row.Priced
	}
	return result
}

func gatewayUsageSince(anchor time.Time, days int) time.Time {
	return anchor.Add(-time.Duration(days) * 24 * time.Hour)
}

func normalizeGatewayUsagePricing(pricing GatewayUsagePricing) GatewayUsagePricing {
	pricing.Source = strings.TrimSpace(pricing.Source)
	if pricing.Source == "" {
		pricing.Source = "unpriced"
	}
	if pricing.InputCostPer1KUSD < 0 {
		pricing.InputCostPer1KUSD = 0
	}
	if pricing.OutputCostPer1KUSD < 0 {
		pricing.OutputCostPer1KUSD = 0
	}
	return pricing
}

func normalizeGatewayUsageSession(session GatewayUsageSession) GatewayUsageSession {
	session.SessionID = strings.TrimSpace(session.SessionID)
	session.Source = strings.TrimSpace(session.Source)
	session.ChatID = strings.TrimSpace(session.ChatID)
	session.Title = strings.TrimSpace(session.Title)
	if session.TokensIn < 0 {
		session.TokensIn = 0
	}
	if session.TokensOut < 0 {
		session.TokensOut = 0
	}
	return session
}

func gatewayUsageTokenTotal(tokensIn, tokensOut int) (int, bool) {
	total := addGatewayUsageTokens(tokensIn, tokensOut)
	return total, total > 0
}

func addGatewayUsageTokens(a, b int) int {
	if a < 0 {
		a = 0
	}
	if b < 0 {
		b = 0
	}
	maxInt := int(^uint(0) >> 1)
	if a > maxInt-b {
		return maxInt
	}
	return a + b
}

func estimateGatewayUsageCost(tokensIn, tokensOut int, pricing GatewayUsagePricing) (float64, bool) {
	priced := pricing.InputCostPer1KUSD > 0 || pricing.OutputCostPer1KUSD > 0
	if !priced {
		return 0, false
	}
	return float64(tokensIn)*pricing.InputCostPer1KUSD/1000 + float64(tokensOut)*pricing.OutputCostPer1KUSD/1000, true
}

func gatewayUsageUnavailable(days int, since time.Time, pricing GatewayUsagePricing, message string) GatewayUsageCostResult {
	return GatewayUsageCostResult{
		Code:    GatewayUsageCostCodeUnavailable,
		Days:    days,
		Since:   since,
		Pricing: pricing,
		Degraded: []GatewayDegradedStatus{{
			Reason:  GatewayDegradedUsageDataUnavailable,
			Message: message,
		}},
	}
}

func sanitizeGatewayError(err error) string {
	if err == nil {
		return ""
	}
	msg := strings.TrimSpace(err.Error())
	if msg == "" {
		return "unknown error"
	}
	return msg
}

var (
	gatewayProbeBearerPattern     = regexp.MustCompile(`(?i)\bbearer\s+["']?[^"'\s]+`)
	gatewayProbeSecretPairPattern = regexp.MustCompile(`(?i)\b(api[_-]?key|token|password|secret)=([^&\s"']+)`)
)

func redactGatewayProbeText(text string, secrets ...string) string {
	redacted := strings.TrimSpace(text)
	if redacted == "" {
		return ""
	}
	for _, secret := range secrets {
		secret = strings.TrimSpace(secret)
		if secret == "" {
			continue
		}
		redacted = strings.ReplaceAll(redacted, secret, "[redacted]")
	}
	redacted = gatewayProbeBearerPattern.ReplaceAllString(redacted, "Bearer [redacted]")
	redacted = gatewayProbeSecretPairPattern.ReplaceAllString(redacted, "$1=[redacted]")
	return redacted
}
