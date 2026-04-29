package yuanbao

import (
	"encoding/json"
	"testing"
)

func TestYuanbaoSticker_NormalizesStaticAndAnimatedStickers(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		fixture     string
		wantID      string
		wantPackID  string
		wantName    string
		wantFormats string
		wantErr     string
	}{
		{
			name:        "static png sticker",
			fixture:     "sticker_static.json",
			wantID:      "278",
			wantPackID:  "1003",
			wantName:    "六六六",
			wantFormats: "png",
			wantErr:     "",
		},
		{
			name:        "animated apng marked degraded",
			fixture:     "sticker_animated.json",
			wantID:      "999",
			wantPackID:  "1003",
			wantName:    "animated-cheer",
			wantFormats: "apng",
			wantErr:     DegradedStickerUnsupported,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			body := loadMsgBody(t, tc.fixture)
			stickers, err := NormalizeStickers(body)
			if err != nil {
				t.Fatalf("NormalizeStickers: %v", err)
			}
			if len(stickers) != 1 {
				t.Fatalf("stickers = %d, want 1", len(stickers))
			}
			s := stickers[0]
			if s.StickerID != tc.wantID {
				t.Errorf("StickerID = %q, want %q", s.StickerID, tc.wantID)
			}
			if s.PackageID != tc.wantPackID {
				t.Errorf("PackageID = %q, want %q", s.PackageID, tc.wantPackID)
			}
			if s.Name != tc.wantName {
				t.Errorf("Name = %q, want %q", s.Name, tc.wantName)
			}
			if s.Formats != tc.wantFormats {
				t.Errorf("Formats = %q, want %q", s.Formats, tc.wantFormats)
			}
			if s.Error != tc.wantErr {
				t.Errorf("Error = %q, want %q", s.Error, tc.wantErr)
			}
		})
	}
}

func TestYuanbaoSticker_MalformedDataPreservesEvent(t *testing.T) {
	t.Parallel()

	body := loadMsgBody(t, "sticker_malformed.json")
	stickers, err := NormalizeStickers(body)
	if err != nil {
		t.Fatalf("NormalizeStickers: %v", err)
	}
	if len(stickers) != 1 {
		t.Fatalf("stickers = %d, want 1 degraded entry", len(stickers))
	}
	if stickers[0].Error != DegradedStickerUnsupported {
		t.Errorf("Error = %q, want %q", stickers[0].Error, DegradedStickerUnsupported)
	}

	if text := ExtractTextBody(body); text != "Sticker with broken metadata." {
		t.Errorf("text = %q, want preserved", text)
	}
}

func TestYuanbaoSticker_AsAttachmentExposesIdentifier(t *testing.T) {
	t.Parallel()

	body := loadMsgBody(t, "sticker_static.json")
	stickers, err := NormalizeStickers(body)
	if err != nil {
		t.Fatalf("NormalizeStickers: %v", err)
	}
	a := stickers[0].AsAttachment()
	if a.Kind != "sticker" {
		t.Errorf("Kind = %q, want sticker", a.Kind)
	}
	if a.SourceID != "278" {
		t.Errorf("SourceID = %q, want sticker_id 278", a.SourceID)
	}
	if a.FileName != "六六六" {
		t.Errorf("FileName = %q, want sticker name", a.FileName)
	}
	if a.MediaType != "image/png" {
		t.Errorf("MediaType = %q, want image/png for png formats", a.MediaType)
	}
}

func TestYuanbaoSticker_NoFaceElementsReturnsEmpty(t *testing.T) {
	t.Parallel()

	raw := `[{"msg_type":"TIMTextElem","msg_content":{"text":"plain"}}]`
	var body []TIMElement
	if err := json.Unmarshal([]byte(raw), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	stickers, err := NormalizeStickers(body)
	if err != nil {
		t.Fatalf("NormalizeStickers: %v", err)
	}
	if len(stickers) != 0 {
		t.Errorf("stickers = %d, want 0", len(stickers))
	}
}
