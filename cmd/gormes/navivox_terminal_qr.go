package main

import (
	"fmt"
	"io"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/skip2/go-qrcode"
	"golang.org/x/term"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
)

type navivoxPairTerminalQR struct {
	Text          string
	Width         int
	Columns       int
	LevelName     string
	TooNarrow     bool
	RequiredWidth int
}

func renderNavivoxPairTerminalQR(out io.Writer, cfg config.NavivoxCfg, baseURL, wsURL, qrPath string) error {
	descriptor := navivoxCompactPairDescriptor(cfg, baseURL, wsURL)
	terminalQR, err := navivoxPairTerminalQRForColumns(descriptor, navivoxTerminalColumns(out))
	if err != nil {
		return err
	}
	if terminalQR.TooNarrow {
		fmt.Fprintf(out, "  Terminal QR hidden: %d cols < %d.\n", terminalQR.Columns, terminalQR.RequiredWidth)
		fmt.Fprintf(out, "  Open: termux-open %s\n", qrPath)
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

func navivoxCompactPairDescriptor(cfg config.NavivoxCfg, baseURL, wsURL string) string {
	values := url.Values{}
	values.Set("base_url", baseURL)
	values.Set("websocket_url", wsURL)
	values.Set("capabilities_url", strings.TrimRight(baseURL, "/")+"/v1/navivox/capabilities")
	values.Set("auth_mode", cfg.AuthMode)
	values.Set("exposure_mode", cfg.ExposureMode)
	values.Set("token_required", "true")
	values.Set("rest_token", cfg.Token)
	return (&url.URL{Scheme: "navivox", Host: "connect", RawQuery: values.Encode()}).String()
}

func navivoxPairTerminalQRForColumns(descriptor string, columns int) (navivoxPairTerminalQR, error) {
	if strings.TrimSpace(descriptor) == "" {
		return navivoxPairTerminalQR{}, fmt.Errorf("navivox pair: pairing descriptor is empty")
	}
	candidates := []struct {
		name  string
		level qrcode.RecoveryLevel
	}{
		{name: "medium", level: qrcode.Medium},
		{name: "low", level: qrcode.Low},
	}
	var narrow navivoxPairTerminalQR
	for _, candidate := range candidates {
		text, width, err := navivoxTerminalQRString(descriptor, candidate.level)
		if err != nil {
			return navivoxPairTerminalQR{}, err
		}
		result := navivoxPairTerminalQR{Text: text, Width: width, Columns: columns, LevelName: candidate.name, RequiredWidth: width}
		if columns <= 0 || width <= columns {
			return result, nil
		}
		narrow = result
	}
	narrow.Text = ""
	narrow.TooNarrow = true
	return narrow, nil
}

func navivoxTerminalQRString(descriptor string, level qrcode.RecoveryLevel) (string, int, error) {
	qr, err := qrcode.New(descriptor, level)
	if err != nil {
		return "", 0, fmt.Errorf("navivox pair: encode terminal QR: %w", err)
	}
	text := qr.ToSmallString(false)
	return text, navivoxTerminalQRWidth(text), nil
}

func navivoxTerminalQRWidth(text string) int {
	maxWidth := 0
	for _, line := range strings.Split(strings.TrimRight(text, "\n"), "\n") {
		if width := len([]rune(line)); width > maxWidth {
			maxWidth = width
		}
	}
	return maxWidth
}

func navivoxTerminalColumns(out io.Writer) int {
	if raw := strings.TrimSpace(os.Getenv("COLUMNS")); raw != "" {
		if width, err := strconv.Atoi(raw); err == nil && width > 0 {
			return width
		}
	}
	file, ok := out.(*os.File)
	if !ok || file == nil {
		return 0
	}
	width, _, err := term.GetSize(int(file.Fd()))
	if err != nil || width <= 0 {
		return 0
	}
	return width
}
