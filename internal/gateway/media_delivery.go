package gateway

import gatewaydelivery "github.com/TrebuchetDynamics/gormes-agent/internal/gateway/delivery"

type MediaDeliveryEvidence = gatewaydelivery.MediaEvidence

const (
	MediaDeliveryEvidenceExtracted MediaDeliveryEvidence = gatewaydelivery.MediaEvidenceExtracted
	MediaDeliveryEvidenceIgnored   MediaDeliveryEvidence = gatewaydelivery.MediaEvidenceIgnored
)

type MediaDeliveryEvidenceRecord = gatewaydelivery.MediaEvidenceRecord

type MediaDeliveryContent = gatewaydelivery.MediaContent

// PrepareMediaDeliveryContent extracts Hermes MEDIA tags from final assistant
// text so channels can deliver files natively without leaking tag syntax to
// the operator-visible message.
func PrepareMediaDeliveryContent(finalText string) MediaDeliveryContent {
	return gatewaydelivery.PrepareMediaContent(finalText)
}

func cleanOutboundMediaPath(raw string) (string, bool) {
	return gatewaydelivery.CleanMediaPath(raw)
}

func supportedOutboundMediaExt(ext string) bool {
	return gatewaydelivery.SupportedMediaExt(ext)
}

// ClassifyOutboundMedia returns the explicit media kind or infers it from the
// local file extension for older call sites that only populated Path.
func ClassifyOutboundMedia(media OutboundMedia) OutboundMediaKind {
	return gatewaydelivery.ClassifyMedia(media)
}

func OutboundMediaKindForPath(path string) OutboundMediaKind {
	return gatewaydelivery.MediaKindForPath(path)
}
