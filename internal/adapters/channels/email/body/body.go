package body

import (
	"fmt"
	"html"
	"io"
	"mime"
	"mime/multipart"
	"regexp"
	"strings"
)

var htmlTagPattern = regexp.MustCompile(`<[^>]+>`)

func Extract(contentType string, body io.Reader) (string, error) {
	mediaType, params, err := mime.ParseMediaType(contentType)
	if err != nil {
		mediaType = "text/plain"
		params = map[string]string{}
	}

	switch {
	case strings.HasPrefix(mediaType, "multipart/"):
		return extractMultipart(params["boundary"], body)
	case mediaType == "text/html":
		return htmlToText(body)
	default:
		text, readErr := io.ReadAll(body)
		if readErr != nil {
			return "", fmt.Errorf("email: read body: %w", readErr)
		}
		return strings.TrimSpace(string(text)), nil
	}
}

func extractMultipart(boundary string, body io.Reader) (string, error) {
	if strings.TrimSpace(boundary) == "" {
		return "", nil
	}
	reader := multipart.NewReader(body, boundary)
	var fallback string
	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", fmt.Errorf("email: read multipart: %w", err)
		}
		content, err := Extract(part.Header.Get("Content-Type"), part)
		if err != nil {
			return "", err
		}
		content = strings.TrimSpace(content)
		if content == "" {
			continue
		}
		mediaType, _, _ := mime.ParseMediaType(part.Header.Get("Content-Type"))
		if mediaType == "text/plain" {
			return content, nil
		}
		if fallback == "" {
			fallback = content
		}
	}
	return fallback, nil
}

func htmlToText(body io.Reader) (string, error) {
	raw, err := io.ReadAll(body)
	if err != nil {
		return "", fmt.Errorf("email: read html body: %w", err)
	}
	text := string(raw)
	replacer := strings.NewReplacer(
		"<br>", "\n",
		"<br/>", "\n",
		"<br />", "\n",
		"</p>", "\n",
		"</div>", "\n",
	)
	text = replacer.Replace(text)
	text = htmlTagPattern.ReplaceAllString(text, "")
	text = html.UnescapeString(text)
	return NormalizeWhitespace(text), nil
}

func NormalizeWhitespace(text string) string {
	lines := strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		out = append(out, line)
	}
	return strings.TrimSpace(strings.Join(out, "\n"))
}
