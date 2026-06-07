package markerfile

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"
)

// CurrentTime returns the injected clock in UTC, falling back to time.Now.
func CurrentTime(now func() time.Time) time.Time {
	if now != nil {
		return now().UTC()
	}
	return time.Now().UTC()
}

// PositiveDuration returns configured when it is positive, otherwise fallback.
func PositiveDuration(configured, fallback time.Duration) time.Duration {
	if configured > 0 {
		return configured
	}
	return fallback
}

// Clear removes a marker file while treating missing files as already clear.
func Clear(ctx context.Context, path, description string) error {
	if path == "" {
		return nil
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove %s: %w", description, err)
	}
	return nil
}
