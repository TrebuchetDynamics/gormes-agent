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

func TestParseSanitizesC1ControlsInAgentName(t *testing.T) {
	cmd, err := Parse("/spawn Re\u009bsearch literature reviewer")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if cmd.Name != "Research" {
		t.Fatalf("Name = %q, want C1 control stripped", cmd.Name)
	}
	if cmd.Persona != "literature reviewer" {
		t.Fatalf("Persona = %q, want preserved", cmd.Persona)
	}
}

func TestParseSanitizesHiddenFormattingInAgentName(t *testing.T) {
	cmd, err := Parse("/spawn Re\u200bsearch literature reviewer")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if cmd.Name != "Research" {
		t.Fatalf("Name = %q, want hidden formatting stripped", cmd.Name)
	}
	if cmd.Persona != "literature reviewer" {
		t.Fatalf("Persona = %q, want preserved", cmd.Persona)
	}
}

func TestParseRejectsBareCommandWordWithoutSlash(t *testing.T) {
	if _, err := Parse("spawn Research literature reviewer"); err != ErrUsage {
		t.Fatalf("Parse bare command word err = %v, want ErrUsage", err)
	}
}

func TestParseRejectsMissingName(t *testing.T) {
	if _, err := Parse("/spawn"); err != ErrUsage {
		t.Fatalf("Parse(/spawn) err = %v, want ErrUsage", err)
	}
}
