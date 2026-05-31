package delivery

import deliverymedia "github.com/TrebuchetDynamics/gormes-agent/internal/gateway/delivery/media"

type MediaKind = deliverymedia.MediaKind

const (
	MediaKindAudio    MediaKind = deliverymedia.MediaKindAudio
	MediaKindDocument MediaKind = deliverymedia.MediaKindDocument
	MediaKindImage    MediaKind = deliverymedia.MediaKindImage
	MediaKindVideo    MediaKind = deliverymedia.MediaKindVideo
)

type Media = deliverymedia.Media
type MediaEvidence = deliverymedia.MediaEvidence

const (
	MediaEvidenceExtracted MediaEvidence = deliverymedia.MediaEvidenceExtracted
	MediaEvidenceIgnored   MediaEvidence = deliverymedia.MediaEvidenceIgnored
)

type MediaEvidenceRecord = deliverymedia.MediaEvidenceRecord
type MediaContent = deliverymedia.MediaContent

// PrepareMediaContent extracts Hermes MEDIA tags from final assistant text so
// channels can deliver files natively without leaking tag syntax to the
// operator-visible message.
func PrepareMediaContent(finalText string) MediaContent {
	return deliverymedia.PrepareMediaContent(finalText)
}

func CleanMediaPath(raw string) (string, bool) {
	return deliverymedia.CleanMediaPath(raw)
}

func SupportedMediaExt(ext string) bool {
	return deliverymedia.SupportedMediaExt(ext)
}

// ClassifyMedia returns the explicit media kind or infers it from the local
// file extension for older call sites that only populated Path.
func ClassifyMedia(media Media) MediaKind {
	return deliverymedia.ClassifyMedia(media)
}

func MediaKindForPath(path string) MediaKind {
	return deliverymedia.MediaKindForPath(path)
}
