package main

import (
	"io"

	"github.com/skip2/go-qrcode"

	navivoxapp "github.com/TrebuchetDynamics/gormes-agent/internal/app/navivox"
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
)

type navivoxPairTerminalQR = navivoxapp.TerminalQR

func renderNavivoxPairTerminalQR(out io.Writer, cfg config.NavivoxCfg, baseURL, wsURL, qrPath string) error {
	return navivoxapp.RenderPairTerminalQR(out, cfg, baseURL, wsURL, qrPath)
}

func navivoxPairOpenQRCommand(qrPath string) string {
	return navivoxapp.OpenCommand(qrPath, navivoxAndroidEnvironment())
}

func navivoxPairOpenQRCommandForPlatform(qrPath, goos string, android bool) string {
	return navivoxapp.OpenCommandForPlatform(qrPath, goos, android)
}

func navivoxPairOpenQRPathArg(qrPath, goos string) string {
	return navivoxapp.OpenPathArg(qrPath, goos)
}

func navivoxCompactPairDescriptor(cfg config.NavivoxCfg, baseURL, wsURL string) string {
	return navivoxapp.CompactPairDescriptor(cfg.AuthMode, cfg.ExposureMode, cfg.Token, baseURL, wsURL)
}

func navivoxPairTerminalQRForColumns(descriptor string, columns int) (navivoxPairTerminalQR, error) {
	return navivoxapp.ForColumns(descriptor, columns)
}

func navivoxTerminalQRString(descriptor string, level qrcode.RecoveryLevel) (string, int, error) {
	return navivoxapp.TerminalQRString(descriptor, level)
}

func navivoxTerminalQRWidth(text string) int {
	return navivoxapp.Width(text)
}

func navivoxTerminalColumns(out io.Writer) int {
	return navivoxapp.Columns(out)
}
