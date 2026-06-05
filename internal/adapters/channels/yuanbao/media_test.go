package yuanbao

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func loadMsgBody(t *testing.T, fixture string) []TIMElement {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("testdata", fixture))
	if err != nil {
		t.Fatalf("read fixture %s: %v", fixture, err)
	}
	var body []TIMElement
	if err := json.Unmarshal(raw, &body); err != nil {
		t.Fatalf("decode fixture %s: %v", fixture, err)
	}
	return body
}

func TestYuanbaoMedia_NormalizesImageFileVoiceFixtures(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		fixture     string
		wantKind    string
		wantURL     string
		wantSource  string
		wantSize    int64
		wantName    string
		wantMedia   string
		wantErrCode string
	}{
		{
			name:       "image prefers medium variant url",
			fixture:    "media_image.json",
			wantKind:   "image",
			wantURL:    "https://cos.example.com/medium/img-001.jpg",
			wantSource: "img-uuid-001",
			wantSize:   51200,
			wantMedia:  "image/jpeg",
		},
		{
			name:       "file preserves filename and size",
			fixture:    "media_file.json",
			wantKind:   "file",
			wantURL:    "https://cos.example.com/files/report.pdf",
			wantSource: "file-uuid-002",
			wantSize:   1048576,
			wantName:   "report.pdf",
			wantMedia:  "application/pdf",
		},
		{
			name:       "voice maps to voice kind",
			fixture:    "media_voice.json",
			wantKind:   "voice",
			wantURL:    "https://cos.example.com/voice/clip-003.amr",
			wantSource: "voice-uuid-003",
			wantSize:   32768,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			body := loadMsgBody(t, tc.fixture)
			atts, err := NormalizeMedia(body, MediaPolicy{MaxBytes: 50 * 1024 * 1024})
			if err != nil {
				t.Fatalf("NormalizeMedia: %v", err)
			}
			if len(atts) != 1 {
				t.Fatalf("attachments = %d, want 1: %#v", len(atts), atts)
			}
			a := atts[0]
			if a.Kind != tc.wantKind {
				t.Errorf("Kind = %q, want %q", a.Kind, tc.wantKind)
			}
			if a.URL != tc.wantURL {
				t.Errorf("URL = %q, want %q", a.URL, tc.wantURL)
			}
			if a.SourceID != tc.wantSource {
				t.Errorf("SourceID = %q, want %q", a.SourceID, tc.wantSource)
			}
			if a.Size != tc.wantSize {
				t.Errorf("Size = %d, want %d", a.Size, tc.wantSize)
			}
			if tc.wantName != "" && a.FileName != tc.wantName {
				t.Errorf("FileName = %q, want %q", a.FileName, tc.wantName)
			}
			if tc.wantMedia != "" && a.MediaType != tc.wantMedia {
				t.Errorf("MediaType = %q, want %q", a.MediaType, tc.wantMedia)
			}
			if a.Error != "" {
				t.Errorf("Error = %q, want empty", a.Error)
			}
		})
	}
}

func TestYuanbaoMedia_MissingBytesProducesDegradedEvidence(t *testing.T) {
	t.Parallel()

	body := loadMsgBody(t, "media_missing_url.json")
	atts, err := NormalizeMedia(body, MediaPolicy{MaxBytes: 50 * 1024 * 1024})
	if err != nil {
		t.Fatalf("NormalizeMedia: %v", err)
	}
	if len(atts) != 1 {
		t.Fatalf("attachments = %d, want 1", len(atts))
	}
	a := atts[0]
	if a.Kind != "image" {
		t.Errorf("Kind = %q, want image", a.Kind)
	}
	if a.URL != "" {
		t.Errorf("URL = %q, want empty", a.URL)
	}
	if a.SourceID == "" {
		t.Errorf("SourceID must be preserved for diagnostics")
	}
	if a.Error != DegradedMediaUnavailable {
		t.Errorf("Error = %q, want %q", a.Error, DegradedMediaUnavailable)
	}
}

func TestYuanbaoMedia_OversizedRejectedAndPreservesText(t *testing.T) {
	t.Parallel()

	body := loadMsgBody(t, "media_oversized.json")
	policy := MediaPolicy{MaxBytes: 5 * 1024 * 1024}

	atts, err := NormalizeMedia(body, policy)
	if err != nil {
		t.Fatalf("NormalizeMedia: %v", err)
	}
	if len(atts) != 1 {
		t.Fatalf("attachments = %d, want 1", len(atts))
	}
	a := atts[0]
	if a.Kind != "file" {
		t.Errorf("Kind = %q, want file", a.Kind)
	}
	if a.SourceID != "file-uuid-005" {
		t.Errorf("SourceID = %q, want file-uuid-005", a.SourceID)
	}
	if a.URL != "" {
		t.Errorf("URL must be cleared on oversized rejection, got %q", a.URL)
	}
	if a.Error != DegradedMediaUnavailable {
		t.Errorf("Error = %q, want %q", a.Error, DegradedMediaUnavailable)
	}

	text := ExtractTextBody(body)
	if text != "Big file incoming." {
		t.Errorf("text body = %q, want preserved %q", text, "Big file incoming.")
	}
}

func TestYuanbaoMedia_MalformedMetadataIsDegraded(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		raw  string
	}{
		{name: "image_info_array missing", raw: `[{"msg_type":"TIMImageElem","msg_content":{"uuid":"x"}}]`},
		{name: "wrong type for content", raw: `[{"msg_type":"TIMFileElem","msg_content":"not-an-object"}]`},
		{name: "file with negative size", raw: `[{"msg_type":"TIMFileElem","msg_content":{"uuid":"y","file_name":"a","file_size":-1,"url":"https://x"}}]`},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var body []TIMElement
			if err := json.Unmarshal([]byte(tc.raw), &body); err != nil {
				t.Fatalf("decode: %v", err)
			}
			atts, err := NormalizeMedia(body, MediaPolicy{MaxBytes: 50 * 1024 * 1024})
			if err != nil {
				t.Fatalf("NormalizeMedia returned error: %v", err)
			}
			if len(atts) != 1 {
				t.Fatalf("attachments = %d, want 1 degraded entry", len(atts))
			}
			if atts[0].Error != DegradedMediaUnavailable {
				t.Errorf("Error = %q, want %q", atts[0].Error, DegradedMediaUnavailable)
			}
		})
	}
}

func TestYuanbaoMedia_DegradedErrorIsTyped(t *testing.T) {
	t.Parallel()

	a := Attachment{Kind: "file", Error: DegradedMediaUnavailable}
	if !a.IsDegraded() {
		t.Errorf("IsDegraded() = false, want true")
	}

	deg := newDegraded(DegradedMediaUnavailable, "oversized")
	var typed *DegradedError
	if !errors.As(deg, &typed) {
		t.Fatalf("expected *DegradedError")
	}
	if typed.Code != DegradedMediaUnavailable {
		t.Errorf("Code = %q", typed.Code)
	}
}
