package tts

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const defaultPiperTimeout = 120 * time.Second

const defaultPiperRegistryBaseURL = "https://huggingface.co/rhasspy/piper-voices/resolve/main"

type PiperVoice struct {
	Name     string
	Language string
	Quality  string
	FileName string
	Path     string
	MinBytes int64
	SHA256   string
}

var builtinPiperVoices = []PiperVoice{
	{Name: "lessac-medium", Language: "en_US", Quality: "medium", FileName: "en_US-lessac-medium.onnx", Path: "en/en_US/lessac/medium/en_US-lessac-medium.onnx", MinBytes: 1024},
	{Name: "lessac-low", Language: "en_US", Quality: "low", FileName: "en_US-lessac-low.onnx", Path: "en/en_US/lessac/low/en_US-lessac-low.onnx", MinBytes: 1024},
	{Name: "amy-medium", Language: "en_US", Quality: "medium", FileName: "en_US-amy-medium.onnx", Path: "en/en_US/amy/medium/en_US-amy-medium.onnx", MinBytes: 1024},
}

type piperInstallSpec struct {
	Source        string
	MinBytes      int64
	SHA256        string
	SidecarSource string
}

type PiperCachedModelStatus struct {
	Path   string
	Usable bool
	Reason string
}

type PiperCommandRunner interface {
	RunPiper(context.Context, PiperCommandRequest) error
}

type PiperCommandRequest struct {
	Binary     string
	ModelPath  string
	OutputPath string
	Text       string
	Timeout    time.Duration
}

type PiperSynthesizer struct {
	Binary        string
	ModelPath     string
	Runner        PiperCommandRunner
	Timeout       time.Duration
	MaxTextLength int
}

func NewPiperSynthesizerFromEnv() *PiperSynthesizer {
	model := strings.TrimSpace(os.Getenv("GORMES_TTS_PIPER_MODEL"))
	if model == "" {
		model = DiscoverCachedPiperModel(os.Getenv("GORMES_TTS_PIPER_VOICE"))
	}
	if model == "" {
		return nil
	}
	binary := strings.TrimSpace(os.Getenv("GORMES_TTS_PIPER_BIN"))
	if binary == "" {
		binary = "piper"
	}
	return &PiperSynthesizer{Binary: binary, ModelPath: model, Timeout: defaultPiperTimeout, MaxTextLength: defaultMaxTextLength}
}

func DiscoverCachedPiperModel(voice string) string {
	models := CachedPiperModels()
	if len(models) == 0 {
		return ""
	}
	voice = strings.ToLower(strings.TrimSpace(voice))
	if voice != "" {
		for _, model := range models {
			if strings.Contains(strings.ToLower(filepath.Base(model)), voice) {
				return model
			}
		}
	}
	for _, preferred := range []string{"en_US-lessac-medium", "en-us-lessac-medium", "lessac", "medium"} {
		for _, model := range models {
			if strings.Contains(strings.ToLower(filepath.Base(model)), preferred) {
				return model
			}
		}
	}
	return models[0]
}

func BuiltinPiperVoices() []PiperVoice {
	out := make([]PiperVoice, len(builtinPiperVoices))
	copy(out, builtinPiperVoices)
	return out
}

func ResolvePiperModelSource(source string) string {
	return resolvePiperInstallSpec(source).Source
}

func resolvePiperInstallSpec(source string) piperInstallSpec {
	source = strings.TrimSpace(source)
	if source == "" || strings.Contains(source, "/") || strings.Contains(source, "\\") || piperSourceIsURL(source) || strings.HasSuffix(strings.ToLower(source), ".onnx") {
		return piperInstallSpec{Source: source, MinBytes: 1}
	}
	key := strings.ToLower(source)
	for _, voice := range builtinPiperVoices {
		if key == strings.ToLower(voice.Name) || key == strings.ToLower(strings.TrimSuffix(voice.FileName, ".onnx")) {
			base := strings.TrimRight(strings.TrimSpace(os.Getenv("GORMES_TTS_PIPER_REGISTRY_BASE_URL")), "/")
			if base == "" {
				base = defaultPiperRegistryBaseURL
			}
			modelSource := base + "/" + strings.TrimLeft(voice.Path, "/")
			return piperInstallSpec{Source: modelSource, MinBytes: voice.MinBytes, SHA256: voice.SHA256, SidecarSource: modelSource + ".json"}
		}
	}
	return piperInstallSpec{Source: source, MinBytes: 1}
}

