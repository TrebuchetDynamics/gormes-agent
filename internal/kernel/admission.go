package kernel

import (
	"errors"
	"strings"
)

var (
	ErrEmptyInput    = errors.New("admission: input is empty")
	ErrInputTooLarge = errors.New("admission: input exceeds byte limit")
	ErrTooManyLines  = errors.New("admission: input exceeds line limit")
	ErrTurnInFlight  = errors.New("admission: still processing previous turn")
)

type Admission struct {
	MaxBytes int
	MaxLines int
}

// Validate runs the local admission guards. Returns nil if the input is safe
// to forward to the provider; otherwise a typed sentinel error the kernel
// surfaces verbatim in RenderFrame.LastError.
func (a Admission) Validate(text string) error {
	if strings.TrimSpace(text) == "" {
		return ErrEmptyInput
	}
	if len(text) > a.MaxBytes {
		return ErrInputTooLarge
	}
	if strings.Count(text, "\n")+1 > a.MaxLines {
		return ErrTooManyLines
	}
	return nil
}
