//go:build !slim

// Package cloud owns shared HTTP request mechanics for cloud STT providers.
package cloud

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// Field is a multipart form field written before the audio file.
type Field struct {
	Name  string
	Value string
}

// PostBearerMultipart posts an audio multipart request with bearer auth.
func PostBearerMultipart(ctx context.Context, client *http.Client, label, baseURL, endpoint, apiKey, audioPath string, fields []Field) (*http.Response, error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	for _, field := range fields {
		if err := writer.WriteField(field.Name, field.Value); err != nil {
			return nil, fmt.Errorf("%s %s field: %w", label, field.Name, err)
		}
	}

	f, err := os.Open(audioPath)
	if err != nil {
		return nil, fmt.Errorf("%s open audio: %w", label, err)
	}
	defer f.Close()

	part, err := writer.CreateFormFile("file", filepath.Base(audioPath))
	if err != nil {
		return nil, fmt.Errorf("%s create form file: %w", label, err)
	}
	if _, err := io.Copy(part, f); err != nil {
		return nil, fmt.Errorf("%s copy audio: %w", label, err)
	}
	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("%s close writer: %w", label, err)
	}

	url := strings.TrimSuffix(baseURL, "/") + endpoint
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, body)
	if err != nil {
		return nil, fmt.Errorf("%s request: %w", label, err)
	}
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)
	httpReq.Header.Set("Content-Type", writer.FormDataContentType())

	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("%s HTTP: %w", label, err)
	}
	return resp, nil
}

// RequireOK converts non-200 cloud STT responses into bounded error text.
func RequireOK(label string, resp *http.Response) error {
	if resp.StatusCode == http.StatusOK {
		return nil
	}
	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
	return fmt.Errorf("%s HTTP %d: %s", label, resp.StatusCode, string(respBody))
}

// ReadTrimmedText reads a raw text transcript response body.
func ReadTrimmedText(label string, body io.Reader) (string, error) {
	bodyBytes, err := io.ReadAll(body)
	if err != nil {
		return "", fmt.Errorf("%s read response: %w", label, err)
	}
	return strings.TrimSpace(string(bodyBytes)), nil
}
