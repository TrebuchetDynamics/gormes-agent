package navivox

import (
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/skip2/go-qrcode"
	"golang.org/x/term"
)

// TerminalQR describes a rendered compact QR and whether it fits the terminal.
type TerminalQR struct {
	Text          string
	Width         int
	Columns       int
	LevelName     string
	TooNarrow     bool
	RequiredWidth int
}

// OpenCommand returns the platform-specific command operators can use to open a QR image.
func OpenCommand(qrPath string, android bool) string {
	return OpenCommandForPlatform(qrPath, runtime.GOOS, android)
}

// OpenCommandForPlatform returns the command for a specific OS; tests use it for stable quoting.
func OpenCommandForPlatform(qrPath, goos string, android bool) string {
	pathArg := OpenPathArg(qrPath, goos)
	if android {
		return "termux-open " + pathArg
	}
	switch goos {
	case "darwin":
		return "open " + pathArg
	case "windows":
		return "start " + pathArg
	default:
		return "xdg-open " + pathArg
	}
}

// OpenPathArg quotes a QR path only when needed for the target shell family.
func OpenPathArg(qrPath, goos string) string {
	if qrPath == "" {
		if goos == "windows" {
			return `""`
		}
		return `''`
	}
	if !strings.ContainsAny(qrPath, " \t\n\r'\"\\$`;&|<>*?[]{}()!") {
		return qrPath
	}
	if goos == "windows" {
		return `"` + strings.ReplaceAll(qrPath, `"`, `\"`) + `"`
	}
	return `'` + strings.ReplaceAll(qrPath, `'`, `'\''`) + `'`
}

// CompactPairDescriptor builds the compact QR descriptor used for terminal rendering.
func CompactPairDescriptor(authMode, exposureMode, token, baseURL, wsURL string) string {
	values := url.Values{}
	values.Set("base_url", baseURL)
	values.Set("websocket_url", wsURL)
	values.Set("rest_token", token)
	return (&url.URL{Scheme: "navivox", Host: "connect", RawQuery: values.Encode()}).String()
}

// ForColumns renders descriptor at the highest recovery level that fits columns.
func ForColumns(descriptor string, columns int) (TerminalQR, error) {
	if strings.TrimSpace(descriptor) == "" {
		return TerminalQR{}, fmt.Errorf("navivox pair: pairing descriptor is empty")
	}
	candidates := []struct {
		name  string
		level qrcode.RecoveryLevel
	}{
		{name: "medium", level: qrcode.Medium},
		{name: "low", level: qrcode.Low},
	}
	var narrow TerminalQR
	for _, candidate := range candidates {
		text, width, err := String(descriptor, candidate.level)
		if err != nil {
			return TerminalQR{}, err
		}
		result := TerminalQR{Text: text, Width: width, Columns: columns, LevelName: candidate.name, RequiredWidth: width}
		if columns <= 0 || width <= columns {
			return result, nil
		}
		narrow = result
	}
	narrow.Text = ""
	narrow.TooNarrow = true
	return narrow, nil
}

// String renders a QR as terminal text and returns its display width.
func String(descriptor string, level qrcode.RecoveryLevel) (string, int, error) {
	qr, err := qrcode.New(descriptor, level)
	if err != nil {
		return "", 0, fmt.Errorf("navivox pair: encode terminal QR: %w", err)
	}
	text := qr.ToSmallString(false)
	return text, Width(text), nil
}

// Width returns the maximum rune width of a terminal QR text block.
func Width(text string) int {
	maxWidth := 0
	for _, line := range strings.Split(strings.TrimRight(text, "\n"), "\n") {
		if width := len([]rune(line)); width > maxWidth {
			maxWidth = width
		}
	}
	return maxWidth
}

// WritePNG writes a descriptor QR image using private file permissions.
func WritePNG(path, descriptor string) error {
	if strings.TrimSpace(descriptor) == "" {
		return fmt.Errorf("navivox pair: pairing descriptor is empty")
	}
	pngBytes, err := qrcode.Encode(descriptor, qrcode.Medium, 512)
	if err != nil {
		return fmt.Errorf("navivox pair: encode pairing QR: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("navivox pair: create QR directory: %w", err)
	}
	if err := os.WriteFile(path, pngBytes, 0o600); err != nil {
		return fmt.Errorf("navivox pair: write pairing QR: %w", err)
	}
	if err := os.Chmod(path, 0o600); err != nil {
		return fmt.Errorf("navivox pair: secure pairing QR: %w", err)
	}
	return nil
}

// Columns returns the terminal width using COLUMNS or an attached file descriptor.
func Columns(out io.Writer) int {
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
