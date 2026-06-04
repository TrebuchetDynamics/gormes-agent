package llm

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
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

const imageExtPattern = "png|jpg|jpeg|gif|webp|bmp|tiff|tif|heic"

var (
	fencedCodeBlockRe = regexp.MustCompile("(?s)```[^\n]*\n.*?```")
	inlineCodeSpanRe  = regexp.MustCompile("`[^`\n]+`")
	localImagePathRe  = regexp.MustCompile(`(?i)(?:^|[^/:A-Za-z0-9_.])((?:~/|/)(?:[A-Za-z0-9_.-]+/)*[A-Za-z0-9_.-]+\.(?:` + imageExtPattern + `))\b`)
	remoteImageURLRe  = regexp.MustCompile(`(?i)https?://[^\s<>"']+?\.(?:` + imageExtPattern + `)(?:\?[^\s<>"']*)?`)
)

// ExtractImageRefs scans free-form text for image references the model should
// see. It returns readable local image paths (absolute or ~/ home-relative,
// expanded and deduplicated) plus HTTP(S) image URLs (deduplicated, not
// fetched). Matches inside fenced or inline code are ignored so examples do not
// become live attachments.
func ExtractImageRefs(text string) ([]string, []string) {
	if text == "" {
		return nil, nil
	}
	codeSpans := imageRoutingCodeSpans(text)
	inCode := func(pos int) bool {
		for _, span := range codeSpans {
			if pos >= span.start && pos < span.end {
				return true
			}
		}
		return false
	}

	paths := make([]string, 0)
	seenPaths := make(map[string]struct{})
	for _, match := range localImagePathRe.FindAllStringSubmatchIndex(text, -1) {
		if len(match) < 4 || match[2] < 0 || inCode(match[2]) {
			continue
		}
		expanded := expandImagePath(text[match[2]:match[3]])
		info, err := os.Stat(expanded)
		if err != nil || info.IsDir() {
			continue
		}
		if _, ok := seenPaths[expanded]; ok {
			continue
		}
		seenPaths[expanded] = struct{}{}
		paths = append(paths, expanded)
	}

	urls := make([]string, 0)
	seenURLs := make(map[string]struct{})
	for _, loc := range remoteImageURLRe.FindAllStringIndex(text, -1) {
		if len(loc) != 2 || inCode(loc[0]) {
			continue
		}
		url := strings.TrimRightFunc(text[loc[0]:loc[1]], func(r rune) bool {
			return strings.ContainsRune(".,;:!?)]>", r)
		})
		if url == "" {
			continue
		}
		if _, ok := seenURLs[url]; ok {
			continue
		}
		seenURLs[url] = struct{}{}
		urls = append(urls, url)
	}

	return paths, urls
}

type imageRoutingSpan struct {
	start int
	end   int
}

func imageRoutingCodeSpans(text string) []imageRoutingSpan {
	spans := make([]imageRoutingSpan, 0)
	for _, loc := range fencedCodeBlockRe.FindAllStringIndex(text, -1) {
		spans = append(spans, imageRoutingSpan{start: loc[0], end: loc[1]})
	}
	for _, loc := range inlineCodeSpanRe.FindAllStringIndex(text, -1) {
		spans = append(spans, imageRoutingSpan{start: loc[0], end: loc[1]})
	}
	return spans
}

func expandImagePath(raw string) string {
	if strings.HasPrefix(raw, "~/") {
		home := os.Getenv("HOME")
		if home == "" {
			if resolved, err := os.UserHomeDir(); err == nil {
				home = resolved
			}
		}
		if home != "" {
			return filepath.Join(home, raw[2:])
		}
	}
	return raw
}

// BuildNativeImageContentParts constructs a multimodal content list for a
// user turn: one leading text part followed by one image_url part per readable
// path or remote URL. The text part preserves the user's caption, or uses a
// default image prompt when the caption is empty, and appends a handle hint for
// each attached image. Unreadable paths are returned in skipped and are not
// advertised in the text hints. Remote URLs are passed through without local
// validation so the provider can fetch them server-side.
func BuildNativeImageContentParts(userText string, imagePaths []string, imageURLGroups ...[]string) ([]MessageContentPart, []string) {
	text := strings.TrimSpace(userText)
	imageURLCount := 0
	for _, group := range imageURLGroups {
		imageURLCount += len(group)
	}
	parts := make([]MessageContentPart, 0, 1+len(imagePaths)+imageURLCount)
	skipped := make([]string, 0)
	attachedPaths := make([]string, 0, len(imagePaths))
	attachedURLs := make([]string, 0, imageURLCount)
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
	for _, group := range imageURLGroups {
		for _, rawURL := range group {
			url := strings.TrimSpace(rawURL)
			if url == "" {
				continue
			}
			parts = append(parts, MessageContentPart{
				Type:     "image_url",
				ImageURL: url,
			})
			attachedURLs = append(attachedURLs, url)
		}
	}
	if len(attachedPaths) > 0 || len(attachedURLs) > 0 {
		baseText := text
		if baseText == "" {
			baseText = defaultImagePromptText
		}
		hints := make([]string, 0, len(attachedPaths)+len(attachedURLs))
		for _, path := range attachedPaths {
			hints = append(hints, "[Image attached at: "+path+"]")
		}
		for _, url := range attachedURLs {
			hints = append(hints, "[Image attached: "+url+"]")
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
	return "data:" + guessImageMIME(rawPath, raw) + ";base64," + base64.StdEncoding.EncodeToString(raw), nil
}

func guessImageMIME(rawPath string, raw []byte) string {
	if mime, ok := sniffImageMIME(raw); ok {
		return mime
	}
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
	case ".tiff", ".tif":
		return "image/tiff"
	case ".heic", ".heif":
		return "image/heic"
	}
	return "image/jpeg"
}

func sniffImageMIME(raw []byte) (string, bool) {
	if len(raw) == 0 {
		return "", false
	}
	if len(raw) >= 8 && string(raw[:8]) == "\x89PNG\r\n\x1a\n" {
		return "image/png", true
	}
	if len(raw) >= 3 && raw[0] == 0xff && raw[1] == 0xd8 && raw[2] == 0xff {
		return "image/jpeg", true
	}
	if len(raw) >= 6 {
		magic := string(raw[:6])
		if magic == "GIF87a" || magic == "GIF89a" {
			return "image/gif", true
		}
	}
	if len(raw) >= 12 && string(raw[:4]) == "RIFF" && string(raw[8:12]) == "WEBP" {
		return "image/webp", true
	}
	if len(raw) >= 2 && raw[0] == 'B' && raw[1] == 'M' {
		return "image/bmp", true
	}
	if len(raw) >= 12 && string(raw[4:8]) == "ftyp" {
		switch string(raw[8:12]) {
		case "heic", "heix", "hevc", "hevx", "mif1", "msf1", "heim", "heis":
			return "image/heic", true
		}
	}
	return "", false
}
