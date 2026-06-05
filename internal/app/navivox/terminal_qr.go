package navivox

import (
	"fmt"
	"io"
	"strings"

	"github.com/skip2/go-qrcode"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
)

// RenderPairTerminalQR prints a compact pairing QR when it fits the terminal.
func RenderPairTerminalQR(out io.Writer, cfg config.NavivoxCfg, baseURL, wsURL, qrPath string) error {
	descriptor := CompactPairDescriptor(cfg.AuthMode, cfg.ExposureMode, cfg.Token, baseURL, wsURL)
	terminalQR, err := ForColumns(descriptor, Columns(out))
	if err != nil {
		return err
	}
	if terminalQR.TooNarrow {
		fmt.Fprintf(out, "  Terminal QR hidden: %d cols < %d.\n", terminalQR.Columns, terminalQR.RequiredWidth)
		fmt.Fprintf(out, "  Open: %s\n", OpenCommand(qrPath, AndroidEnvironment()))
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

// TerminalQRString renders a QR as terminal text for compatibility callers.
func TerminalQRString(descriptor string, level qrcode.RecoveryLevel) (string, int, error) {
	return String(descriptor, level)
}

func renderNavivoxPairTerminalQR(out io.Writer, cfg config.NavivoxCfg, baseURL, wsURL, qrPath string) error {
	return RenderPairTerminalQR(out, cfg, baseURL, wsURL, qrPath)
}

func navivoxPairOpenQRCommand(qrPath string) string {
	return OpenCommand(qrPath, AndroidEnvironment())
}

func navivoxPairOpenQRCommandForPlatform(qrPath, goos string, android bool) string {
	return OpenCommandForPlatform(qrPath, goos, android)
}

func navivoxPairOpenQRPathArg(qrPath, goos string) string {
	return OpenPathArg(qrPath, goos)
}

func navivoxCompactPairDescriptor(cfg config.NavivoxCfg, baseURL, wsURL string) string {
	return CompactPairDescriptor(cfg.AuthMode, cfg.ExposureMode, cfg.Token, baseURL, wsURL)
}

func navivoxPairTerminalQRForColumns(descriptor string, columns int) (TerminalQR, error) {
	return ForColumns(descriptor, columns)
}

func navivoxTerminalQRString(descriptor string, level qrcode.RecoveryLevel) (string, int, error) {
	return TerminalQRString(descriptor, level)
}

func navivoxTerminalQRWidth(text string) int { return Width(text) }

func navivoxTerminalColumns(out io.Writer) int { return Columns(out) }
