package tts

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	defaultSampleRate    = 16000
	defaultBitsPerSample = 16
	defaultChannels      = 1
	defaultMaxTextLength = 2000
)

// ErrorCode is stable typed evidence for Go-owned local TTS runtime failures.
type ErrorCode string

const (
	ErrorCodeProviderUnavailable ErrorCode = "tts_provider_unavailable"
	ErrorCodeInvalidInput        ErrorCode = "tts_invalid_input"
	ErrorCodeSynthesisFailed     ErrorCode = "tts_api_error"
)

// Error is a redacted local TTS error. It deliberately carries a code rather
// than raw runtime details so callers can map it onto operator-facing evidence.
type Error struct {
	Code    ErrorCode
	Message string
}

func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	msg := strings.TrimSpace(e.Message)
	if msg == "" {
		msg = string(e.Code)
	}
	return string(e.Code) + ": " + msg
}

func IsErrorCode(err error, code ErrorCode) bool {
	var typed *Error
	return errors.As(err, &typed) && typed.Code == code
}

// Request is the Go-owned local speech synthesis contract. It is intentionally
// file-oriented to match Hermes/Gormes text_to_speech provider behavior.
type Request struct {
	Text          string
	OutputPath    string
	Voice         string
	Speed         float64
	SampleRate    int
	MaxTextLength int
}

type Result struct {
	FilePath      string
	Format        string
	SampleRate    int
	Channels      int
	BitsPerSample int
	Duration      time.Duration
	Bytes         int
}

type Runtime interface {
	Synthesize(context.Context, Request) (Result, error)
}

// FixtureSynthesizer is a deliberately low-quality pure-Go fixture/formant
// synthesizer. It exists to prove the runtime seam and offline WAV envelope;
// neural Piper/WASI quality belongs behind the same Runtime interface later.
type FixtureSynthesizer struct {
	Disabled      bool
	SampleRate    int
	MaxTextLength int
}

func NewFixtureSynthesizer() *FixtureSynthesizer {
	return &FixtureSynthesizer{SampleRate: defaultSampleRate, MaxTextLength: defaultMaxTextLength}
}

func (s *FixtureSynthesizer) Synthesize(ctx context.Context, req Request) (Result, error) {
	if err := ctx.Err(); err != nil {
		return Result{}, &Error{Code: ErrorCodeProviderUnavailable, Message: err.Error()}
	}
	if s == nil || s.Disabled {
		return Result{}, &Error{Code: ErrorCodeProviderUnavailable, Message: "local fixture TTS runtime is disabled"}
	}
	text := strings.TrimSpace(req.Text)
	if text == "" {
		return Result{}, &Error{Code: ErrorCodeInvalidInput, Message: "text is required"}
	}
	maxLen := req.MaxTextLength
	if maxLen <= 0 {
		maxLen = s.MaxTextLength
	}
	if maxLen <= 0 {
		maxLen = defaultMaxTextLength
	}
	if len([]rune(text)) > maxLen {
		return Result{}, &Error{Code: ErrorCodeInvalidInput, Message: fmt.Sprintf("text exceeds local fixture TTS limit of %d characters", maxLen)}
	}
	out := strings.TrimSpace(req.OutputPath)
	if out == "" {
		return Result{}, &Error{Code: ErrorCodeInvalidInput, Message: "output path is required"}
	}
	if strings.ContainsRune(out, 0) {
		return Result{}, &Error{Code: ErrorCodeInvalidInput, Message: "output path contains NUL"}
	}
	sampleRate := req.SampleRate
	if sampleRate <= 0 {
		sampleRate = s.SampleRate
	}
	if sampleRate <= 0 {
		sampleRate = defaultSampleRate
	}
	duration := fixtureDuration(text, req.Speed)
	samples := synthesizeFixturePCM(text, sampleRate, duration)
	if err := os.MkdirAll(filepath.Dir(filepath.Clean(out)), 0o700); err != nil {
		return Result{}, &Error{Code: ErrorCodeSynthesisFailed, Message: err.Error()}
	}
	if err := writePCM16WAV(out, sampleRate, samples); err != nil {
		return Result{}, &Error{Code: ErrorCodeSynthesisFailed, Message: err.Error()}
	}
	info, err := os.Stat(out)
	if err != nil {
		return Result{}, &Error{Code: ErrorCodeSynthesisFailed, Message: err.Error()}
	}
	return Result{FilePath: out, Format: "wav", SampleRate: sampleRate, Channels: defaultChannels, BitsPerSample: defaultBitsPerSample, Duration: duration, Bytes: int(info.Size())}, nil
}

