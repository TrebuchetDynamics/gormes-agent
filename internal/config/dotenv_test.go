package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseDotenv_SimpleKeyValue(t *testing.T) {
	input := "FOO=bar\nBAZ=qux\n"
	got, err := parseDotenv(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parseDotenv: %v", err)
	}
	if got["FOO"] != "bar" || got["BAZ"] != "qux" {
		t.Errorf("got = %v, want FOO=bar BAZ=qux", got)
	}
}

func TestParseDotenv_CommentsAndBlankLines(t *testing.T) {
	input := "# top-level comment\n\nFOO=bar\n  # indented comment\nBAZ=qux\n"
	got, _ := parseDotenv(strings.NewReader(input))
	if len(got) != 2 {
		t.Errorf("len = %d, want 2 (FOO + BAZ only)", len(got))
	}
}

func TestParseDotenv_ExportPrefix(t *testing.T) {
	input := "export FOO=bar\nexport   BAZ=qux\n"
	got, _ := parseDotenv(strings.NewReader(input))
	if got["FOO"] != "bar" || got["BAZ"] != "qux" {
		t.Errorf("got = %v, want FOO=bar BAZ=qux (export stripped)", got)
	}
}

func TestParseDotenv_QuotedValues(t *testing.T) {
	input := `DOUBLE="hello world"
SINGLE='hello world'
EMPTY=""
WITH_HASH="value # still value"
`
	got, _ := parseDotenv(strings.NewReader(input))
	if got["DOUBLE"] != "hello world" {
		t.Errorf("DOUBLE = %q, want 'hello world'", got["DOUBLE"])
	}
	if got["SINGLE"] != "hello world" {
		t.Errorf("SINGLE = %q, want 'hello world'", got["SINGLE"])
	}
	if got["EMPTY"] != "" {
		t.Errorf("EMPTY = %q, want empty string", got["EMPTY"])
	}
	if got["WITH_HASH"] != "value # still value" {
		t.Errorf("WITH_HASH = %q, want literal '# still value' (quoted)", got["WITH_HASH"])
	}
}

func TestParseDotenv_EscapeSequencesInDoubleQuotes(t *testing.T) {
	input := `LINES="a\nb"` + "\n" + `TAB="a\tb"` + "\n" + `QUOTE="a\"b"` + "\n"
	got, _ := parseDotenv(strings.NewReader(input))
	if got["LINES"] != "a\nb" {
		t.Errorf("LINES = %q, want 'a\\nb' (escape expanded)", got["LINES"])
	}
	if got["TAB"] != "a\tb" {
		t.Errorf("TAB = %q, want 'a\\tb'", got["TAB"])
	}
	if got["QUOTE"] != `a"b` {
		t.Errorf("QUOTE = %q, want literal 'a\"b'", got["QUOTE"])
	}
}

func TestParseDotenv_SingleQuotedValuesAreLiteral(t *testing.T) {
	input := `RAW='a\nb'` + "\n"
	got, _ := parseDotenv(strings.NewReader(input))
	if got["RAW"] != `a\nb` {
		t.Errorf("RAW = %q, want literal 'a\\nb' (single-quoted = no escape)", got["RAW"])
	}
}

func TestParseDotenv_InvalidLinesSkipped(t *testing.T) {
	input := "FOO=bar\nnoEqualsSign\n=noKey\nBAZ=qux\n"
	got, _ := parseDotenv(strings.NewReader(input))
	if len(got) != 2 {
		t.Errorf("len = %d, want 2 (invalid lines skipped)", len(got))
	}
	if got["FOO"] != "bar" || got["BAZ"] != "qux" {
		t.Errorf("got = %v, want FOO+BAZ only", got)
	}
}

func TestParseDotenv_TrailingWhitespaceStrippedFromUnquoted(t *testing.T) {
	input := "FOO=bar   \nBAZ=qux\t\n"
	got, _ := parseDotenv(strings.NewReader(input))
	if got["FOO"] != "bar" || got["BAZ"] != "qux" {
		t.Errorf("got = %v, want trailing whitespace stripped", got)
	}
}

