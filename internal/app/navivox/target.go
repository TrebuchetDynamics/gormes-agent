package navivox

import (
	"context"
	"fmt"
	"net"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/network/vpnhost"
)

// Target describes the network endpoint chosen for a temporary Navivox pair bridge.
type Target struct {
	Host            string
	ExposureMode    string
	PublicConfirmed bool
	Source          string
}

// VPNHostLister returns VPN-visible host addresses for target selection.
type VPNHostLister func(context.Context) ([]vpnhost.Host, error)

// Resolve chooses the best Navivox pair target from an explicit host, VPN hosts,
// or the first non-container LAN IPv4 address.
func Resolve(ctx context.Context, requestedHost string, listVPNHosts VPNHostLister) (Target, error) {
	requestedHost = strings.TrimSpace(requestedHost)
	if requestedHost != "" {
		return Target{
			Host:            requestedHost,
			ExposureMode:    ExposureForHost(requestedHost),
			PublicConfirmed: !LoopbackHost(requestedHost),
			Source:          "operator override",
		}, nil
	}

	var hosts []vpnhost.Host
	if listVPNHosts != nil {
		hosts, _ = listVPNHosts(ctx)
	}
	for _, h := range hosts {
		if h.Kind != vpnhost.KindTailscale || strings.TrimSpace(h.IPv4) == "" {
			continue
		}
		return Target{Host: h.IPv4, ExposureMode: config.NavivoxExposureTailscale, Source: "tailscale auto-detected"}, nil
	}
	for _, h := range hosts {
		if strings.TrimSpace(h.IPv4) == "" {
			continue
		}
		switch h.Kind {
		case vpnhost.KindWireGuard:
			return Target{Host: h.IPv4, ExposureMode: config.NavivoxExposureWireGuard, Source: "wireguard auto-detected"}, nil
		case vpnhost.KindTunOther:
			return Target{Host: h.IPv4, ExposureMode: config.NavivoxExposureVPN, Source: "vpn auto-detected"}, nil
		}
	}
	if host := LANIPv4(); host != "" {
		return Target{Host: host, ExposureMode: config.NavivoxExposurePublic, PublicConfirmed: true, Source: "lan auto-detected"}, nil
	}
	return Target{}, fmt.Errorf("navivox pair: no network IP detected; connect to Tailscale/Wi-Fi or pass --host <network-ip>")
}

// ExposureForHost maps an explicit host to the least permissive Navivox exposure mode.
func ExposureForHost(host string) string {
	if LoopbackHost(host) {
		return config.NavivoxExposureLocal
	}
	return config.NavivoxExposurePublic
}

// LoopbackHost reports whether host is localhost or a loopback IP literal.
func LoopbackHost(host string) bool {
	clean := strings.Trim(strings.TrimSpace(host), "[]")
	if clean == "localhost" {
		return true
	}
	if ip := net.ParseIP(clean); ip != nil && ip.IsLoopback() {
		return true
	}
	return false
}

// LANIPv4 returns the first non-container, non-loopback LAN IPv4 address.
func LANIPv4() string {
	ifaces, err := net.Interfaces()
	if err != nil {
		return ""
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		name := strings.ToLower(iface.Name)
		if strings.HasPrefix(name, "docker") || strings.HasPrefix(name, "br-") || strings.HasPrefix(name, "veth") || strings.HasPrefix(name, "virbr") || strings.HasPrefix(name, "podman") {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			ip = ip.To4()
			if ip == nil || ip.IsLoopback() || ip.IsLinkLocalUnicast() {
				continue
			}
			return ip.String()
		}
	}
	return ""
}