func fixtureDuration(text string, speed float64) time.Duration {
	if speed <= 0 {
		speed = 1
	}
	runes := len([]rune(text))
	ms := 300 + runes*35
	if ms > 2500 {
		ms = 2500
	}
	ms = int(float64(ms) / speed)
	if ms < 180 {
		ms = 180
	}
	return time.Duration(ms) * time.Millisecond
}

func synthesizeFixturePCM(text string, sampleRate int, duration time.Duration) []int16 {
	count := int(duration.Seconds() * float64(sampleRate))
	if count < sampleRate/10 {
		count = sampleRate / 10
	}
	samples := make([]int16, count)
	seed := 0
	for _, r := range text {
		seed += int(r)
	}
	base := 180.0 + float64(seed%140)
	for i := range samples {
		t := float64(i) / float64(sampleRate)
		// Tiny formant-ish buzz: enough to be audible and deterministic, not a
		// claim of natural speech quality.
		v := 0.42*math.Sin(2*math.Pi*base*t) + 0.25*math.Sin(2*math.Pi*(base*2.4)*t) + 0.12*math.Sin(2*math.Pi*(base*3.7)*t)
		fade := 1.0
		fadeSamples := sampleRate / 40
		if fadeSamples > 0 {
			if i < fadeSamples {
				fade = float64(i) / float64(fadeSamples)
			} else if remain := len(samples) - i - 1; remain < fadeSamples {
				fade = float64(remain) / float64(fadeSamples)
			}
		}
		samples[i] = int16(v * fade * 22000)
	}
	return samples
}

func writePCM16WAV(path string, sampleRate int, samples []int16) (err error) {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	// Capture the Close error so a deferred write failure (e.g. fsync-on-close
	// on NFS / a full disk) surfaces instead of being silently dropped, leaving
	// a truncated WAV that callers would treat as a successful synthesis.
	defer func() {
		if cerr := file.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}()
	dataBytes := uint32(len(samples) * 2)
	byteRate := uint32(sampleRate * defaultChannels * defaultBitsPerSample / 8)
	blockAlign := uint16(defaultChannels * defaultBitsPerSample / 8)
	if _, err := file.Write([]byte("RIFF")); err != nil {
		return err
	}
	if err := binary.Write(file, binary.LittleEndian, uint32(36)+dataBytes); err != nil {
		return err
	}
	if _, err := file.Write([]byte("WAVEfmt ")); err != nil {
		return err
	}
	if err := binary.Write(file, binary.LittleEndian, uint32(16)); err != nil {
		return err
	}
	if err := binary.Write(file, binary.LittleEndian, uint16(1)); err != nil {
		return err
	}
	if err := binary.Write(file, binary.LittleEndian, uint16(defaultChannels)); err != nil {
		return err
	}
	if err := binary.Write(file, binary.LittleEndian, uint32(sampleRate)); err != nil {
		return err
	}
	if err := binary.Write(file, binary.LittleEndian, byteRate); err != nil {
		return err
	}
	if err := binary.Write(file, binary.LittleEndian, blockAlign); err != nil {
		return err
	}
	if err := binary.Write(file, binary.LittleEndian, uint16(defaultBitsPerSample)); err != nil {
		return err
	}
	if _, err := file.Write([]byte("data")); err != nil {
		return err
	}
	if err := binary.Write(file, binary.LittleEndian, dataBytes); err != nil {
		return err
	}
	for _, sample := range samples {
		if err := binary.Write(file, binary.LittleEndian, sample); err != nil {
			return err
		}
	}
	return nil
}
