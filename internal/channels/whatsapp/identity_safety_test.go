package whatsapp

import "testing"

func TestWhatsAppIdentifierSafetyPredicate_AllowsASCIIJIDForms(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want string
	}{
		{name: "phone digits", raw: "15551234567", want: "15551234567"},
		{name: "plus prefixed phone", raw: "+15551234567", want: "15551234567"},
		{name: "phone jid", raw: "15551234567@s.whatsapp.net", want: "15551234567"},
		{name: "phone device jid", raw: "15551234567:47@s.whatsapp.net", want: "15551234567"},
		{name: "lid like bare id", raw: "999999999999999", want: "999999999999999"},
		{name: "lid jid", raw: "999999999999999@lid", want: "999999999999999"},
		{name: "group jid", raw: "120363012345678901@g.us", want: "120363012345678901"},
		{name: "trimmed jid", raw: " 15551234567@s.whatsapp.net ", want: "15551234567"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, safe, evidence := NormalizeSafeWhatsAppIdentifier(tt.raw)
			if !safe {
				t.Fatalf("NormalizeSafeWhatsAppIdentifier(%q) safe = false, evidence = %q, want safe", tt.raw, evidence)
			}
			if evidence != "" {
				t.Fatalf("NormalizeSafeWhatsAppIdentifier(%q) evidence = %q, want empty", tt.raw, evidence)
			}
			if got != tt.want {
				t.Fatalf("NormalizeSafeWhatsAppIdentifier(%q) = %q, want %q", tt.raw, got, tt.want)
			}

			gotAgain, safeAgain, evidenceAgain := NormalizeSafeWhatsAppIdentifier(tt.raw)
			if gotAgain != got || safeAgain != safe || evidenceAgain != evidence {
				t.Fatalf("NormalizeSafeWhatsAppIdentifier(%q) not deterministic: first (%q, %v, %q), second (%q, %v, %q)",
					tt.raw, got, safe, evidence, gotAgain, safeAgain, evidenceAgain)
			}
		})
	}
}

func TestWhatsAppIdentifierSafetyPredicate_RejectsPathLikeValues(t *testing.T) {
	tests := []string{
		"",
		" ",
		"\t \n",
		"15551234567/../../secret",
		"15551234567\\..\\secret",
		"..",
		"../15551234567",
		"..\\15551234567",
		"15551234567..backup",
		"/tmp/15551234567",
		"C:\\tmp\\15551234567",
		"\\\\server\\share\\15551234567",
		"%2f15551234567",
		"15551234567%2Fsecret",
		"%5c15551234567",
		"15551234567%5Csecret",
	}

	for _, raw := range tests {
		t.Run(raw, func(t *testing.T) {
			got, safe, evidence := NormalizeSafeWhatsAppIdentifier(raw)
			if safe {
				t.Fatalf("NormalizeSafeWhatsAppIdentifier(%q) safe = true, got %q", raw, got)
			}
			if got != "" {
				t.Fatalf("NormalizeSafeWhatsAppIdentifier(%q) normalized = %q, want empty", raw, got)
			}
			if evidence != WhatsAppIdentifierUnsafeEvidence {
				t.Fatalf("NormalizeSafeWhatsAppIdentifier(%q) evidence = %q, want %q", raw, evidence, WhatsAppIdentifierUnsafeEvidence)
			}
		})
	}
}

func TestWhatsAppIdentifierSafetyPredicate_RejectsNonASCIILookalikes(t *testing.T) {
	tests := []string{
		"１５５５１２３４５６７",
		"１２３４５@s.whatsapp.net",
		"١٢٣٤٥@s.whatsapp.net",
		"15551234567＠s.whatsapp.net",
		"15551234567∕s.whatsapp.net",
		"álîçé@s.whatsapp.net",
		"пользователь@s.whatsapp.net",
		"用户@s.whatsapp.net",
	}

	for _, raw := range tests {
		t.Run(raw, func(t *testing.T) {
			got, safe, evidence := NormalizeSafeWhatsAppIdentifier(raw)
			if safe {
				t.Fatalf("NormalizeSafeWhatsAppIdentifier(%q) safe = true, got %q", raw, got)
			}
			if got != "" {
				t.Fatalf("NormalizeSafeWhatsAppIdentifier(%q) normalized = %q, want empty", raw, got)
			}
			if evidence != WhatsAppIdentifierUnsafeEvidence {
				t.Fatalf("NormalizeSafeWhatsAppIdentifier(%q) evidence = %q, want %q", raw, evidence, WhatsAppIdentifierUnsafeEvidence)
			}
		})
	}
}
