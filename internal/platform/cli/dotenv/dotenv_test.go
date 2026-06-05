package dotenv

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadValuesParsesSimpleQuotedAndCommentedLines(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".env")
	body := "\n# comment\nGORMES_API_KEY = 'sk-test'\nEMPTY=\"\"\nPLAIN=value\nMALFORMED\n =missing-key\nWHATSAPP_ALLOWED_USERS=  user1,user2  \n"
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	got := ReadValues(path)
	if got["GORMES_API_KEY"] != "sk-test" {
		t.Fatalf("GORMES_API_KEY = %q, want sk-test", got["GORMES_API_KEY"])
	}
	if got["EMPTY"] != "" {
		t.Fatalf("EMPTY = %q, want empty", got["EMPTY"])
	}
	if got["PLAIN"] != "value" {
		t.Fatalf("PLAIN = %q, want value", got["PLAIN"])
	}
	if got["WHATSAPP_ALLOWED_USERS"] != "user1,user2" {
		t.Fatalf("WHATSAPP_ALLOWED_USERS = %q, want user1,user2", got["WHATSAPP_ALLOWED_USERS"])
	}
	if _, ok := got["MALFORMED"]; ok {
		t.Fatalf("malformed line parsed: %+v", got)
	}
}

func TestReadValuesMissingFileReturnsEmptyMap(t *testing.T) {
	got := ReadValues(filepath.Join(t.TempDir(), "missing.env"))
	if got == nil || len(got) != 0 {
		t.Fatalf("ReadValues missing = %+v, want empty map", got)
	}
}
