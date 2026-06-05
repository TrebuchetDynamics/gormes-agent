package spawncmd

import "testing"

func TestParseNameAndPersona(t *testing.T) {
	cmd, err := Parse("/spawn Research literature reviewer")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if cmd.Name != "Research" || cmd.Persona != "literature reviewer" {
		t.Fatalf("parsed spawn = %+v, want name Research and persona literature reviewer", cmd)
	}
}

func TestParseRejectsMissingName(t *testing.T) {
	if _, err := Parse("/spawn"); err != ErrUsage {
		t.Fatalf("Parse(/spawn) err = %v, want ErrUsage", err)
	}
}
