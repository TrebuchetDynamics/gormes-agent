package gormescli

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// SanitizeTermuxExecArgs removes the duplicate executable path that
// termux-exec injects into os.Args on Android (termux-app#4630).
func SanitizeTermuxExecArgs(args []string) []string {
	if runtime.GOOS != "android" || len(args) < 1 {
		return args
	}
	exe, err := os.Executable()
	if err != nil {
		return args
	}
	return SanitizeTermuxExecArgsWithExe(args, exe)
}

func SanitizeTermuxExecArgsWithExe(args []string, exe string) []string {
	if len(args) < 1 || exe == "" {
		return args
	}
	if termuxExecArgMatchesExecutable(args[0], exe) {
		return args[1:]
	}
	if len(args) > 1 && termuxExecArgMatchesExecutable(args[1], exe) {
		return append([]string{args[0]}, args[2:]...)
	}
	return args
}

func termuxExecArgMatchesExecutable(arg string, exe string) bool {
	if arg == "" || exe == "" {
		return false
	}
	arg = filepath.Clean(arg)
	exe = filepath.Clean(exe)
	if arg == exe {
		return true
	}
	return normalizeTermuxDataAlias(arg) == normalizeTermuxDataAlias(exe)
}

func normalizeTermuxDataAlias(path string) string {
	const dataDataPrefix = "/data/data/com.termux/"
	if strings.HasPrefix(path, dataDataPrefix) {
		return "/data/user/0/com.termux/" + strings.TrimPrefix(path, dataDataPrefix)
	}
	return path
}
