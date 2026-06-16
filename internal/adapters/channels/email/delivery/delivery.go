package delivery

import (
	"bytes"
	"fmt"
	"strings"
	"time"
)

// Delivery captures the outbound email payload Build serializes into RFC 5322 byte form.
type Delivery struct {
	From       string
	To         string
	Subject    string
	InReplyTo  string
	References string
	Body       string
	// Date is preserved verbatim when non-empty; otherwise Build supplies an RFC 5322 Date from the injected clock.
	Date string
}

// Build serializes an outbound email reply to RFC 5322 bytes.
func Build(d Delivery, now func() time.Time) []byte {
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
