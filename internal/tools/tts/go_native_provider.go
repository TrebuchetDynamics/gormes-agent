//go:build !slim

package tts

import "github.com/TrebuchetDynamics/gormes-agent/internal/tools/tts/localgo"

type GoNativeTTSProviderConfig = localgo.Config

type GoNativeTTSProvider = localgo.Provider

func NewGoNativeTTSProvider(cfg GoNativeTTSProviderConfig) *GoNativeTTSProvider {
	return localgo.NewProvider(cfg)
}