func CachedPiperModels() []string {
	statuses := CachedPiperModelStatuses()
	models := make([]string, 0, len(statuses))
	for _, status := range statuses {
		if status.Usable {
			models = append(models, status.Path)
		}
	}
	return models
}

func CachedPiperModelStatuses() []PiperCachedModelStatus {
	dirs := piperCacheDirs()
	seen := map[string]bool{}
	statusByPath := map[string]PiperCachedModelStatus{}
	for _, dir := range dirs {
		dir = filepath.Clean(strings.TrimSpace(dir))
		if dir == "." || seen[dir] {
			continue
		}
		seen[dir] = true
		_ = filepath.WalkDir(dir, func(path string, entry os.DirEntry, err error) error {
			if err != nil || entry == nil || entry.IsDir() || !strings.EqualFold(filepath.Ext(path), ".onnx") {
				return nil
			}
			status := PiperCachedModelStatus{Path: path, Usable: true}
			if err := verifyPiperCachedModel(path); err != nil {
				status.Usable = false
				status.Reason = err.Error()
			}
			statusByPath[path] = status
			return nil
		})
	}
	paths := make([]string, 0, len(statusByPath))
	for path := range statusByPath {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	statuses := make([]PiperCachedModelStatus, 0, len(paths))
	for _, path := range paths {
		statuses = append(statuses, statusByPath[path])
	}
	return statuses
}

func PiperCacheDir() string {
	if dir := strings.TrimSpace(os.Getenv("GORMES_TTS_PIPER_MODEL_CACHE")); dir != "" {
		return filepath.Clean(dir)
	}
	if dir := strings.TrimSpace(os.Getenv("GORMES_TTS_PIPER_CACHE_DIR")); dir != "" {
		return filepath.Clean(dir)
	}
	if userCache, err := os.UserCacheDir(); err == nil && strings.TrimSpace(userCache) != "" {
		return filepath.Join(userCache, "gormes", "piper")
	}
	return ""
}

func piperCacheDirs() []string {
	var dirs []string
	if dir := strings.TrimSpace(os.Getenv("GORMES_TTS_PIPER_MODEL_CACHE")); dir != "" {
		dirs = append(dirs, dir)
	}
	if dir := strings.TrimSpace(os.Getenv("GORMES_TTS_PIPER_CACHE_DIR")); dir != "" {
		dirs = append(dirs, dir)
	}
	if userCache, err := os.UserCacheDir(); err == nil && strings.TrimSpace(userCache) != "" {
		dirs = append(dirs, filepath.Join(userCache, "gormes", "piper"))
	}
	return dirs
}

func InstallPiperModel(ctx context.Context, source string) (string, error) {
	cacheDir := PiperCacheDir()
	if cacheDir == "" {
		return "", errors.New("Piper model cache directory is unavailable")
	}
	source = strings.TrimSpace(source)
	if source == "" {
		return "", errors.New("Piper model source is required")
	}
	spec := resolvePiperInstallSpec(source)
	source = spec.Source
	name, err := piperModelFileName(source)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(cacheDir, 0o700); err != nil {
		return "", err
	}
	dest := filepath.Join(cacheDir, name)
	if piperSourceIsURL(source) {
		if err := downloadPiperModel(ctx, source, dest, spec.MinBytes, spec.SHA256); err != nil {
			return dest, err
		}
		if spec.SidecarSource != "" {
			if err := downloadPiperSidecar(ctx, spec.SidecarSource, dest+".json"); err != nil {
				_ = os.Remove(dest)
				return dest, err
			}
		}
		return dest, nil
	}
	if err := copyPiperModelFile(source, dest); err != nil {
		return dest, err
	}
	if err := verifyPiperModelFile(dest, spec.MinBytes, spec.SHA256); err != nil {
		_ = os.Remove(dest)
		return dest, err
	}
	return dest, nil
}

func IsPiperModelUsable(path string) bool {
	return verifyPiperCachedModel(path) == nil
}

func RemoveUnusablePiperModels() ([]string, error) {
	statuses := CachedPiperModelStatuses()
	var removed []string
	var errs []string
	for _, status := range statuses {
		if status.Usable {
			continue
		}
		if err := os.Remove(status.Path); err != nil && !os.IsNotExist(err) {
			errs = append(errs, fmt.Sprintf("%s: %v", status.Path, err))
			continue
		}
		removed = append(removed, status.Path)
	}
	if len(errs) > 0 {
		return removed, errors.New(strings.Join(errs, "; "))
	}
	return removed, nil
}

func piperModelFileName(source string) (string, error) {
	name := filepath.Base(source)
	if piperSourceIsURL(source) {
		parsed, err := url.Parse(source)
		if err != nil {
			return "", err
		}
		name = filepath.Base(parsed.Path)
	}
	name = strings.TrimSpace(name)
	if name == "" || name == "." || name == string(filepath.Separator) {
		return "", errors.New("Piper model source must name an .onnx file")
	}
	if !strings.EqualFold(filepath.Ext(name), ".onnx") {
		return "", errors.New("Piper model source must be an .onnx file")
	}
	return name, nil
}

func piperSourceIsURL(source string) bool {
	parsed, err := url.Parse(source)
	return err == nil && (parsed.Scheme == "http" || parsed.Scheme == "https") && parsed.Host != ""
}

func copyPiperModelFile(source, dest string) error {
	in, err := os.Open(source)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(out, in)
	closeErr := out.Close()
	if copyErr != nil {
		return copyErr
	}
	return closeErr
}

func downloadPiperSidecar(ctx context.Context, source, dest string) error {
	if err := downloadFile(ctx, source, dest); err != nil {
		return err
	}
	info, err := os.Stat(dest)
	if err != nil {
		return err
	}
	if info.Size() == 0 {
		_ = os.Remove(dest)
		return fmt.Errorf("Piper sidecar %s is empty", filepath.Base(dest))
	}
	return nil
}

func downloadPiperModel(ctx context.Context, source, dest string, minBytes int64, wantSHA256 string) error {
	if err := downloadFile(ctx, source, dest); err != nil {
		return err
	}
	if err := verifyPiperModelFile(dest, minBytes, wantSHA256); err != nil {
		_ = os.Remove(dest)
		return err
	}
	return nil
}

func downloadFile(ctx context.Context, source, dest string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, source, nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("download Piper model: HTTP %d", resp.StatusCode)
	}
	out, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(out, resp.Body)
	closeErr := out.Close()
	if copyErr != nil {
		return copyErr
	}
	return closeErr
}

