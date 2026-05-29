// Package vpnhost enumerates active VPN interfaces (Tailscale, WireGuard,
// and tun-class OpenVPN/IPSec devices) on the local host. It is the shared
// source of truth for code that needs to bind a service to a VPN-only
// interface or describe a VPN-reachable URL to operators.
//
// All side effects (CLI subprocess calls) flow through a Lister so callers
// in tests can inject deterministic fakes.
package vpnhost

import (
	"context"
	"encoding/json"
	"os/exec"
	"strings"
)

// Kind identifies the VPN transport an interface belongs to.
type Kind string

const (
	KindTailscale Kind = "tailscale"
	KindWireGuard Kind = "wireguard"
	KindTunOther  Kind = "tun-other"
)

// Host describes one VPN interface and the addresses it currently exposes.
// Empty IPv4 or IPv6 means the interface has no address of that family.
type Host struct {
	Iface string
	Kind  Kind
	IPv4  string
	IPv6  string
}

// Lister enumerates VPN hosts. The zero value uses real CLIs (`tailscale`
// and `ip`); tests construct a Lister with their own seam functions.
type Lister struct {
	LookPath     func(string) (string, error)
	RunTailscale func(ctx context.Context, args ...string) ([]byte, error)
	RunIP        func(ctx context.Context, args ...string) ([]byte, error)
}

// List returns every active VPN host in priority order:
// tailscale → wireguard → tun-other. An empty slice with nil error means
// no VPN interface was detected.
func (l Lister) List(ctx context.Context) ([]Host, error) {
	l = l.withDefaults()
	var out []Host
	if h, ok := l.tailscale(ctx); ok {
		out = append(out, h)
	}
	out = append(out, l.wireguard(ctx)...)
	out = append(out, l.tunOther(ctx, ifaceSet(out))...)
	return out, nil
}

// List runs the default Lister against real CLIs.
func List(ctx context.Context) ([]Host, error) {
	return Lister{}.List(ctx)
}

func (l Lister) withDefaults() Lister {
	if l.LookPath == nil {
		l.LookPath = exec.LookPath
	}
	if l.RunTailscale == nil {
		l.RunTailscale = func(ctx context.Context, args ...string) ([]byte, error) {
			return exec.CommandContext(ctx, "tailscale", args...).Output()
		}
	}
	if l.RunIP == nil {
		l.RunIP = func(ctx context.Context, args ...string) ([]byte, error) {
			return exec.CommandContext(ctx, "ip", args...).Output()
		}
	}
	return l
}

func (l Lister) tailscale(ctx context.Context) (Host, bool) {
	if _, err := l.LookPath("tailscale"); err != nil {
		return Host{}, false
	}
	v4 := firstNonEmptyLine(mustRun(l.RunTailscale, ctx, "ip", "-4"))
	v6 := firstNonEmptyLine(mustRun(l.RunTailscale, ctx, "ip", "-6"))
	if v4 == "" && v6 == "" {
		return Host{}, false
	}
	return Host{
		Iface: "tailscale0",
		Kind:  KindTailscale,
		IPv4:  v4,
		IPv6:  v6,
	}, true
}

func (l Lister) wireguard(ctx context.Context) []Host {
	if _, err := l.LookPath("ip"); err != nil {
		return nil
	}
	out, err := l.RunIP(ctx, "-j", "-d", "link", "show", "type", "wireguard")
	if err != nil {
		return nil
	}
	var links []ipLink
	if err := json.Unmarshal(out, &links); err != nil {
		return nil
	}
	hosts := make([]Host, 0, len(links))
	for _, link := range links {
		if link.IfName == "" {
			continue
		}
		v4, v6 := l.ifaceAddrs(ctx, link.IfName)
		hosts = append(hosts, Host{
			Iface: link.IfName,
			Kind:  KindWireGuard,
			IPv4:  v4,
			IPv6:  v6,
		})
	}
	return hosts
}

func (l Lister) tunOther(ctx context.Context, exclude map[string]struct{}) []Host {
	if _, err := l.LookPath("ip"); err != nil {
		return nil
	}
	out, err := l.RunIP(ctx, "-j", "link", "show", "type", "tun")
	if err != nil {
		return nil
	}
	var links []ipLink
	if err := json.Unmarshal(out, &links); err != nil {
		return nil
	}
	hosts := make([]Host, 0, len(links))
	for _, link := range links {
		if link.IfName == "" {
			continue
		}
		if _, skip := exclude[link.IfName]; skip {
			continue
		}
		v4, v6 := l.ifaceAddrs(ctx, link.IfName)
		hosts = append(hosts, Host{
			Iface: link.IfName,
			Kind:  KindTunOther,
			IPv4:  v4,
			IPv6:  v6,
		})
	}
	return hosts
}

func (l Lister) ifaceAddrs(ctx context.Context, iface string) (v4, v6 string) {
	out, err := l.RunIP(ctx, "-j", "addr", "show", "dev", iface)
	if err != nil {
		return "", ""
	}
	var entries []ipAddrEntry
	if err := json.Unmarshal(out, &entries); err != nil {
		return "", ""
	}
	for _, entry := range entries {
		for _, addr := range entry.AddrInfo {
			switch addr.Family {
			case "inet":
				if v4 == "" {
					v4 = addr.Local
				}
			case "inet6":
				if v6 == "" {
					v6 = addr.Local
				}
			}
		}
	}
	return v4, v6
}

type ipLink struct {
	IfName string `json:"ifname"`
}

type ipAddrEntry struct {
	IfName   string       `json:"ifname"`
	AddrInfo []ipAddrInfo `json:"addr_info"`
}

type ipAddrInfo struct {
	Family string `json:"family"`
	Local  string `json:"local"`
}

func mustRun(fn func(context.Context, ...string) ([]byte, error), ctx context.Context, args ...string) []byte {
	out, err := fn(ctx, args...)
	if err != nil {
		return nil
	}
	return out
}

func firstNonEmptyLine(b []byte) string {
	for _, line := range strings.Split(string(b), "\n") {
		if s := strings.TrimSpace(line); s != "" {
			return s
		}
	}
	return ""
}

func ifaceSet(hosts []Host) map[string]struct{} {
	set := make(map[string]struct{}, len(hosts))
	for _, h := range hosts {
		if h.Iface != "" {
			set[h.Iface] = struct{}{}
		}
	}
	return set
}
