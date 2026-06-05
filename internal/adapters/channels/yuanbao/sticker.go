package yuanbao

import (
	"encoding/json"
	"strings"
)

// Sticker is a normalized TIMFaceElem descriptor. Stickers are emoji-style
// metadata, not binary attachments, so they retain a separate type from
// Attachment; AsAttachment lifts the identifier into the gateway-neutral shape
// when a runtime row needs to surface it alongside other media.
type Sticker struct {
	StickerID string
	PackageID string
	Name      string
	Formats   string
	Width     int
	Height    int
	Error     string
}

// supportedStickerFormats locks the inbound rendering invariant: only static
// raster formats are safe to render without a transcoder. apng, gif, lottie,
// and any unknown format are surfaced as degraded so the event survives.
var supportedStickerFormats = map[string]struct{}{
	"png":  {},
	"jpg":  {},
	"jpeg": {},
	"webp": {},
}

// NormalizeStickers extracts every TIMFaceElem from body. Malformed or
// animated sticker payloads are returned with an explicit Error so callers can
// preserve the inbound event instead of dropping it.
func NormalizeStickers(body []TIMElement) ([]Sticker, error) {
	var out []Sticker
	for _, elem := range body {
		if elem.MsgType != timFaceElem {
			continue
		}
		out = append(out, normalizeSticker(elem.MsgContent))
	}
	return out, nil
}

func normalizeSticker(raw json.RawMessage) Sticker {
	var envelope struct {
		Index int    `json:"index"`
		Data  string `json:"data"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return Sticker{Error: DegradedStickerUnsupported}
	}
	if strings.TrimSpace(envelope.Data) == "" {
		return Sticker{Error: DegradedStickerUnsupported}
	}

	var meta struct {
		StickerID string `json:"sticker_id"`
		PackageID string `json:"package_id"`
		Name      string `json:"name"`
		Formats   string `json:"formats"`
		Width     int    `json:"width"`
		Height    int    `json:"height"`
	}
	if err := json.Unmarshal([]byte(envelope.Data), &meta); err != nil {
		return Sticker{Error: DegradedStickerUnsupported}
	}

	s := Sticker{
		StickerID: meta.StickerID,
		PackageID: meta.PackageID,
		Name:      meta.Name,
		Formats:   meta.Formats,
		Width:     meta.Width,
		Height:    meta.Height,
	}
	if meta.StickerID == "" {
		s.Error = DegradedStickerUnsupported
		return s
	}
	if _, ok := supportedStickerFormats[strings.ToLower(strings.TrimSpace(meta.Formats))]; !ok {
		s.Error = DegradedStickerUnsupported
	}
	return s
}

// AsAttachment lifts a Sticker into a gateway-neutral Attachment so the
// runtime can deliver stickers alongside other inbound media. The MediaType is
// inferred from the sticker's Formats; degraded stickers carry the same Error
// code through the conversion.
func (s Sticker) AsAttachment() Attachment {
	mediaType := ""
	switch strings.ToLower(strings.TrimSpace(s.Formats)) {
	case "png":
		mediaType = "image/png"
	case "jpg", "jpeg":
		mediaType = "image/jpeg"
	case "webp":
		mediaType = "image/webp"
	case "apng":
		mediaType = "image/apng"
	case "gif":
		mediaType = "image/gif"
	case "lottie":
		mediaType = "application/json"
	}
	return Attachment{
		Kind:      AttachmentSticker,
		SourceID:  s.StickerID,
		FileName:  s.Name,
		MediaType: mediaType,
		Error:     s.Error,
	}
}
