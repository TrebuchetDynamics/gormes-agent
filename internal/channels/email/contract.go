package email

import (
	"bufio"
	"bytes"
	"fmt"
	"html"
	"io"
	"mime"
	"mime/multipart"
	"net/mail"
	"regexp"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
)

const platformName = "email"

var htmlTagPattern = regexp.MustCompile(`<[^>]+>`)

// ReplyTarget captures the outbound threading headers for an email reply.
type ReplyTarget struct {
	To         string
	Subject    string
	InReplyTo  string
	References string
}

// NormalizedInbound is the email adapter output consumed by the shared gateway
// and the future SMTP delivery edge.
type NormalizedInbound struct {
	Event gateway.InboundEvent
	Reply ReplyTarget
}

// NormalizeInbound parses an RFC 822 message into the shared gateway contract.
func NormalizeInbound(raw []byte) (NormalizedInbound, bool, error) {
	msg, err := mail.ReadMessage(bufio.NewReader(bytes.NewReader(raw)))
	if err != nil {
		return NormalizedInbound{}, false, fmt.Errorf("email: parse message: %w", err)
	}

	from, ok := firstAddress(msg.Header.Get("From"))
	if !ok {
		return NormalizedInbound{}, false, nil
	}

	body, err := extractBody(msg.Header.Get("Content-Type"), msg.Body)
	if err != nil {
		return NormalizedInbound{}, false, err
	}
	body = strings.TrimSpace(body)
	if body == "" {
		return NormalizedInbound{}, false, nil
	}

	subject := strings.TrimSpace(msg.Header.Get("Subject"))
	messageID := strings.TrimSpace(msg.Header.Get("Message-ID"))
	kind, parsedBody := gateway.ParseInboundText(body)
	if kind == gateway.EventSubmit && subject != "" && !isReplySubject(subject) {
		parsedBody = "[Subject: " + subject + "]\n\n" + parsedBody
	}

	reply := ReplyTarget{
		To:         strings.ToLower(strings.TrimSpace(from.Address)),
		Subject:    replySubject(subject),
		InReplyTo:  messageID,
		References: replyReferences(strings.TrimSpace(msg.Header.Get("References")), messageID),
	}
	return NormalizedInbound{
		Event: gateway.InboundEvent{
			Platform: platformName,
			ChatID:   reply.To,
			UserID:   reply.To,
			UserName: strings.TrimSpace(from.Name),
			MsgID:    messageID,
			Kind:     kind,
			Text:     parsedBody,
		},
		Reply: reply,
	}, true, nil
}

func firstAddress(raw string) (*mail.Address, bool) {
	list, err := mail.ParseAddressList(raw)
	if err != nil || len(list) == 0 {
		return nil, false
	}
	return list[0], true
}

func extractBody(contentType string, body io.Reader) (string, error) {
	mediaType, params, err := mime.ParseMediaType(contentType)
	if err != nil {
		mediaType = "text/plain"
		params = map[string]string{}
	}

	switch {
	case strings.HasPrefix(mediaType, "multipart/"):
		return extractMultipartBody(params["boundary"], body)
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

func extractMultipartBody(boundary string, body io.Reader) (string, error) {
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
		content, err := extractBody(part.Header.Get("Content-Type"), part)
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
	return normalizeWhitespace(text), nil
}

func normalizeWhitespace(text string) string {
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

func isReplySubject(subject string) bool {
	return strings.HasPrefix(strings.ToLower(strings.TrimSpace(subject)), "re:")
}

func replySubject(subject string) string {
	subject = strings.TrimSpace(subject)
	if subject == "" || isReplySubject(subject) {
		return subject
	}
	return "Re: " + subject
}

func replyReferences(existing, messageID string) string {
	existing = strings.TrimSpace(existing)
	messageID = strings.TrimSpace(messageID)
	switch {
	case existing == "":
		return messageID
	case messageID == "":
		return existing
	default:
		return existing + " " + messageID
	}
}

// Delivery captures the outbound email payload BuildDelivery serializes
// into RFC 5322 byte form.
type Delivery struct {
	From       string
	To         string
	Subject    string
	InReplyTo  string
	References string
	Body       string
	// Date is preserved verbatim when non-empty; otherwise BuildDelivery
	// supplies an RFC 5322 Date from the injected clock.
	Date string
}

// BuildDelivery serializes an outbound email reply to RFC 5322 bytes. It
// always emits exactly one Date header: caller-supplied Date is preserved,
// otherwise a Date is computed from now() (or time.Now when nil).
func BuildDelivery(d Delivery, now func() time.Time) []byte {
	if now == nil {
		now = time.Now
	}
	var b bytes.Buffer
	if d.From != "" {
		fmt.Fprintf(&b, "From: %s\r\n", d.From)
	}
	if d.To != "" {
		fmt.Fprintf(&b, "To: %s\r\n", d.To)
	}
	if d.Subject != "" {
		fmt.Fprintf(&b, "Subject: %s\r\n", d.Subject)
	}
	if d.InReplyTo != "" {
		fmt.Fprintf(&b, "In-Reply-To: %s\r\n", d.InReplyTo)
	}
	if d.References != "" {
		fmt.Fprintf(&b, "References: %s\r\n", d.References)
	}
	date := strings.TrimSpace(d.Date)
	if date == "" {
		date = now().Format(time.RFC1123Z)
	}
	fmt.Fprintf(&b, "Date: %s\r\n", date)
	fmt.Fprintf(&b, "Content-Type: text/plain; charset=UTF-8\r\n")
	fmt.Fprintf(&b, "\r\n")
	fmt.Fprint(&b, d.Body)
	return b.Bytes()
}
