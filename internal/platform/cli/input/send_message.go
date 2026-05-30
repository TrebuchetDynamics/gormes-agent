package input

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"unicode/utf8"
)

var (
	ErrSendMessageMissingBody = errors.New("send_message_body_missing")
	ErrSendMessageInvalidText = errors.New("send_message_invalid_text")
)

type SendMessageBodyOptions struct {
	Positional string
	FilePath   string
	Stdin      io.Reader
	StdinIsTTY bool
	ReadFile   func(string) ([]byte, error)
}

type SendMessageBody struct {
	Text          string
	Source        string
	SanitizerMeta TerminalResponseSanitizerMeta
}

func ResolveSendMessageBody(opts SendMessageBodyOptions) (SendMessageBody, error) {
	if opts.Positional != "" {
		return sendMessageBodyFromBytes([]byte(opts.Positional), "positional", "")
	}

	if opts.FilePath != "" {
		if opts.FilePath == "-" {
			data, err := readSendMessageStdin(opts.Stdin)
			if err != nil {
				return SendMessageBody{}, fmt.Errorf("cannot read stdin: %w", err)
			}
			return sendMessageBodyFromBytes(data, "stdin", "stdin")
		}
		readFile := opts.ReadFile
		if readFile == nil {
			readFile = os.ReadFile
		}
		data, err := readFile(opts.FilePath)
		if err != nil {
			return SendMessageBody{}, fmt.Errorf("cannot read %s: %w", opts.FilePath, err)
		}
		return sendMessageBodyFromBytes(data, "file", opts.FilePath)
	}

	if !opts.StdinIsTTY {
		data, err := readSendMessageStdin(opts.Stdin)
		if err != nil {
			return SendMessageBody{}, fmt.Errorf("cannot read stdin: %w", err)
		}
		if len(data) > 0 {
			return sendMessageBodyFromBytes(data, "stdin", "stdin")
		}
	}

	return SendMessageBody{}, fmt.Errorf("%w: no message provided", ErrSendMessageMissingBody)
}

func readSendMessageStdin(stdin io.Reader) ([]byte, error) {
	if stdin == nil {
		stdin = os.Stdin
	}
	return io.ReadAll(stdin)
}

func sendMessageBodyFromBytes(data []byte, source, label string) (SendMessageBody, error) {
	if !utf8.Valid(data) || bytes.Contains(data, []byte{0}) {
		if label == "" {
			return SendMessageBody{}, fmt.Errorf("%w: message body must be UTF-8 text", ErrSendMessageInvalidText)
		}
		return SendMessageBody{}, fmt.Errorf("cannot read %s: %w: message body must be UTF-8 text", label, ErrSendMessageInvalidText)
	}
	text := string(data)
	cleaned, meta := StripLeakedTerminalResponsesWithMeta(text)
	if strings.TrimSpace(cleaned) == "" {
		return SendMessageBody{}, fmt.Errorf("%w: no message provided", ErrSendMessageMissingBody)
	}
	return SendMessageBody{Text: cleaned, Source: source, SanitizerMeta: meta}, nil
}
