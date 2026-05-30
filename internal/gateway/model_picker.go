package gateway

import (
	"context"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway/modelpicker"
)

type SessionModelOverride = modelpicker.SessionModelOverride

type ModelPickerRequest = modelpicker.ModelPickerRequest

type ModelPickerCallback = modelpicker.ModelPickerCallback

type ModelPickerResponse = modelpicker.ModelPickerResponse

type ModelPickerResolver interface {
	OpenModelPicker(ctx context.Context, req ModelPickerRequest) (ModelPickerResponse, error)
	HandleModelPickerCallback(ctx context.Context, cb ModelPickerCallback) (ModelPickerResponse, error)
	PickerProviders() []string
	PickerModels(providerSlug string) []string
}

func NewModelPickerResolver(ov *SessionModelOverride) ModelPickerResolver {
	return modelpicker.NewModelPickerResolver(ov)
}
