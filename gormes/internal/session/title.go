package session

import (
	"context"
	"strings"
	"unicode"

	"github.com/TrebuchetDynamics/gormes-agent/gormes/internal/hermes"
)

var titleStopWords = map[string]struct{}{
	"a": {}, "about": {}, "after": {}, "an": {}, "and": {}, "are": {}, "as": {},
	"at": {}, "be": {}, "before": {}, "by": {}, "can": {}, "could": {}, "did": {},
	"do": {}, "does": {}, "for": {}, "from": {}, "hello": {}, "help": {}, "hi": {},
	"how": {}, "i": {}, "if": {}, "in": {}, "into": {}, "is": {}, "it": {},
	"just": {}, "let": {}, "lets": {}, "me": {}, "my": {}, "need": {}, "of": {},
	"on": {}, "or": {}, "our": {}, "please": {}, "really": {}, "should": {},
	"show": {}, "tell": {}, "thank": {}, "thanks": {}, "that": {}, "the": {},
	"this": {}, "to": {}, "we": {}, "what": {}, "when": {}, "where": {}, "why": {},
	"with": {}, "would": {}, "you": {}, "your": {},
}

// MaybeAutoTitle persists a deterministic short title after the first
// completed exchange. Later exchanges are ignored unless no title exists
// and the conversation is still within its first two user turns.
func MaybeAutoTitle(ctx context.Context, store MetadataStore, sessionID string, history []hermes.Message) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if store == nil || strings.TrimSpace(sessionID) == "" {
		return nil
	}

	userCount, userMessage, assistantResponse, ok := latestTitleExchange(history)
	if !ok || userCount > 2 {
		return nil
	}

	meta, found, err := store.GetMetadata(ctx, sessionID)
	if err != nil {
		return err
	}
	if found && strings.TrimSpace(meta.Title) != "" {
		return nil
	}

	title := GenerateTitle(userMessage, assistantResponse)
	if title == "" {
		return nil
	}
	return store.PutMetadata(ctx, Metadata{
		SessionID: sessionID,
		Title:     title,
	})
}

// GenerateTitle derives a short, bounded session title from the opening
// exchange without relying on an auxiliary model.
func GenerateTitle(userMessage, assistantResponse string) string {
	words := titleKeywords(userMessage, 4)
	if len(words) < 3 {
		words = append(words, titleKeywords(assistantResponse, 4-len(words))...)
	}
	if len(words) == 0 {
		return ""
	}

	parts := make([]string, 0, len(words))
	for _, word := range words {
		parts = append(parts, titleCaseWord(word))
	}

	title := strings.Join(parts, " ")
	if len(title) > 80 {
		title = strings.TrimSpace(title[:80])
	}
	return title
}

func latestTitleExchange(history []hermes.Message) (userCount int, userMessage, assistantResponse string, ok bool) {
	for _, msg := range history {
		if strings.EqualFold(strings.TrimSpace(msg.Role), "user") {
			userCount++
		}
	}

	for i := len(history) - 1; i >= 0; i-- {
		msg := history[i]
		role := strings.ToLower(strings.TrimSpace(msg.Role))
		content := strings.TrimSpace(msg.Content)
		if assistantResponse == "" {
			if role == "assistant" && content != "" {
				assistantResponse = content
			}
			continue
		}
		if role == "user" && content != "" {
			userMessage = content
			break
		}
	}

	return userCount, userMessage, assistantResponse, userMessage != "" && assistantResponse != ""
}

func titleKeywords(text string, limit int) []string {
	if limit <= 0 {
		return nil
	}
	tokens := strings.FieldsFunc(text, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
	if len(tokens) == 0 {
		return nil
	}

	out := make([]string, 0, limit)
	seen := make(map[string]struct{}, limit)
	for _, token := range tokens {
		word := strings.ToLower(strings.TrimSpace(token))
		if len(word) < 2 {
			continue
		}
		if _, skip := titleStopWords[word]; skip {
			continue
		}
		if _, dup := seen[word]; dup {
			continue
		}
		seen[word] = struct{}{}
		out = append(out, word)
		if len(out) == limit {
			break
		}
	}
	return out
}

func titleCaseWord(word string) string {
	runes := []rune(strings.ToLower(strings.TrimSpace(word)))
	if len(runes) == 0 {
		return ""
	}
	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
}
