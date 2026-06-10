package tools

import (
	"fmt"
	"strings"
)

func normalizeWebSearch(raw map[string]any) webSearchResponse {
	out := webSearchResponse{Success: true}
	candidates := webSearchCandidates(raw)
	out.Data.Web = make([]webSearchResult, 0, len(candidates))
	for i, item := range candidates {
		row, ok := item.(map[string]any)
		if !ok {
			continue
		}
		position := intValue(row["position"])
		if position <= 0 {
			position = i + 1
		}
		out.Data.Web = append(out.Data.Web, webSearchResult{
			Title:       webStringValue(row["title"]),
			URL:         webStringValue(row["url"]),
			Description: firstNonEmpty(webStringValue(row["description"]), webStringValue(row["content"]), webStringValue(row["snippet"]), strings.Join(webStringList(row["highlights"]), " "), strings.Join(webStringList(row["excerpts"]), " ")),
			Position:    position,
		})
	}
	if success, ok := raw["success"].(bool); ok {
		out.Success = success
	}
	if !out.Success {
		out.Error = webStringValue(raw["error"])
		out.Evidence = WebEvidenceRequestFailed
	}
	return out
}

func webSearchResponseHasNoSourceURLs(response webSearchResponse) bool {
	if !response.Success || len(response.Data.Web) == 0 {
		return false
	}
	for _, result := range response.Data.Web {
		if strings.TrimSpace(result.URL) != "" {
			return false
		}
	}
	return true
}

func normalizePerplexitySearch(raw map[string]any) map[string]any {
	answer := ""
	for _, choice := range mapList(raw["choices"]) {
		if message, _ := choice["message"].(map[string]any); message != nil {
			answer = webStringValue(message["content"])
			if answer != "" {
				break
			}
		}
	}
	citations := webStringList(raw["citations"])
	capacity := len(citations)
	if capacity == 0 {
		capacity = 1
	}
	results := make([]any, 0, capacity)
	if len(citations) == 0 {
		if answer != "" {
			results = append(results, map[string]any{
				"title":       "Perplexity answer",
				"description": answer,
				"position":    1,
			})
		}
		return map[string]any{"results": results}
	}
	for i, citation := range citations {
		citation = strings.TrimSpace(citation)
		if citation == "" {
			continue
		}
		results = append(results, map[string]any{
			"title":       fmt.Sprintf("Perplexity citation %d", i+1),
			"url":         citation,
			"description": answer,
			"position":    len(results) + 1,
		})
	}
	return map[string]any{"results": results}
}

func normalizeWebExtract(requestedURL, format string, raw map[string]any) webExtractResult {
	payload := raw
	if data, ok := raw["data"].(map[string]any); ok {
		payload = data
	}
	metadata, _ := payload["metadata"].(map[string]any)
	if metadata == nil {
		metadata = map[string]any{}
	}
	finalURL := firstNonEmpty(webStringValue(metadata["sourceURL"]), webStringValue(payload["url"]), requestedURL)
	title := firstNonEmpty(webStringValue(metadata["title"]), webStringValue(payload["title"]))
	content := webExtractContent(payload, format)
	result := webExtractResult{
		URL:        finalURL,
		Title:      title,
		Content:    content,
		Extraction: webExtractionFromRaw(payload["extraction"]),
	}
	if errMsg := webStringValue(payload["error"]); errMsg != "" {
		result.Error = errMsg
		result.Evidence = WebEvidenceRequestFailed
	}
	return result
}

func normalizeWebExtractDocuments(requestedURLs []string, format string, raw map[string]any) []webExtractResult {
	var rows []map[string]any
	if results := mapList(raw["results"]); len(results) > 0 {
		rows = append(rows, results...)
	}
	if content, ok := raw["content"].(map[string]any); ok {
		rows = append(rows, content)
	}
	out := make([]webExtractResult, 0, len(rows))
	for i, row := range rows {
		fallbackURL := ""
		if i < len(requestedURLs) {
			fallbackURL = requestedURLs[i]
		}
		out = append(out, normalizeWebExtractDocument(row, fallbackURL, format))
	}
	for _, fail := range mapList(raw["failed_results"]) {
		out = append(out, webExtractResult{
			URL:   firstNonEmpty(webStringValue(fail["url"]), firstRequestedURL(requestedURLs)),
			Error: firstNonEmpty(webStringValue(fail["error"]), "extraction failed"),
		})
	}
	for _, failURL := range webStringList(raw["failed_urls"]) {
		out = append(out, webExtractResult{
			URL:   failURL,
			Error: "extraction failed",
		})
	}
	return out
}

func normalizeWebExtractDocument(row map[string]any, fallbackURL string, format string) webExtractResult {
	metadata, _ := row["metadata"].(map[string]any)
	if metadata == nil {
		metadata = map[string]any{}
	}
	content := webExtractContent(row, format)
	if content == "" {
		content = firstNonEmpty(webStringValue(row["text"]), webStringValue(row["raw_content"]), webStringValue(row["full_content"]), strings.Join(webStringList(row["excerpts"]), "\n\n"))
	}
	result := webExtractResult{
		URL:        firstNonEmpty(webStringValue(row["url"]), webStringValue(metadata["sourceURL"]), webStringValue(metadata["url"]), fallbackURL),
		Title:      firstNonEmpty(webStringValue(row["title"]), webStringValue(metadata["title"])),
		Content:    content,
		Extraction: webExtractionFromRaw(row["extraction"]),
	}
	if errMsg := webStringValue(row["error"]); errMsg != "" {
		result.Error = errMsg
		result.Evidence = WebEvidenceRequestFailed
	}
	return result
}

func normalizeWebCrawlDocuments(requestedURL string, raw map[string]any) []webExtractResult {
	rows := make([]map[string]any, 0)
	if data := mapList(raw["data"]); len(data) > 0 {
		rows = append(rows, data...)
	}
	if results := mapList(raw["results"]); len(results) > 0 {
		rows = append(rows, results...)
	}
	out := make([]webExtractResult, 0, len(rows))
	for _, row := range rows {
		out = append(out, normalizeWebExtractDocument(row, requestedURL, "markdown"))
	}
	return out
}

func webSearchCandidates(raw map[string]any) []any {
	if results, ok := raw["results"].([]any); ok {
		return results
	}
	if web, ok := raw["web"].([]any); ok {
		return web
	}
	if web, ok := raw["web"].(map[string]any); ok {
		if results, ok := web["results"].([]any); ok {
			return results
		}
	}
	if data, ok := raw["data"].([]any); ok {
		return data
	}
	if data, ok := raw["data"].(map[string]any); ok {
		if web, ok := data["web"].([]any); ok {
			return web
		}
		if results, ok := data["results"].([]any); ok {
			return results
		}
	}
	return nil
}

// duckDuckGoExtract provides basic extraction when no API backends are
// configured. Public URLs are fetched locally with goscrapling first; the DDG
// Instant Answer API is only a fallback when static extraction cannot produce
