package hermes

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ImageInputMode selects how user-attached images are presented to the
// active model.
type ImageInputMode string

const (
	// ImageInputModeAuto resolves to native or text based on model vision
	// capability and auxiliary vision config.
	ImageInputModeAuto ImageInputMode = "auto"
	// ImageInputModeNative attaches images as multimodal content parts.
	ImageInputModeNative ImageInputMode = "native"
	// ImageInputModeText runs vision_analyze up-front and prepends a text
	// description.
	ImageInputModeText ImageInputMode = "text"
)

// AuxiliaryVisionConfig captures the config.yaml auxiliary.vision block.
// When non-empty (anything other than provider in {"", "auto"} with empty
// model and base_url), the user has opted into a dedicated vision pipeline
// and auto mode must respect that.
type AuxiliaryVisionConfig struct {
	Provider string
	Model    string
	BaseURL  string
}

// ImageRoutingConfig is the pure input to DecideImageInputMode. Model
// capability lookup and config parsing are caller responsibilities.
type ImageRoutingConfig struct {
	Mode                  ImageInputMode
	AuxiliaryVision       AuxiliaryVisionConfig
	ModelVisionCapability ModelCapabilityFlag
}

// DecideImageInputMode resolves auto/native/text for the current turn,
// mirroring upstream Hermes agent/image_routing.py:decide_image_input_mode.
// Auto mode chooses native only when the active model is known to support
// vision and no auxiliary vision provider/model/base_url override is set.
func DecideImageInputMode(cfg ImageRoutingConfig) ImageInputMode {
	switch normalizeImageInputMode(cfg.Mode) {
	case ImageInputModeNative:
		return ImageInputModeNative
	case ImageInputModeText:
		return ImageInputModeText
	}

	if explicitAuxVisionOverride(cfg.AuxiliaryVision) {
		return ImageInputModeText
	}
	if cfg.ModelVisionCapability == ModelCapabilitySupported {
		return ImageInputModeNative
	}
	return ImageInputModeText
}

func normalizeImageInputMode(raw ImageInputMode) ImageInputMode {
	switch ImageInputMode(strings.ToLower(strings.TrimSpace(string(raw)))) {
	case ImageInputModeNative:
		return ImageInputModeNative
	case ImageInputModeText:
		return ImageInputModeText
	}
	return ImageInputModeAuto
}

func explicitAuxVisionOverride(aux AuxiliaryVisionConfig) bool {
	provider := strings.ToLower(strings.TrimSpace(aux.Provider))
	model := strings.TrimSpace(aux.Model)
	baseURL := strings.TrimSpace(aux.BaseURL)
	if (provider == "" || provider == "auto") && model == "" && baseURL == "" {
		return false
	}
	return true
}

const defaultImagePromptText = "What do you see in this image?"

// BuildNativeImageContentParts constructs a multimodal content list for a
// user turn: one leading text part followed by one image_url part per readable
// path. The text part preserves the user's caption, or uses a default image
// prompt when the caption is empty, and appends a local path hint for each
// attached image. Unreadable paths are returned in skipped and are not
// advertised in the text hints.
func BuildNativeImageContentParts(userText string, imagePaths []string) ([]MessageContentPart, []string) {
	text := strings.TrimSpace(userText)
	parts := make([]MessageContentPart, 0, 1+len(imagePaths))
	skipped := make([]string, 0)
	attachedPaths := make([]string, 0, len(imagePaths))
	for _, raw := range imagePaths {
		dataURL, err := imageFileToDataURL(raw)
		if err != nil {
			skipped = append(skipped, raw)
			continue
		}
		parts = append(parts, MessageContentPart{
			Type:     "image_url",
			ImageURL: dataURL,
		})
		attachedPaths = append(attachedPaths, raw)
	}
	if len(attachedPaths) > 0 {
		baseText := text
		if baseText == "" {
			baseText = defaultImagePromptText
		}
		hints := make([]string, 0, len(attachedPaths))
		for _, path := range attachedPaths {
			hints = append(hints, "[Image attached at: "+path+"]")
		}
		parts = append([]MessageContentPart{{Type: "text", Text: baseText + "\n\n" + strings.Join(hints, "\n")}}, parts...)
	} else if text != "" {
		parts = append(parts, MessageContentPart{Type: "text", Text: text})
	}
	return parts, skipped
}

func imageFileToDataURL(rawPath string) (string, error) {
	info, err := os.Stat(rawPath)
	if err != nil {
		return "", err
	}
	if info.IsDir() {
		return "", fmt.Errorf("image_routing: %q is a directory", rawPath)
	}
	raw, err := os.ReadFile(rawPath)
	if err != nil {
		return "", err
	}
	return "data:" + guessImageMIME(rawPath) + ";base64," + base64.StdEncoding.EncodeToString(raw), nil
}

func guessImageMIME(rawPath string) string {
	switch strings.ToLower(filepath.Ext(rawPath)) {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	case ".bmp":
		return "image/bmp"
	}
	return "image/jpeg"
}
