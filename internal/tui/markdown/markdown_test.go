package markdown

import "testing"

func TestRenderNumberedListsPreservesSourceMarkers(t *testing.T) {
	got := Render("3. third\n4) fourth", Styles{})
	want := "3. third\n4) fourth"
	if got != want {
		t.Fatalf("Render() numbered list = %q, want %q", got, want)
	}
}

func TestRenderPipeTableParsesMarkdownDivider(t *testing.T) {
	got := Render("| Name | Value |\n| --- | ---: |\n| Foo | 42 |", Styles{})
	want := "Name  Value\n────  ─────\nFoo   42   "
	if got != want {
		t.Fatalf("Render() table = %q, want %q", got, want)
	}
}

func TestRenderFencedCodeRequiresMatchingFence(t *testing.T) {
	got := Render("````\ninside\n```\nstill code\n````", Styles{})
	want := "inside\n```\nstill code"
	if got != want {
		t.Fatalf("Render() fenced code = %q, want %q", got, want)
	}
}
