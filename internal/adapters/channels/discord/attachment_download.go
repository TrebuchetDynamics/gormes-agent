package discord

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/bwmarrin/discordgo"

	"github.com/TrebuchetDynamics/gormes-agent/internal/adapters/channels/discord/attachments"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools"
)

const (
	discordMaxAttachmentBytes        int64 = 32 * 1024 * 1024
	discordInlineTextDocumentBytes   int64 = 100 * 1024
	discordAttachmentHTTPTimeout           = 30 * time.Second
	discordAttachmentUnavailableCode       = "discord_attachment_unavailable"
	discordAttachmentBlockedSSRFCode       = "discord_attachment_blocked_ssrf"
	discordAttachmentTooLargeCode          = "discord_attachment_too_large"
	discordAttachmentUnsupportedCode       = "discord_attachment_unsupported"
)

var (
	errDiscordAttachmentReadUnavailable = errors.New("discord attachment read unavailable")
	errDiscordAttachmentBlockedSSRF     = errors.New("discord attachment blocked by SSRF guard")
	errDiscordAttachmentTooLarge        = errors.New("discord attachment too large")
	errDiscordAttachmentInvalidPayload  = errors.New("discord attachment invalid payload")
)

var discordSupportedDocumentExtensions = map[string]string{
	".cfg":  "text/plain",
	".csv":  "text/csv",
	".docx": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
	".ini":  "text/plain",
	".json": "application/json",
	".log":  "text/plain",
	".md":   "text/markdown",
	".pdf":  "application/pdf",
	".pptx": "application/vnd.openxmlformats-officedocument.presentationml.presentation",
	".toml": "application/toml",
	".txt":  "text/plain",
	".xlsx": "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
	".xml":  "application/xml",
	".yaml": "application/yaml",
	".yml":  "application/yaml",
	".zip":  "application/zip",
}

var discordMIMEExtensionFallbacks = map[string]string{
	"application/json": ".json",
	"application/pdf":  ".pdf",
	"application/toml": ".toml",
	"application/xml":  ".xml",
	"application/yaml": ".yaml",
	"application/zip":  ".zip",
	"text/csv":         ".csv",
	"text/markdown":    ".md",
	"text/plain":       ".txt",
}

type discordAttachmentClassification struct {
	kind      string
	mediaType string
	ext       string
	fileName  string
}

func (b *Bot) discordInboundTextAndAttachments(ctx context.Context, m *discordgo.Message) (string, []gateway.Attachment) {
	if m == nil {
		return "", nil
	}
	text := strings.TrimSpace(m.Content)
	if len(m.Attachments) == 0 {
		return text, nil
	}

	var (
		prefixes    []string
		markers     []string
		attachments []gateway.Attachment
	)
	for _, raw := range m.Attachments {
		prefix, attachment, marker := b.discordAttachmentDescriptor(ctx, raw)
		if prefix != "" {
			prefixes = append(prefixes, prefix)
		}
		if attachment != nil {
			attachments = append(attachments, *attachment)
		}
		if marker != "" {
			markers = append(markers, marker)
		}
	}
	if len(prefixes) > 0 {
		prefix := strings.Join(prefixes, "\n\n")
		if text == "" {
			text = prefix
		} else {
			text = prefix + "\n\n" + text
		}
	}
	for _, marker := range markers {
		if strings.TrimSpace(marker) == "" {
			continue
		}
		if text == "" {
			text = marker
		} else {
			text += "\n\n" + marker
		}
	}
	return text, attachments
}

