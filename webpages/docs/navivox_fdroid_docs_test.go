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
