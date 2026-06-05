package rendering

import (
	"strings"
	"testing"
)

func TestFormatFinalTelegramText_RewritesMarkdownTablesAsBulletRows(t *testing.T) {
	input := "Scores:\n\n| Player | Score |\n|--------|-------|\n| Alice  | 150   |\n| Bob    | 120   |\n\nEnd."

	got := FormatFinalTelegramText(input)

	for _, want := range []string{
		`*Alice*`,
		`• Player: Alice`,
		`• Score: 150`,
		`*Bob*`,
		`• Score: 120`,
		`Scores:`,
		`End\.`,
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("FormatFinalTelegramText missing %q in:\n%s", want, got)
		}
	}
	if strings.Contains(got, `\|`) || strings.Contains(got, "```") {
		t.Fatalf("FormatFinalTelegramText left table syntax in:\n%s", got)
	}
}

func TestFormatFinalTelegramText_TableRowLabelDoesNotDuplicateAsBullet(t *testing.T) {
	input := "Daily results:\n\n| Pass | Fail |\n|------|------|\n| 2026-05-09 | good | bad |\n\nEnd."

	got := FormatFinalTelegramText(input)

	for _, want := range []string{
		`*2026\-05\-09*`,
		`• Pass: good`,
		`• Fail: bad`,
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("FormatFinalTelegramText missing %q in:\n%s", want, got)
		}
	}
	for _, forbidden := range []string{
		`• Pass: 2026\-05\-09`,
		`• Fail: good`,
	} {
		if strings.Contains(got, forbidden) {
			t.Fatalf("FormatFinalTelegramText duplicated row-label cell as bullet %q in:\n%s", forbidden, got)
		}
	}
}

func TestFormatFinalTelegramText_DoesNotRewriteFencedTables(t *testing.T) {
	input := "```\n| A | B |\n|---|---|\n| 1 | 2 |\n```"

	got := FormatFinalTelegramText(input)

	for _, want := range []string{
		"```\n",
		`| A | B |`,
		`|---|---|`,
		`| 1 | 2 |`,
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("FormatFinalTelegramText missing fenced table %q in:\n%s", want, got)
		}
	}
	if strings.Contains(got, `• A: 1`) {
		t.Fatalf("FormatFinalTelegramText rewrote fenced table:\n%s", got)
	}
}

func TestFormatFinalTelegramText_ConvertsStrikeSpoilerAndBlockquote(t *testing.T) {
	input := "~~deleted!~~ ||hidden.value||\n> Quote (ok)!\n5 > 3"

	got := FormatFinalTelegramText(input)

	for _, want := range []string{
		`~deleted\!~`,
		`||hidden\.value||`,
		`> Quote \(ok\)\!`,
		`5 \> 3`,
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("FormatFinalTelegramText missing %q in:\n%s", want, got)
		}
	}
	for _, forbidden := range []string{`❯ Quote`, `\|\|hidden`, `\~\~deleted`} {
		if strings.Contains(got, forbidden) {
			t.Fatalf("FormatFinalTelegramText contains forbidden %q in:\n%s", forbidden, got)
		}
	}
}