func (b *Bot) discordAttachmentDescriptor(ctx context.Context, att *discordgo.MessageAttachment) (string, *gateway.Attachment, string) {
	classification, ok := classifyDiscordAttachment(att)
	if !ok {
		return "", nil, discordAttachmentEvidence(discordAttachmentUnsupportedCode, att, "unsupported attachment type")
	}
	if att != nil && att.Size > 0 && int64(att.Size) > discordMaxAttachmentBytes {
		return "", nil, discordAttachmentEvidence(discordAttachmentTooLargeCode, att, "attachment exceeds 32 MB")
	}

	data, err := b.readDiscordAttachmentBytes(ctx, att, classification)
	if err != nil {
		code := discordAttachmentUnavailableCode
		if errors.Is(err, errDiscordAttachmentBlockedSSRF) {
			code = discordAttachmentBlockedSSRFCode
		} else if errors.Is(err, errDiscordAttachmentTooLarge) {
			code = discordAttachmentTooLargeCode
		}
		return "", nil, discordAttachmentEvidence(code, att, discordAttachmentEvidenceReason(err))
	}
	if int64(len(data)) > discordMaxAttachmentBytes {
		return "", nil, discordAttachmentEvidence(discordAttachmentTooLargeCode, att, "attachment exceeds 32 MB")
	}
	if err := validateDiscordAttachmentPayload(classification, data); err != nil {
		return "", nil, discordAttachmentEvidence(discordAttachmentUnavailableCode, att, discordAttachmentEvidenceReason(err))
	}

	path, err := b.cacheDiscordAttachmentBytes(classification.kind+"s", classification.fileName, data)
	if err != nil {
		return "", nil, discordAttachmentEvidence(discordAttachmentUnavailableCode, att, "cache write failed")
	}
	attachment := &gateway.Attachment{
		Kind:      classification.kind,
		URL:       path,
		MediaType: classification.mediaType,
		FileName:  classification.fileName,
		SourceID:  discordAttachmentSourceID(att),
		SizeBytes: int64(len(data)),
	}

	var prefix string
	if classification.kind == "document" && discordShouldInlineDocument(classification.ext, int64(len(data))) && utf8.Valid(data) {
		prefix = fmt.Sprintf("[Content of %s]:\n%s", classification.fileName, string(data))
	}
	return prefix, attachment, ""
}

func (b *Bot) readDiscordAttachmentBytes(ctx context.Context, att *discordgo.MessageAttachment, classification discordAttachmentClassification) ([]byte, error) {
	if b.session != nil {
		if data, err := b.session.ReadAttachment(ctx, att); err == nil && len(data) > 0 {
			if int64(len(data)) > discordMaxAttachmentBytes {
				return nil, errDiscordAttachmentTooLarge
			}
			if validateDiscordAttachmentPayload(classification, data) == nil {
				return data, nil
			}
		}
	}
	data, _, err := b.fetchDiscordAttachmentURL(ctx, att)
	return data, err
}