func TestLoadDotenvFiles_SetsUnsetEnvVars(t *testing.T) {
	cfgHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", cfgHome)
	t.Setenv("HERMES_HOME", "")

	cfgDir := filepath.Join(cfgHome, "gormes")
	_ = os.MkdirAll(cfgDir, 0o755)
	body := "GORMES_DOTENV_TEST_UNSET=from-file\n"
	if err := os.WriteFile(filepath.Join(cfgDir, ".env"), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	// Ensure the var is not set before loadDotenvFiles runs.
	_ = os.Unsetenv("GORMES_DOTENV_TEST_UNSET")

	loadDotenvFiles()

	if got := os.Getenv("GORMES_DOTENV_TEST_UNSET"); got != "from-file" {
		t.Errorf("env after load = %q, want 'from-file'", got)
	}
	// Cleanup.
	_ = os.Unsetenv("GORMES_DOTENV_TEST_UNSET")
}

func TestLoadDotenvFiles_DoesNotOverrideExistingEnv(t *testing.T) {
	cfgHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", cfgHome)
	t.Setenv("HERMES_HOME", "")

	cfgDir := filepath.Join(cfgHome, "gormes")
	_ = os.MkdirAll(cfgDir, 0o755)
	body := "GORMES_DOTENV_TEST_SET=from-file\n"
	if err := os.WriteFile(filepath.Join(cfgDir, ".env"), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	// Shell env already has it — must NOT be overridden.
	t.Setenv("GORMES_DOTENV_TEST_SET", "from-shell")

	loadDotenvFiles()

	if got := os.Getenv("GORMES_DOTENV_TEST_SET"); got != "from-shell" {
		t.Errorf("env after load = %q, want shell 'from-shell' to win", got)
	}
}

func TestLoadDotenvFiles_GormesBeatsHermes(t *testing.T) {
	cfgHome := t.TempDir()
	hermesHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", cfgHome)
	t.Setenv("HERMES_HOME", hermesHome)

	gormesDir := filepath.Join(cfgHome, "gormes")
	_ = os.MkdirAll(gormesDir, 0o755)
	_ = os.WriteFile(filepath.Join(gormesDir, ".env"),
		[]byte("GORMES_DOTENV_PREC_TEST=gormes\n"), 0o600)
	_ = os.WriteFile(filepath.Join(hermesHome, ".env"),
		[]byte("GORMES_DOTENV_PREC_TEST=hermes\n"), 0o600)

	_ = os.Unsetenv("GORMES_DOTENV_PREC_TEST")
	loadDotenvFiles()

	if got := os.Getenv("GORMES_DOTENV_PREC_TEST"); got != "gormes" {
		t.Errorf("precedence = %q, want 'gormes' to win over 'hermes'", got)
	}
	_ = os.Unsetenv("GORMES_DOTENV_PREC_TEST")
}

func TestLoadDotenvFiles_HermesFallsBackToDefaultHome(t *testing.T) {
	cfgHome := t.TempDir()
	fakeHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", cfgHome)
	t.Setenv("HERMES_HOME", "") // unset => fall back to ~/.hermes/.env
	t.Setenv("HOME", fakeHome)  // so ~/.hermes = fakeHome/.hermes

	hermesDir := filepath.Join(fakeHome, ".hermes")
	_ = os.MkdirAll(hermesDir, 0o755)
	_ = os.WriteFile(filepath.Join(hermesDir, ".env"),
		[]byte("GORMES_DOTENV_LEGACY_TEST=from-legacy\n"), 0o600)

	_ = os.Unsetenv("GORMES_DOTENV_LEGACY_TEST")
	loadDotenvFiles()

	if got := os.Getenv("GORMES_DOTENV_LEGACY_TEST"); got != "from-legacy" {
		t.Errorf("legacy ~/.hermes/.env load = %q, want 'from-legacy'", got)
	}
	_ = os.Unsetenv("GORMES_DOTENV_LEGACY_TEST")
}

func TestLoadDotenvFiles_MissingFilesAreSilentNoop(t *testing.T) {
	cfgHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", cfgHome)
	t.Setenv("HERMES_HOME", "")
	// No .env file exists. Must not error, must not panic.
	loadDotenvFiles() // expected: silent return
}

func TestOptionalEnvAnyReportsMissingAirtableCredentialWithoutSecret(t *testing.T) {
	t.Setenv("AIRTABLE_API_KEY", "")
	t.Setenv("AIRTABLE_PAT", "")

	got := CheckOptionalEnvAny("AIRTABLE_API_KEY", "AIRTABLE_PAT")

	if got.Available {
		t.Fatalf("Available = true, want false: %+v", got)
	}
	if got.PresentName != "" {
		t.Fatalf("PresentName = %q, want empty", got.PresentName)
	}
	if got.Evidence != "missing optional environment variable: AIRTABLE_API_KEY or AIRTABLE_PAT" {
		t.Fatalf("Evidence = %q, want missing AIRTABLE_API_KEY/AIRTABLE_PAT", got.Evidence)
	}
	if strings.Contains(got.Evidence, "pat_") || strings.Contains(got.Evidence, "key_") {
		t.Fatalf("Evidence leaked a credential-looking value: %q", got.Evidence)
	}
}

func TestOptionalEnvAnyLoadsDotenvAndRedactsPresentSecret(t *testing.T) {
	cfgHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", cfgHome)
	t.Setenv("HERMES_HOME", "")
	clearTestEnvVars(t, "AIRTABLE_API_KEY", "AIRTABLE_PAT")

	cfgDir := filepath.Join(cfgHome, "gormes")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(%q): %v", cfgDir, err)
	}
	secret := "pat_secret_from_dotenv"
	if err := os.WriteFile(filepath.Join(cfgDir, ".env"), []byte("AIRTABLE_PAT="+secret+"\n"), 0o600); err != nil {
		t.Fatalf("WriteFile(.env): %v", err)
	}

	got := CheckOptionalEnvAny("AIRTABLE_API_KEY", "AIRTABLE_PAT")

	if !got.Available {
		t.Fatalf("Available = false, want true: %+v", got)
	}
	if got.PresentName != "AIRTABLE_PAT" {
		t.Fatalf("PresentName = %q, want AIRTABLE_PAT", got.PresentName)
	}
	if got.Evidence != "optional environment variable available: AIRTABLE_PAT=[redacted]" {
		t.Fatalf("Evidence = %q, want redacted AIRTABLE_PAT evidence", got.Evidence)
	}
	if strings.Contains(got.Evidence, secret) {
		t.Fatalf("Evidence leaked secret: %q", got.Evidence)
	}
}

func clearTestEnvVars(t *testing.T, names ...string) {
	t.Helper()
	type previous struct {
		name  string
		value string
		ok    bool
	}
	old := make([]previous, 0, len(names))
	for _, name := range names {
		value, ok := os.LookupEnv(name)
		old = append(old, previous{name: name, value: value, ok: ok})
		if err := os.Unsetenv(name); err != nil {
			t.Fatalf("Unsetenv(%q): %v", name, err)
		}
	}
	t.Cleanup(func() {
		for _, entry := range old {
			if entry.ok {
				_ = os.Setenv(entry.name, entry.value)
			} else {
				_ = os.Unsetenv(entry.name)
			}
		}
	})
}