func verifyPiperCachedModel(path string) error {
	if err := verifyPiperModelFile(path, 1, ""); err != nil {
		return err
	}
	base := filepath.Base(path)
	for _, voice := range builtinPiperVoices {
		if strings.EqualFold(base, voice.FileName) {
			info, err := os.Stat(path + ".json")
			if err != nil {
				return fmt.Errorf("Piper model %s is missing sidecar %s", base, base+".json")
			}
			if info.Size() == 0 {
				return fmt.Errorf("Piper sidecar %s is empty", base+".json")
			}
		}
	}
	return nil
}

func verifyPiperModelFile(path string, minBytes int64, wantSHA256 string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if minBytes <= 0 {
		minBytes = 1
	}
	if info.Size() < minBytes {
		return fmt.Errorf("Piper model %s is too small: %d bytes < %d", filepath.Base(path), info.Size(), minBytes)
	}
	wantSHA256 = strings.ToLower(strings.TrimSpace(wantSHA256))
	if wantSHA256 == "" {
		return nil
	}
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return err
	}
	got := hex.EncodeToString(hash.Sum(nil))
	if got != wantSHA256 {
		return fmt.Errorf("Piper model %s checksum mismatch", filepath.Base(path))
	}
	return nil
}

func (s *PiperSynthesizer) Synthesize(ctx context.Context, req Request) (Result, error) {
	if err := ctx.Err(); err != nil {
		return Result{}, &Error{Code: ErrorCodeProviderUnavailable, Message: err.Error()}
	}
	if s == nil {
		return Result{}, &Error{Code: ErrorCodeProviderUnavailable, Message: "Piper TTS runtime is not configured"}
	}
	binary := strings.TrimSpace(s.Binary)
	if binary == "" {
		binary = "piper"
	}
	modelPath := strings.TrimSpace(firstNonEmpty(req.Voice, s.ModelPath))
	if modelPath == "" {
		return Result{}, &Error{Code: ErrorCodeProviderUnavailable, Message: "Piper TTS model is not configured"}
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
		return Result{}, &Error{Code: ErrorCodeInvalidInput, Message: fmt.Sprintf("text exceeds Piper TTS limit of %d characters", maxLen)}
	}
	out := strings.TrimSpace(req.OutputPath)
	if out == "" {
		return Result{}, &Error{Code: ErrorCodeInvalidInput, Message: "output path is required"}
	}
	if strings.ContainsRune(out, 0) {
		return Result{}, &Error{Code: ErrorCodeInvalidInput, Message: "output path contains NUL"}
	}
	if err := os.MkdirAll(filepath.Dir(filepath.Clean(out)), 0o700); err != nil {
		return Result{}, &Error{Code: ErrorCodeSynthesisFailed, Message: err.Error()}
	}
	if err := os.Remove(out); err != nil && !os.IsNotExist(err) {
		return Result{}, &Error{Code: ErrorCodeSynthesisFailed, Message: err.Error()}
	}
	runner := s.Runner
	if runner == nil {
		runner = shellPiperCommandRunner{}
	}
	timeout := s.Timeout
	if timeout <= 0 {
		timeout = defaultPiperTimeout
	}
	if err := runner.RunPiper(ctx, PiperCommandRequest{Binary: binary, ModelPath: modelPath, OutputPath: out, Text: text, Timeout: timeout}); err != nil {
		return Result{}, &Error{Code: ErrorCodeSynthesisFailed, Message: err.Error()}
	}
	info, err := os.Stat(out)
	if err != nil {
		return Result{}, &Error{Code: ErrorCodeSynthesisFailed, Message: err.Error()}
	}
	if info.Size() == 0 {
		return Result{}, &Error{Code: ErrorCodeSynthesisFailed, Message: "Piper TTS produced empty output"}
	}
	return Result{FilePath: out, Format: strings.TrimPrefix(strings.ToLower(filepath.Ext(out)), "."), SampleRate: req.SampleRate, Channels: defaultChannels, BitsPerSample: defaultBitsPerSample, Bytes: int(info.Size())}, nil
}

type shellPiperCommandRunner struct{}

func (shellPiperCommandRunner) RunPiper(ctx context.Context, req PiperCommandRequest) error {
	if strings.TrimSpace(req.Binary) == "" || strings.TrimSpace(req.ModelPath) == "" {
		return errors.New("Piper binary and model are required")
	}
	timeout := req.Timeout
	if timeout <= 0 {
		timeout = defaultPiperTimeout
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, req.Binary, "--model", req.ModelPath, "--output_file", req.OutputPath)
	cmd.Stdin = strings.NewReader(req.Text)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if ctx.Err() != nil {
			return fmt.Errorf("Piper TTS timed out after %s: %w", timeout, ctx.Err())
		}
		msg := strings.TrimSpace(stderr.String())
		if msg != "" {
			return fmt.Errorf("Piper TTS failed: %s", msg)
		}
		return fmt.Errorf("Piper TTS failed: %w", err)
	}
	return nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
