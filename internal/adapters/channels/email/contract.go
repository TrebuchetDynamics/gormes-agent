package email

import (
	"bufio"
	"bytes"
	"fmt"
	"net/mail"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/adapters/channels/email/allowlist"
	"github.com/TrebuchetDynamics/gormes-agent/internal/adapters/channels/email/body"
	"github.com/TrebuchetDynamics/gormes-agent/internal/adapters/channels/email/delivery"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
)

const platformName = "email"

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

// InboundDispatchOptions controls the pure email ingress contract before a
// live IMAP/SMTP adapter exists.
type InboundDispatchOptions struct {
	AllowedSenders     []string
	BuildThreadContext func(NormalizedInbound) error
	Dispatch           func(gateway.InboundEvent) error
}

// SenderDeniedEvidence is bounded denial evidence for non-allowlisted email.
type SenderDeniedEvidence = allowlist.SenderDeniedEvidence

// InboundDispatchResult reports whether an RFC 822 message was accepted,
// dropped by policy, or ignored because it had no usable sender/body.
type InboundDispatchResult struct {
	Accepted   bool
	Dropped    bool
	Normalized bool
	Inbound    NormalizedInbound
	Evidence   SenderDeniedEvidence
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

	return normalizeInboundMessage(msg, from)
}

// DispatchInboundWithAllowlist applies the Hermes email sender allowlist before
// constructing thread context or dispatching a gateway event.
func DispatchInboundWithAllowlist(raw []byte, opts InboundDispatchOptions) (InboundDispatchResult, error) {
	msg, err := mail.ReadMessage(bufio.NewReader(bytes.NewReader(raw)))
	if err != nil {
		return InboundDispatchResult{}, fmt.Errorf("email: parse message: %w", err)
	}
	from, ok := firstAddress(msg.Header.Get("From"))
	if !ok {
		return InboundDispatchResult{}, nil
	}
	sender := normalizeEmailAddress(from.Address)
	if !emailSenderAllowed(sender, opts.AllowedSenders) {
		return InboundDispatchResult{
			Dropped: true,
			Evidence: SenderDeniedEvidence{
				Code:   "email_sender_denied",
				Sender: evidenceSender(sender),
				Domain: emailAddressDomain(sender),
				Reason: "sender_not_allowlisted",
			},
		}, nil
	}

	inbound, ok, err := normalizeInboundMessage(msg, from)
	if err != nil {
		return InboundDispatchResult{}, err
	}
	if !ok {
		return InboundDispatchResult{}, nil
	}
	result := InboundDispatchResult{
		Accepted:   true,
		Normalized: true,
		Inbound:    inbound,
	}
	if opts.BuildThreadContext != nil {
		if err := opts.BuildThreadContext(inbound); err != nil {
			return result, fmt.Errorf("email: build thread context: %w", err)
		}
	}
	if opts.Dispatch != nil {
		if err := opts.Dispatch(inbound.Event); err != nil {
			return result, fmt.Errorf("email: dispatch inbound: %w", err)
		}
	}
	return result, nil
}

func normalizeInboundMessage(msg *mail.Message, from *mail.Address) (NormalizedInbound, bool, error) {
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

func normalizeEmailAddress(addr string) string {
	return allowlist.NormalizeAddress(addr)
}

func emailSenderAllowed(sender string, allowed []string) bool {
	return allowlist.SenderAllowed(sender, allowed)
}

func emailAddressDomain(sender string) string {
	return allowlist.AddressDomain(sender)
}

func evidenceSender(sender string) string {
	return allowlist.EvidenceSender(sender)
}

func firstAddress(raw string) (*mail.Address, bool) {
	list, err := mail.ParseAddressList(raw)
	if err != nil || len(list) == 0 {
		return nil, false
	}
	return list[0], true
}

func extractBody(contentType string, reader interface{ Read([]byte) (int, error) }) (string, error) {
	return body.Extract(contentType, reader)
}

func normalizeWhitespace(text string) string {
	return body.NormalizeWhitespace(text)
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
type Delivery = delivery.Delivery

// BuildDelivery serializes an outbound email reply to RFC 5322 bytes. It
// always emits exactly one Date header: caller-supplied Date is preserved,
// otherwise a Date is computed from now() (or time.Now when nil).
func BuildDelivery(d Delivery, now func() time.Time) []byte {
	return delivery.Build(d, now)
}
