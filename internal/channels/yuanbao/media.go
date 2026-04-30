package yuanbao

import (
	"encoding/json"
	"path"
	"strings"
)

// Degraded codes specific to media/sticker normalization.
const (
	DegradedMediaUnavailable   = "yuanbao_media_unavailable"
	DegradedStickerUnsupported = "yuanbao_sticker_unsupported"
)

// AttachmentKind values used by the Yuanbao normalizers.
const (
	AttachmentImage   = "image"
	AttachmentFile    = "file"
	AttachmentVoice   = "voice"
	AttachmentVideo   = "video"
	AttachmentSticker = "sticker"
)

// TIM elem msg_type constants.
const (
	timTextElem  = "TIMTextElem"
	timImageElem = "TIMImageElem"
	timFileElem  = "TIMFileElem"
	timSoundElem = "TIMSoundElem"
	timVideoElem = "TIMVideoFileElem"
	timFaceElem  = "TIMFaceElem"
)

// DefaultMaxMediaBytes is the per-attachment ceiling used when a caller does
// not provide one. It mirrors Hermes' DEFAULT_MAX_SIZE_MB (50 MiB) so a Go
// runtime swap stays parity-faithful without tightening.
const DefaultMaxMediaBytes int64 = 50 * 1024 * 1024

// TIMElement is the channel-neutral wrapper for a single TIM elem inside a
// MsgBody. msg_content is intentionally json.RawMessage so callers can branch
// per type without introducing a polymorphic schema.
type TIMElement struct {
	MsgType    string          `json:"msg_type"`
	MsgContent json.RawMessage `json:"msg_content"`
}

// MediaPolicy bounds attachment normalization. MaxBytes <= 0 means
// DefaultMaxMediaBytes.
type MediaPolicy struct {
	MaxBytes int64
}

// Attachment is the gateway-neutral media descriptor produced by the Yuanbao
// channel. Runtime adapters convert this into internal/gateway.Attachment when
// the row that wires Yuanbao to the gateway lands.
type Attachment struct {
	Kind      string
	URL       string
	MediaType string
	FileName  string
	SourceID  string
	Size      int64
	Error     string
}

// IsDegraded reports whether the attachment carries explicit unavailable
// evidence and therefore must not be treated as fetchable.
func (a Attachment) IsDegraded() bool {
	return strings.TrimSpace(a.Error) != ""
}

// NormalizeMedia produces gateway-neutral attachments for image/file/voice/
// video TIM elements present in body. Stickers are handled separately by
// NormalizeStickers; this keeps the binary-bytes and emoji-metadata paths from
// drifting in invariants.
func NormalizeMedia(body []TIMElement, policy MediaPolicy) ([]Attachment, error) {
	maxBytes := policy.MaxBytes
	if maxBytes <= 0 {
		maxBytes = DefaultMaxMediaBytes
	}

	var out []Attachment
	for _, elem := range body {
		switch elem.MsgType {
		case timImageElem:
			out = append(out, normalizeImage(elem.MsgContent, maxBytes))
		case timFileElem:
			out = append(out, normalizeFile(elem.MsgContent, maxBytes))
		case timSoundElem:
			out = append(out, normalizeSound(elem.MsgContent, maxBytes))
		case timVideoElem:
			out = append(out, normalizeVideo(elem.MsgContent, maxBytes))
		}
	}
	return out, nil
}

// ExtractTextBody returns the joined plain-text content of a TIM msg_body.
// It is the minimum text-preservation primitive the row's acceptance bullet
// "keeps the text portion of the inbound event" relies on; it intentionally
// does not duplicate the rich placeholder logic in Hermes' _extract_text — that
// belongs to a runtime row.
func ExtractTextBody(body []TIMElement) string {
	var parts []string
	for _, elem := range body {
		if elem.MsgType != timTextElem {
			continue
		}
		var content struct {
			Text string `json:"text"`
		}
		if err := json.Unmarshal(elem.MsgContent, &content); err != nil {
			continue
		}
		if t := strings.TrimSpace(content.Text); t != "" {
			parts = append(parts, t)
		}
	}
	return strings.Join(parts, " ")
}

func normalizeImage(raw json.RawMessage, maxBytes int64) Attachment {
	att := Attachment{Kind: AttachmentImage}

	var content struct {
		UUID           string `json:"uuid"`
		ImageFormat    int    `json:"image_format"`
		ImageInfoArray []struct {
			Type   int    `json:"type"`
			Size   int64  `json:"size"`
			Width  int    `json:"width"`
			Height int    `json:"height"`
			URL    string `json:"url"`
		} `json:"image_info_array"`
	}
	if err := json.Unmarshal(raw, &content); err != nil || len(content.ImageInfoArray) == 0 {
		att.Error = DegradedMediaUnavailable
		att.SourceID = peekSourceID(raw)
		return att
	}

	att.SourceID = content.UUID

	pick := content.ImageInfoArray[0]
	if len(content.ImageInfoArray) > 1 {
		pick = content.ImageInfoArray[1]
	}
	if strings.TrimSpace(pick.URL) == "" {
		att.Error = DegradedMediaUnavailable
		return att
	}
	if pick.Size < 0 || pick.Size > maxBytes {
		att.Error = DegradedMediaUnavailable
		return att
	}

	att.URL = pick.URL
	att.Size = pick.Size
	att.MediaType = imageFormatToMIME(content.ImageFormat, pick.URL)
	return att
}