func (b *Bot) fetchDiscordAttachmentURL(ctx context.Context, att *discordgo.MessageAttachment) ([]byte, string, error) {
	if att == nil || strings.TrimSpace(att.URL) == "" {
		return nil, "", errDiscordAttachmentReadUnavailable
	}
	if !b.discordAttachmentURLAllowed(att.URL) {
		return nil, "", errDiscordAttachmentBlockedSSRF
	}
	client := b.cfg.AttachmentHTTPClient
	if client == nil {
		client = &http.Client{Timeout: discordAttachmentHTTPTimeout}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, att.URL, nil)
	if err != nil {
		return nil, "", err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("fallback HTTP %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, discordMaxAttachmentBytes+1))
	if err != nil {
		return nil, "", err
	}
	if int64(len(body)) > discordMaxAttachmentBytes {
		return nil, "", errDiscordAttachmentTooLarge
	}
	return body, cleanDiscordMediaType(resp.Header.Get("Content-Type")), nil
}

func (b *Bot) discordAttachmentURLAllowed(rawURL string) bool {
	policy := tools.DefaultURLSafetyPolicy()
	policy.Allowlist = append(policy.Allowlist,
		tools.URLSafetyAllowlistEntry{Pattern: "cdn.discordapp.com", Source: "discord"},
		tools.URLSafetyAllowlistEntry{Pattern: "media.discordapp.net", Source: "discord"},
	)
	result := tools.NewURLSafetyChecker(policy).CheckURL(rawURL)
	return result.Safe
}

func classifyDiscordAttachment(att *discordgo.MessageAttachment) (discordAttachmentClassification, bool) {
	if att == nil {
		return discordAttachmentClassification{}, false
	}
	mediaType := cleanDiscordMediaType(att.ContentType)
	fileName := safeDiscordFileName(att.Filename)
	ext := inferDiscordAttachmentExt(fileName, mediaType)
	if mediaType == "" {
		mediaType = mime.TypeByExtension(ext)
		mediaType = cleanDiscordMediaType(mediaType)
	}
	switch {
	case strings.HasPrefix(mediaType, "image/"):
		if ext == "" {
			ext = discordImageExt(mediaType)
		}
		if !discordImageExtSupported(ext) {
			ext = ".jpg"
		}
		if fileName == "" {
			fileName = "image" + ext
		}
		return discordAttachmentClassification{kind: "image", mediaType: mediaType, ext: ext, fileName: fileName}, true
	case strings.HasPrefix(mediaType, "audio/"):
		if ext == "" {
			ext = discordAudioExt(mediaType)
		}
		if !discordAudioExtSupported(ext) {
			ext = ".ogg"
		}
		if fileName == "" {
			fileName = "audio" + ext
		}
		return discordAttachmentClassification{kind: "audio", mediaType: mediaType, ext: ext, fileName: fileName}, true
	default:
		if ext == "" {
			return discordAttachmentClassification{}, false
		}
		docType, ok := discordSupportedDocumentExtensions[ext]
		if !ok {
			return discordAttachmentClassification{}, false
		}
		if mediaType == "" || mediaType == "application/octet-stream" {
			mediaType = docType
		}
		if fileName == "" {
			fileName = "document" + ext
		}
		return discordAttachmentClassification{kind: "document", mediaType: mediaType, ext: ext, fileName: fileName}, true
	}
}

func inferDiscordAttachmentExt(fileName, mediaType string) string {
	if ext := strings.ToLower(filepath.Ext(strings.TrimSpace(fileName))); ext != "" {
		return ext
	}
	mediaType = cleanDiscordMediaType(mediaType)
	if ext := discordMIMEExtensionFallbacks[mediaType]; ext != "" {
		return ext
	}
	extensions, _ := mime.ExtensionsByType(mediaType)
	sort.Strings(extensions)
	for _, ext := range extensions {
		ext = strings.ToLower(ext)
		if _, ok := discordSupportedDocumentExtensions[ext]; ok || discordImageExtSupported(ext) || discordAudioExtSupported(ext) {
			return ext
		}
	}
	return ""
}

func validateDiscordAttachmentPayload(classification discordAttachmentClassification, data []byte) error {
	if len(data) == 0 {
		return errDiscordAttachmentInvalidPayload
	}
	if looksLikeDiscordHTMLPayload(data) {
		return errDiscordAttachmentInvalidPayload
	}
	switch classification.kind {
	case "image":
		if !looksLikeDiscordImage(data) {
			return errDiscordAttachmentInvalidPayload
		}
	case "audio":
		if !looksLikeDiscordAudio(data) {
			return errDiscordAttachmentInvalidPayload
		}
	case "document":
		switch classification.ext {
		case ".pdf":
			if !bytes.HasPrefix(data, []byte("%PDF")) {
				return errDiscordAttachmentInvalidPayload
			}
		case ".zip", ".docx", ".xlsx", ".pptx":
			if !bytes.HasPrefix(data, []byte("PK")) {
				return errDiscordAttachmentInvalidPayload
			}
		}
	}
	return nil
}

func (b *Bot) cacheDiscordAttachmentBytes(category, fileName string, data []byte) (string, error) {
	root := strings.TrimSpace(b.cfg.AttachmentCacheDir)
	if root == "" {
		cacheDir, err := os.UserCacheDir()
		if err != nil || strings.TrimSpace(cacheDir) == "" {
			cacheDir = os.TempDir()
		}
		root = filepath.Join(cacheDir, "gormes", "discord")
	}
	dir := filepath.Join(root, safeDiscordToken(category))
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	path := filepath.Join(dir, fmt.Sprintf("%d_%s", time.Now().UnixNano(), safeDiscordFileName(fileName)))
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return "", err
	}
	return filepath.Clean(path), nil
}

func discordTrustedAttachmentHost(rawURL string) bool {
	return attachments.TrustedHost(rawURL)
}

func cleanDiscordMediaType(mediaType string) string {
	return attachments.CleanMediaType(mediaType)
}

func safeDiscordFileName(fileName string) string {
	return attachments.SafeFileName(fileName)
}

func safeDiscordToken(s string) string {
	return attachments.SafeToken(s)
}

func discordAttachmentSourceID(att *discordgo.MessageAttachment) string {
	if att == nil {
		return ""
	}
	if id := strings.TrimSpace(att.ID); id != "" {
		return id
	}
	return safeDiscordFileName(att.Filename)
}

func discordAttachmentEvidence(code string, att *discordgo.MessageAttachment, reason string) string {
	var parts []string
	if code = strings.TrimSpace(code); code == "" {
		code = discordAttachmentUnavailableCode
	}
	parts = append(parts, "code="+code)
	if att != nil {
		if fileName := safeDiscordFileName(att.Filename); fileName != "" {
			parts = append(parts, "file="+fileName)
		}
		if mediaType := cleanDiscordMediaType(att.ContentType); mediaType != "" {
			parts = append(parts, "mime="+mediaType)
		}
		if att.Size > 0 {
			parts = append(parts, fmt.Sprintf("size=%d bytes", att.Size))
		}
	}
	if reason = strings.TrimSpace(reason); reason != "" {
		parts = append(parts, "reason="+reason)
	}
	return "[Discord attachment skipped: " + strings.Join(parts, "; ") + "]"
}

func discordAttachmentEvidenceReason(err error) string {
	switch {
	case err == nil:
		return ""
	case errors.Is(err, errDiscordAttachmentBlockedSSRF):
		return "blocked by SSRF guard"
	case errors.Is(err, errDiscordAttachmentTooLarge):
		return "attachment exceeds 32 MB"
	case errors.Is(err, errDiscordAttachmentInvalidPayload):
		return "invalid or HTML/error payload"
	default:
		return "download unavailable"
	}
}

func discordShouldInlineDocument(ext string, size int64) bool {
	if size <= 0 || size > discordInlineTextDocumentBytes {
		return false
	}
	switch strings.ToLower(ext) {
	case ".log", ".md", ".txt":
		return true
	default:
		return false
	}
}

func looksLikeDiscordHTMLPayload(data []byte) bool {
	trimmed := bytes.TrimSpace(bytes.ToLower(data))
	return bytes.HasPrefix(trimmed, []byte("<html")) ||
		bytes.HasPrefix(trimmed, []byte("<!doctype html")) ||
		bytes.Contains(trimmed[:min(len(trimmed), 256)], []byte("<title>forbidden"))
}

func looksLikeDiscordImage(data []byte) bool {
	return bytes.HasPrefix(data, []byte("\x89PNG\r\n\x1a\n")) ||
		bytes.HasPrefix(data, []byte("\xff\xd8\xff")) ||
		bytes.HasPrefix(data, []byte("GIF87a")) ||
		bytes.HasPrefix(data, []byte("GIF89a")) ||
		bytes.HasPrefix(data, []byte("RIFF")) && len(data) >= 12 && string(data[8:12]) == "WEBP"
}

func looksLikeDiscordAudio(data []byte) bool {
	return bytes.HasPrefix(data, []byte("OggS")) ||
		bytes.HasPrefix(data, []byte("ID3")) ||
		bytes.HasPrefix(data, []byte("RIFF")) && len(data) >= 12 && string(data[8:12]) == "WAVE" ||
		bytes.HasPrefix(data, []byte("fLaC")) ||
		len(data) >= 2 && data[0] == 0xff && data[1]&0xe0 == 0xe0
}

func discordImageExt(mediaType string) string {
	switch cleanDiscordMediaType(mediaType) {
	case "image/gif":
		return ".gif"
	case "image/jpeg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/webp":
		return ".webp"
	default:
		return ".jpg"
	}
}

func discordImageExtSupported(ext string) bool {
	switch strings.ToLower(strings.TrimSpace(ext)) {
	case ".gif", ".jpeg", ".jpg", ".png", ".webp":
		return true
	default:
		return false
	}
}

func discordAudioExt(mediaType string) string {
	switch cleanDiscordMediaType(mediaType) {
	case "audio/mpeg":
		return ".mp3"
	case "audio/wav", "audio/wave", "audio/x-wav":
		return ".wav"
	case "audio/webm":
		return ".webm"
	case "audio/mp4", "audio/aac":
		return ".m4a"
	default:
		return ".ogg"
	}
}

func discordAudioExtSupported(ext string) bool {
	switch strings.ToLower(strings.TrimSpace(ext)) {
	case ".aac", ".flac", ".m4a", ".mp3", ".ogg", ".opus", ".wav", ".webm":
		return true
	default:
		return false
	}
}
