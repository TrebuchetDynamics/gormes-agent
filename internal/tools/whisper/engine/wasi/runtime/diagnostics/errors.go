package diagnostics

import (
	"errors"
	"fmt"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/whisper/pathredact"
)

func RedactTranscriberError(err error, paths ...string) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s", pathredact.Text(err.Error(), paths...))
}

func Join(errs []error) error {
	var parts []string
	for _, err := range errs {
		if err != nil {
			parts = append(parts, err.Error())
		}
	}
	return errors.New(strings.Join(parts, "; "))
}