func normalizeFile(raw json.RawMessage, maxBytes int64) Attachment {
	att := Attachment{Kind: AttachmentFile}

	var content struct {
		UUID     string `json:"uuid"`
		FileName string `json:"file_name"`
		FileSize int64  `json:"file_size"`
		URL      string `json:"url"`
	}
	if err := json.Unmarshal(raw, &content); err != nil {
		att.Error = DegradedMediaUnavailable
		att.SourceID = peekSourceID(raw)
		return att
	}

	att.SourceID = content.UUID
	att.FileName = content.FileName
	att.Size = content.FileSize

	if content.FileSize < 0 || content.FileSize > maxBytes {
		att.Error = DegradedMediaUnavailable
		return att
	}
	if strings.TrimSpace(content.URL) == "" {
		att.Error = DegradedMediaUnavailable
		return att
	}

	att.URL = content.URL
	att.MediaType = guessMimeFromName(content.FileName)
	return att
}

func normalizeSound(raw json.RawMessage, maxBytes int64) Attachment {
	att := Attachment{Kind: AttachmentVoice}

	var content struct {
		UUID   string `json:"uuid"`
		Size   int64  `json:"size"`
		Second int    `json:"second"`
		URL    string `json:"url"`
	}
	if err := json.Unmarshal(raw, &content); err != nil {
		att.Error = DegradedMediaUnavailable
		att.SourceID = peekSourceID(raw)
		return att
	}

	att.SourceID = content.UUID
	att.Size = content.Size

	if content.Size < 0 || content.Size > maxBytes {
		att.Error = DegradedMediaUnavailable
		return att
	}
	if strings.TrimSpace(content.URL) == "" {
		att.Error = DegradedMediaUnavailable
		return att
	}

	att.URL = content.URL
	att.MediaType = guessMimeFromName(content.URL)
	return att
}

func normalizeVideo(raw json.RawMessage, maxBytes int64) Attachment {
	att := Attachment{Kind: AttachmentVideo}

	var content struct {
		UUID      string `json:"uuid"`
		VideoSize int64  `json:"video_size"`
		VideoURL  string `json:"video_url"`
		FileName  string `json:"file_name"`
	}
	if err := json.Unmarshal(raw, &content); err != nil {
		att.Error = DegradedMediaUnavailable
		att.SourceID = peekSourceID(raw)
		return att
	}

	att.SourceID = content.UUID
	att.FileName = content.FileName
	att.Size = content.VideoSize

	if content.VideoSize < 0 || content.VideoSize > maxBytes {
		att.Error = DegradedMediaUnavailable
		return att
	}
	if strings.TrimSpace(content.VideoURL) == "" {
		att.Error = DegradedMediaUnavailable
		return att
	}

	att.URL = content.VideoURL
	att.MediaType = guessMimeFromName(content.FileName)
	return att
}

func peekSourceID(raw json.RawMessage) string {
	var probe struct {
		UUID string `json:"uuid"`
	}
	_ = json.Unmarshal(raw, &probe)
	return probe.UUID
}

// imageFormatToMIME mirrors Hermes' _MIME_TO_IMAGE_FORMAT map, inverted: Yuanbao
// inbound TIMImageElems carry a numeric image_format. Falls back to inferring
// from the URL extension when the numeric code is unset/unknown.
func imageFormatToMIME(format int, url string) string {
	switch format {
	case 1:
		return "image/jpeg"
	case 2:
		return "image/gif"
	case 3:
		return "image/png"
	case 4:
		return "image/bmp"
	}
	if mime := guessMimeFromName(url); strings.HasPrefix(mime, "image/") {
		return mime
	}
	return ""
}

// guessMimeFromName maps a filename or URL extension to a MIME type using a
// narrow, fixture-locked table. The contract intentionally stays narrow — a
// real download/CDN binding belongs to the runtime row.
func guessMimeFromName(name string) string {
	ext := strings.ToLower(path.Ext(extractPath(name)))
	switch ext {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	case ".bmp":
		return "image/bmp"
	case ".pdf":
		return "application/pdf"
	case ".doc":
		return "application/msword"
	case ".docx":
		return "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	case ".xls":
		return "application/vnd.ms-excel"
	case ".xlsx":
		return "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	case ".ppt":
		return "application/vnd.ms-powerpoint"
	case ".pptx":
		return "application/vnd.openxmlformats-officedocument.presentationml.presentation"
	case ".txt":
		return "text/plain"
	case ".zip":
		return "application/zip"
	case ".mp3":
		return "audio/mpeg"
	case ".mp4":
		return "video/mp4"
	case ".wav":
		return "audio/wav"
	case ".ogg":
		return "audio/ogg"
	case ".webm":
		return "video/webm"
	case ".amr":
		return "audio/amr"
	}
	return ""
}

// extractPath strips a query/fragment from a URL-shaped string so path.Ext
// looks at the actual filename. Bare filenames pass through unchanged.
func extractPath(s string) string {
	if i := strings.IndexAny(s, "?#"); i >= 0 {
		s = s[:i]
	}
	return s
}
