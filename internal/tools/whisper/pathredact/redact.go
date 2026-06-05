package pathredact

import (
	"errors"
	"path/filepath"
	"strings"
)

func Error(err error, paths ...string) error {
	if err == nil {
		return nil
	}
	return errors.New(Text(err.Error(), paths...))
}

func Text(text string, paths ...string) string {
	redacted := text
	for _, path := range paths {
		if path == "" {
			continue
		}
		redacted = strings.ReplaceAll(redacted, path, filepath.Base(path))
	}
	return redacted
}
