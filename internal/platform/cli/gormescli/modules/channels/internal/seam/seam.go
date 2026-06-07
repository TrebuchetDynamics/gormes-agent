package seam

import "fmt"

func Missing(name string) error {
	return fmt.Errorf("%s seam is not configured", name)
}
