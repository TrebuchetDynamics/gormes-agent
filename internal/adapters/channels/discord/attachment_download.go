package discord

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
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
	errDiscordAttachmentInvalidPayload  = attachments.ErrInvalidPayload
)

type discordAttachmentClassification = attachments.Classification

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

	path, err := b.cacheDiscordAttachmentBytes(classification.Kind+"s", classification.FileName, data)
	if err != nil {
		return "", nil, discordAttachmentEvidence(discordAttachmentUnavailableCode, att, "cache write failed")
	}
	attachment := &gateway.Attachment{
		Kind:      classification.Kind,
		URL:       path,
		MediaType: classification.MediaType,
		FileName:  classification.FileName,
		SourceID:  discordAttachmentSourceID(att),
		SizeBytes: int64(len(data)),
	}

	var prefix string
	if classification.Kind == "document" && discordShouldInlineDocument(classification.Ext, int64(len(data))) && utf8.Valid(data) {
		prefix = fmt.Sprintf("[Content of %s]:\n%s", classification.FileName, string(data))
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
	return attachments.Classify(att)
}

func inferDiscordAttachmentExt(fileName, mediaType string) string {
	return attachments.InferExt(fileName, mediaType)
}

func validateDiscordAttachmentPayload(classification discordAttachmentClassification, data []byte) error {
	return attachments.ValidatePayload(classification, data)
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
	return attachments.ShouldInlineDocument(ext, size)
}

func looksLikeDiscordHTMLPayload(data []byte) bool {
	return attachments.LooksLikeHTMLPayload(data)
}

func looksLikeDiscordImage(data []byte) bool {
	return attachments.LooksLikeImage(data)
}

func looksLikeDiscordAudio(data []byte) bool {
	return attachments.LooksLikeAudio(data)
}

func discordImageExt(mediaType string) string {
	return attachments.ImageExt(mediaType)
}

func discordImageExtSupported(ext string) bool {
	return attachments.ImageExtSupported(ext)
}

func discordAudioExt(mediaType string) string {
	return attachments.AudioExt(mediaType)
}

func discordAudioExtSupported(ext string) bool {
	return attachments.AudioExtSupported(ext)
}
