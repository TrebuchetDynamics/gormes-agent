package navigation

import "testing"

func TestIsPrivateBrowserHost(t *testing.T) {
	tests := []struct {
		name string
		host string
		want bool
	}{
		{name: "localhost", host: "localhost", want: true},
		{name: "localhost_with_trailing_dot", host: "LOCALHOST.", want: true},
		{name: "ipv4_loopback", host: "127.0.0.1", want: true},
		{name: "rfc1918_10", host: "10.20.30.40", want: true},
		{name: "rfc1918_172_lower_bound", host: "172.16.0.1", want: true},
		{name: "rfc1918_172_upper_bound", host: "172.31.255.254", want: true},
		{name: "rfc1918_192", host: "192.168.1.50", want: true},
		{name: "ipv4_link_local", host: "169.254.1.10", want: true},
		{name: "ipv6_loopback", host: "::1", want: true},
		{name: "ipv6_unique_local", host: "fd12:3456:789a::1", want: true},
		{name: "ipv6_link_local", host: "fe80::1", want: true},
		{name: "ipv6_link_local_with_zone", host: "fe80::1%25en0", want: true},
		{name: "mdns_local_suffix", host: "raspberrypi.local", want: true},
		{name: "lan_suffix", host: "printer.lan", want: true},
		{name: "internal_suffix", host: "db.internal", want: true},
		{name: "public_hostname", host: "github.com", want: false},
		{name: "public_ip_literal", host: "8.8.8.8", want: false},
		{name: "outside_172_private_lower", host: "172.15.255.255", want: false},
		{name: "outside_172_private_upper", host: "172.32.0.1", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsPrivateBrowserHost(tt.host); got != tt.want {
				t.Fatalf("IsPrivateBrowserHost(%q) = %v, want %v", tt.host, got, tt.want)
			}
		})
	}
}

func TestClassifyBrowserHost_ExposesDecisionInputs(t *testing.T) {
	private := classifyBrowserHost("[fe80::1%25en0]")
	if private.hostname != "fe80::1%25en0" {
		t.Fatalf("hostname = %q, want zone-preserving IPv6 literal", private.hostname)
	}
	if private.localName {
		t.Fatalf("localName = true, want address classification for IPv6 literal")
	}
	if !private.hasAddr || !isPrivateBrowserAddr(private.addr) {
		t.Fatalf("candidate = %#v, want parsed private address", private)
	}

	public := classifyBrowserHost("github.com")
	if public.hostname != "github.com" || public.localName || public.hasAddr {
		t.Fatalf("public candidate = %#v, want DNS name without local/private classification", public)
	}
}

func TestRouteNavigation_PrivateHostsUseLocalSidecar(t *testing.T) {
	tests := []struct {
		name    string
		taskID  string
		rawURL  string
		wantKey string
	}{
		{name: "localhost_default_task", rawURL: "http://localhost:3000/", wantKey: "default::local"},
		{name: "schemeless_localhost_with_port", rawURL: "localhost:3000", wantKey: "default::local"},
		{name: "loopback_ipv4", taskID: "task-1", rawURL: "http://127.0.0.1:8080/", wantKey: "task-1::local"},
		{name: "rfc1918_10", taskID: "task-1", rawURL: "http://10.2.3.4/", wantKey: "task-1::local"},
		{name: "rfc1918_172_lower_bound", taskID: "task-1", rawURL: "http://172.16.0.10/", wantKey: "task-1::local"},
		{name: "rfc1918_172_upper_bound", taskID: "task-1", rawURL: "http://172.31.255.250/", wantKey: "task-1::local"},
		{name: "rfc1918_192", taskID: "task-1", rawURL: "http://192.168.1.50:8000/", wantKey: "task-1::local"},
		{name: "ipv4_link_local", taskID: "task-1", rawURL: "http://169.254.10.20/", wantKey: "task-1::local"},
		{name: "ipv6_loopback", taskID: "task-1", rawURL: "http://[::1]:3000/", wantKey: "task-1::local"},
		{name: "ipv6_unique_local", taskID: "task-1", rawURL: "http://[fd12:3456:789a::1]/", wantKey: "task-1::local"},
		{name: "ipv6_link_local", taskID: "task-1", rawURL: "http://[fe80::1]/", wantKey: "task-1::local"},
		{name: "ipv6_link_local_with_zone", taskID: "task-1", rawURL: "http://[fe80::1%25en0]/", wantKey: "task-1::local"},
		{name: "local_suffix", taskID: "task-1", rawURL: "http://raspberrypi.local/", wantKey: "task-1::local"},
		{name: "lan_suffix", taskID: "task-1", rawURL: "http://printer.lan/", wantKey: "task-1::local"},
		{name: "internal_suffix", taskID: "task-1", rawURL: "http://db.internal/", wantKey: "task-1::local"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RouteNavigation(tt.taskID, tt.rawURL, true, true, false, false)
			want := Route{
				SessionKey: tt.wantKey,
				ForceLocal: true,
				Reason:     "private_url_local_sidecar",
			}
			if got != want {
				t.Fatalf("RouteNavigation(%q, %q) = %#v, want %#v", tt.taskID, tt.rawURL, got, want)
			}
		})
	}
}

func TestRouteNavigation_PublicURLsUseCloudKey(t *testing.T) {
	tests := []struct {
		name   string
		rawURL string
	}{
		{name: "public_hostname", rawURL: "https://github.com/x/y"},
		{name: "public_ip_literal", rawURL: "https://8.8.8.8/"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RouteNavigation("task-2", tt.rawURL, true, true, false, false)
			want := Route{SessionKey: "task-2"}
			if got != want {
				t.Fatalf("RouteNavigation(%q) = %#v, want %#v", tt.rawURL, got, want)
			}
		})
	}
}

func TestRouteNavigation_DisabledOrOverrideCases(t *testing.T) {
	tests := []struct {
		name                    string
		cloudConfigured         bool
		autoLocalForPrivateURLs bool
		cdpOverride             bool
		camofoxMode             bool
	}{
		{name: "no_cloud_provider", cloudConfigured: false, autoLocalForPrivateURLs: true},
		{name: "auto_local_disabled", cloudConfigured: true, autoLocalForPrivateURLs: false},
		{name: "cdp_override", cloudConfigured: true, autoLocalForPrivateURLs: true, cdpOverride: true},
		{name: "camofox_mode", cloudConfigured: true, autoLocalForPrivateURLs: true, camofoxMode: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RouteNavigation(
				"task-3",
				"http://localhost:3000/",
				tt.cloudConfigured,
				tt.autoLocalForPrivateURLs,
				tt.cdpOverride,
				tt.camofoxMode,
			)
			want := Route{SessionKey: "task-3"}
			if got != want {
				t.Fatalf("RouteNavigation(%s) = %#v, want %#v", tt.name, got, want)
			}
		})
	}
}

func TestRouteNavigation_DefaultTaskID(t *testing.T) {
	tests := []struct {
		name    string
		rawURL  string
		wantKey string
	}{
		{name: "public_url_uses_default", rawURL: "https://github.com/", wantKey: "default"},
		{name: "private_url_uses_default_local_sidecar", rawURL: "http://localhost:3000/", wantKey: "default::local"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RouteNavigation("", tt.rawURL, true, true, false, false)
			want := Route{SessionKey: tt.wantKey}
			if tt.wantKey == "default::local" {
				want.ForceLocal = true
				want.Reason = "private_url_local_sidecar"
			}
			if got != want {
				t.Fatalf("RouteNavigation(empty task, %q) = %#v, want %#v", tt.rawURL, got, want)
			}
		})
	}
}
