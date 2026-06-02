package main

import (
	"fmt"
	"io"
	"strings"

	"github.com/skip2/go-qrcode"

	"github.com/TrebuchetDynamics/gormes-agent/internal/app/navivox"
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
)

type navivoxPairTerminalQR = navivox.TerminalQR

func renderNavivoxPairTerminalQR(out io.Writer, cfg config.NavivoxCfg, baseURL, wsURL, qrPath string) error {
	descriptor := navivoxCompactPairDescriptor(cfg, baseURL, wsURL)
	terminalQR, err := navivoxPairTerminalQRForColumns(descriptor, navivoxTerminalColumns(out))
	if err != nil {
		return err
	}
	if terminalQR.TooNarrow {
		fmt.Fprintf(out, "  Terminal QR hidden: %d cols < %d.\n", terminalQR.Columns, terminalQR.RequiredWidth)
		fmt.Fprintf(out, "  Open: %s\n", navivoxPairOpenQRCommand(qrPath))
		return nil
	}
	fmt.Fprintln(out, "  Scan QR:")
	for _, line := range strings.Split(strings.TrimRight(terminalQR.Text, "\n"), "\n") {
		fmt.Fprintln(out, line)
	}
	if terminalQR.LevelName == "low" {
		fmt.Fprintln(out, "  QR compacted to fit this terminal.")
	}
	return nil
}

func navivoxPairOpenQRCommand(qrPath string) string {
	return navivox.OpenCommand(qrPath, navivoxAndroidEnvironment())
}

func navivoxPairOpenQRCommandForPlatform(qrPath, goos string, android bool) string {
	return navivox.OpenCommandForPlatform(qrPath, goos, android)
}

func navivoxPairOpenQRPathArg(qrPath, goos string) string {
	return navivox.OpenPathArg(qrPath, goos)
}

func navivoxCompactPairDescriptor(cfg config.NavivoxCfg, baseURL, wsURL string) string {
	return navivox.CompactPairDescriptor(cfg.AuthMode, cfg.ExposureMode, cfg.Token, baseURL, wsURL)
}

func navivoxPairTerminalQRForColumns(descriptor string, columns int) (navivoxPairTerminalQR, error) {
	return navivox.ForColumns(descriptor, columns)
}

func navivoxTerminalQRString(descriptor string, level qrcode.RecoveryLevel) (string, int, error) {
	return navivox.String(descriptor, level)
}

func navivoxTerminalQRWidth(text string) int {
	return navivox.Width(text)
}

func navivoxTerminalColumns(out io.Writer) int {
	return navivox.Columns(out)
}
