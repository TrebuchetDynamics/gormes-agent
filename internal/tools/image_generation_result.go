package tools

import "github.com/TrebuchetDynamics/gormes-agent/internal/tools/imagegen"

type ImageGenerationStatus = imagegen.ImageGenerationStatus

const (
	ImageGenerationStatusOK                    ImageGenerationStatus = imagegen.ImageGenerationStatusOK
	ImageGenerationStatusUnavailable           ImageGenerationStatus = imagegen.ImageGenerationStatusUnavailable
	ImageGenerationStatusProviderError         ImageGenerationStatus = imagegen.ImageGenerationStatusProviderError
	ImageGenerationStatusProviderNotRegistered ImageGenerationStatus = imagegen.ImageGenerationStatusProviderNotRegistered
)

type ImageGenerationEnvelope = imagegen.ImageGenerationEnvelope
type ImageGenerationRequest = imagegen.ImageGenerationRequest

func BuildImageGenerationEnvelope(req ImageGenerationRequest) (ImageGenerationEnvelope, error) {
	return imagegen.BuildImageGenerationEnvelope(req)
}
