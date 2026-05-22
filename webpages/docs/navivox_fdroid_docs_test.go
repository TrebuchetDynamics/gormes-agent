package docs_test

import (
	"strings"
	"testing"
)

func TestNavivoxFDroidDocsDescribeChannelStrategy(t *testing.T) {
	page := readDoc(t, "content/operate/navivox-fdroid.md")

	assertContainsAll(t, "content/operate/navivox-fdroid.md", page, []string{
		"## Channel strategy",
		"| Channel | Role | Use when |",
		"| F-Droid | Primary trust channel |",
		"| Google Play | Later reach channel |",
		"| Direct APK or GitHub release | Test and fallback channel |",
		"Do not treat Google Play as the first proof that Navivox belongs on Android.",
		"F-Droid is the audience-fit proof: open-source Android users can inspect, build,\nand install it without a Google account.",
	})

	for _, forbidden := range []string{
		"Available on F-Droid",
		"Install from F-Droid",
	} {
		if strings.Contains(page, forbidden) {
			t.Fatalf("F-Droid docs must not claim availability before release evidence; found %q", forbidden)
		}
	}
}

func TestNavivoxFDroidDocsCarryListingCopyGuardrails(t *testing.T) {
	page := readDoc(t, "content/operate/navivox-fdroid.md")

	assertContainsAll(t, "content/operate/navivox-fdroid.md", page, []string{
		"## Listing copy guardrails",
		"Short summary: Navivox is an open-source Android companion for a user-owned Gormes runtime.",
		"Audience: Termux users, self-hosters, AI tinkerers, privacy-first users, and OSS Android users.",
		"Promise: pair the app to your own Gormes gateway over HTTP/WebSocket; do not imply a hosted SaaS assistant.",
		"Avoid saying: one-tap install, no setup required, hosted cloud assistant, or available on F-Droid.",
	})
}

func TestNavivoxFDroidDocsExplainWhyGooglePlayIsNotFirstProof(t *testing.T) {
	page := readDoc(t, "content/operate/navivox-fdroid.md")

	assertContainsAll(t, "content/operate/navivox-fdroid.md", page, []string{
		"## Why not start with Google Play",
		"Google Play is useful for reach later, but it is a poor first proof point for Navivox.",
		"Store policy review can mistake self-hosted gateway setup for an unfinished app.",
		"Privacy-first users may distrust a Google-account-only install path.",
		"Commercial-store screenshots and copy reward broad consumer positioning, not developer-tool clarity.",
		"F-Droid lets the first audience validate source, builds, permissions, and gateway pairing before a mass-market listing.",
	})
}

func TestNavivoxFDroidDocsNameAudienceFitMatrix(t *testing.T) {
	page := readDoc(t, "content/operate/navivox-fdroid.md")

	assertContainsAll(t, "content/operate/navivox-fdroid.md", page, []string{
		"## Audience fit matrix",
		"| Audience | Why F-Droid fits |",
		"| Termux users | Already treat Android like a Linux workstation and expect package-manager-style installs. |",
		"| Self-hosters | Prefer companion apps that pair to their own services instead of opaque hosted accounts. |",
		"| AI tinkerers | Want inspectable local tooling before trusting an assistant on a phone. |",
		"| Privacy-first users | Avoid Google-account-only distribution and look for explicit permission boundaries. |",
		"| OSS Android users | Discover tools through source-available, reproducible, community-reviewed channels. |",
	})
}

func TestNavivoxFDroidDocsCarryPermissionAndPrivacyGuardrails(t *testing.T) {
	page := readDoc(t, "content/operate/navivox-fdroid.md")

	assertContainsAll(t, "content/operate/navivox-fdroid.md", page, []string{
		"## Permission and privacy guardrails",
		"Ask for only the permissions the pairing flow needs.",
		"Explain network access as communication with a local or user-configured Gormes gateway.",
		"Do not bundle analytics SDKs, proprietary crash reporters, ad networks, or account trackers.",
		"Document any broad-looking permission before release review so F-Droid users can audit the tradeoff.",
		"Screenshots must not show real gateway URLs, pairing tokens, chat payloads, workspace paths, or provider names.",
	})
}

func TestNavivoxFDroidDocsCarryMetadataReadinessChecklist(t *testing.T) {
	page := readDoc(t, "content/operate/navivox-fdroid.md")

	assertContainsAll(t, "content/operate/navivox-fdroid.md", page, []string{
		"## Metadata readiness checklist",
		"Keep the listing title, summary, and description focused on a user-owned Gormes companion, not a generic chatbot.",
		"Record the source repository, license, app ID, release tag, and build commit beside the listing copy.",
		"Prepare icon, screenshots, feature graphic, and alt text that show setup with placeholder endpoints only.",
		"Name Termux, self-hosting, privacy, local AI, and OSS Android in the long description so the right users recognize themselves.",
		"Keep Google Play language out of the F-Droid metadata except as a later reach channel in internal strategy notes.",
	})
}
