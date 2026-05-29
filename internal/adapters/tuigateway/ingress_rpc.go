package tuigateway

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tui"
)

// IngressOptions keeps composer RPC handlers hermetic in tests.
type IngressOptions struct {
	HomeDir           string
	Stat              func(string) (fs.FileInfo, error)
	ReadImageMetadata func(string) (ImageMetadata, error)
	CollapsePaste     func(string) (string, error)
}

// ImageAttachRequest is the image.attach JSON-RPC payload used by Hermes Ink.
type ImageAttachRequest struct {
	SessionID string `json:"session_id"`
	Path      string `json:"path"`
}

// ImageAttachResponse is the image.attach result. Name is intentionally
// enough for the local composer to render an attachment row; metadata is
// included for gateway observers that also need dimensions or token estimate.
type ImageAttachResponse struct {
	Name      string        `json:"name"`
	Path      string        `json:"path"`
	Remainder string        `json:"remainder,omitempty"`
	Metadata  ImageMetadata `json:"metadata,omitempty"`
}

// InputDetectDropRequest is the input.detect_drop payload used after
// image.attach declines or fails.
type InputDetectDropRequest struct {
	SessionID string `json:"session_id"`
	Text      string `json:"text"`
}

// InputDetectDropResponse is the text replacement envelope for dropped files.
type InputDetectDropResponse struct {
	Matched   bool   `json:"matched"`
	Text      string `json:"text,omitempty"`
	Path      string `json:"path,omitempty"`
	IsImage   bool   `json:"is_image,omitempty"`
	Remainder string `json:"remainder,omitempty"`
	Evidence  string `json:"evidence,omitempty"`
}

// PasteCollapseRequest is the paste.collapse payload used for large pastes.
type PasteCollapseRequest struct {
	Text string `json:"text"`
}

// PasteCollapseResponse points at the caller-managed paste artifact.
type PasteCollapseResponse struct {
	Path string `json:"path,omitempty"`
}

// HandleImageAttach validates and attaches one local image path without
// sending anything to a provider.
func HandleImageAttach(req ImageAttachRequest, opts IngressOptions) (ImageAttachResponse, error) {
	drop := detectGatewayIngressDrop(stripImageCommandPrefix(req.Path), opts)
	if !drop.Matched {
		return ImageAttachResponse{}, fmt.Errorf("tui_ingress_file_missing")
	}
	if !drop.IsImage {
		return ImageAttachResponse{}, fmt.Errorf("tui_ingress_not_image")
	}
	readMeta := opts.ReadImageMetadata
	if readMeta == nil {
		readMeta = ReadImageMetadata
	}
	meta, err := readMeta(drop.Path)
	if err != nil {
		return ImageAttachResponse{}, err
	}
	if meta.Name == "" {
		meta.Name = filepath.Base(drop.Path)
	}
	return ImageAttachResponse{
		Name:      meta.Name,
		Path:      drop.Path,
		Remainder: drop.Remainder,
		Metadata:  meta,
	}, nil
}

// HandleInputDetectDrop validates any local file path and returns the
// composer-visible replacement text.
func HandleInputDetectDrop(req InputDetectDropRequest, opts IngressOptions) (InputDetectDropResponse, error) {
	drop := detectGatewayIngressDrop(req.Text, opts)
	if !drop.Matched {
		return InputDetectDropResponse{Evidence: drop.Evidence}, nil
	}
	text := filepath.Base(drop.Path)
	if drop.Remainder != "" {
		text += " " + drop.Remainder
	}
	return InputDetectDropResponse{
		Matched:   true,
		Text:      text,
		Path:      drop.Path,
		IsImage:   drop.IsImage,
		Remainder: drop.Remainder,
	}, nil
}

// HandlePasteCollapse delegates long-paste storage to an injected artifact
// writer and returns only the pointer envelope.
func HandlePasteCollapse(req PasteCollapseRequest, opts IngressOptions) (PasteCollapseResponse, error) {
	if opts.CollapsePaste == nil {
		return PasteCollapseResponse{}, fmt.Errorf("tui_ingress_paste_collapse_unavailable")
	}
	path, err := opts.CollapsePaste(req.Text)
	if err != nil {
		return PasteCollapseResponse{}, err
	}
	return PasteCollapseResponse{Path: path}, nil
}

func detectGatewayIngressDrop(input string, opts IngressOptions) tui.ComposerDropResult {
	return tui.DetectComposerDroppedFile(input, tui.ComposerDropOptions{
		HomeDir: opts.HomeDir,
		Stat:    opts.Stat,
	})
}

func stripImageCommandPrefix(input string) string {
	trimmed := strings.TrimSpace(input)
	if trimmed == "/image" {
		return ""
	}
	if strings.HasPrefix(trimmed, "/image ") {
		return strings.TrimSpace(strings.TrimPrefix(trimmed, "/image"))
	}
	return trimmed
}
