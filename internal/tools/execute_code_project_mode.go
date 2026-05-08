package tools

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// ProjectModeCWDInput captures Hermes' project-mode cwd decision without
// reading process state directly. Tests inject directory predicates so the
// resolver never depends on a developer's live terminal session.
type ProjectModeCWDInput struct {
	TerminalCWD string
	ProcessCWD  string
	StagingDir  string
	IsDir       func(string) bool
}

func ResolveProjectModeCWD(input ProjectModeCWDInput) string {
	isDir := input.IsDir
	if isDir == nil {
		isDir = projectModeDirExists
	}
	if cwd := normalizeProjectModePath(input.TerminalCWD); cwd != "" && isDir(cwd) {
		return cwd
	}
	if cwd := normalizeProjectModePath(input.ProcessCWD); cwd != "" && isDir(cwd) {
		return cwd
	}
	return input.StagingDir
}

// ProjectModePythonInput freezes Hermes' active virtualenv/conda interpreter
// selection as a pure helper. Gormes still keeps Python execution disabled in
// execute_code; this helper exists for parity evidence and future wiring.
type ProjectModePythonInput struct {
	Env             map[string]string
	SystemPython    string
	IsExecutable    func(string) bool
	IsUsablePython  func(string) bool
	WindowsPlatform bool
}

func ResolveProjectModePython(input ProjectModePythonInput) string {
	fallback := strings.TrimSpace(input.SystemPython)
	if fallback == "" {
		fallback = "python"
	}
	isExecutable := input.IsExecutable
	if isExecutable == nil {
		isExecutable = projectModeExecutableFile
	}
	isUsable := input.IsUsablePython
	if isUsable == nil {
		isUsable = func(string) bool { return true }
	}

	subdirs := []string{"bin"}
	exes := []string{"python", "python3"}
	if input.WindowsPlatform {
		subdirs = []string{"Scripts"}
		exes = []string{"python.exe", "python3.exe"}
	}

	for _, key := range []string{"VIRTUAL_ENV", "CONDA_PREFIX"} {
		root := strings.TrimSpace(input.Env[key])
		if root == "" {
			continue
		}
		for _, subdir := range subdirs {
			for _, exe := range exes {
				candidate := filepath.Join(root, subdir, exe)
				if !isExecutable(candidate) {
					continue
				}
				if isUsable(candidate) {
					return candidate
				}
				return fallback
			}
		}
	}
	return fallback
}

type ProjectModeSandbox struct {
	inner *LocalCodeSandbox
}

type projectModeSandboxHooks struct {
	lookupEnv func(string) (string, bool)
	getwd     func() (string, error)
	isDir     func(string) bool
	lookPath  func(string) (string, error)
}

func NewProjectModeSandbox() *ProjectModeSandbox {
	return newProjectModeSandboxWithHooks(projectModeSandboxHooks{
		lookupEnv: os.LookupEnv,
		getwd:     os.Getwd,
		isDir:     projectModeDirExists,
		lookPath:  exec.LookPath,
	})
}

func newProjectModeSandboxWithHooks(h projectModeSandboxHooks) *ProjectModeSandbox {
	if h.lookupEnv == nil {
		h.lookupEnv = os.LookupEnv
	}
	if h.getwd == nil {
		h.getwd = os.Getwd
	}
	if h.isDir == nil {
		h.isDir = projectModeDirExists
	}
	if h.lookPath == nil {
		h.lookPath = exec.LookPath
	}
	workdir := func(stagingDir string) string {
		terminalCWD, _ := h.lookupEnv("TERMINAL_CWD")
		processCWD, err := h.getwd()
		if err != nil {
			processCWD = ""
		}
		return ResolveProjectModeCWD(ProjectModeCWDInput{
			TerminalCWD: terminalCWD,
			ProcessCWD:  processCWD,
			StagingDir:  stagingDir,
			IsDir:       h.isDir,
		})
	}
	return &ProjectModeSandbox{
		inner: &LocalCodeSandbox{
			lookPath: h.lookPath,
			languages: map[string]runtimeSpec{
				"sh":    {Binaries: []string{"sh"}, Extension: ".sh"},
				"shell": {Binaries: []string{"sh"}, Extension: ".sh"},
			},
			workdir: workdir,
		},
	}
}

func (s *ProjectModeSandbox) Execute(ctx context.Context, req CodeExecutionRequest) (CodeExecutionResult, error) {
	return s.inner.Execute(ctx, req)
}

func normalizeProjectModePath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	if strings.HasPrefix(path, "~") {
		if expanded, err := expandUserPath(path); err == nil {
			return expanded
		}
	}
	return path
}

func projectModeDirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func projectModeExecutableFile(path string) bool {
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return false
	}
	if runtime.GOOS == "windows" {
		return true
	}
	return info.Mode()&0o111 != 0
}
