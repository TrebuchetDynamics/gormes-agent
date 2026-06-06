package runtime

import (
	"errors"
	"fmt"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/whisper/pathredact"
)

func redactTranscriberError(err error, paths ...string) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s", pathredact.Text(err.Error(), paths...))
}

func errorsJoin(errs []error) error {
	var parts []string
	for _, err := range errs {
		if err != nil {
			parts = append(parts, err.Error())
		}
	}
	return errors.New(strings.Join(parts, "; "))
}
