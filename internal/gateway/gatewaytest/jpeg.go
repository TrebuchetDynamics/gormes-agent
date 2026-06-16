package gatewaytest

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"os"
	"path/filepath"
	"testing"
)

// WriteFixtureJPEG creates a tiny synthetic JPEG so tests stay hermetic without
// committing binary fixtures. Returns the on-disk path in dir.
func WriteFixtureJPEG(t *testing.T, dir, name string, fillR, fillG, fillB uint8) string {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 4, 4))
	fill := color.RGBA{R: fillR, G: fillG, B: fillB, A: 255}
	for y := 0; y < 4; y++ {
		for x := 0; x < 4; x++ {
			img.Set(x, y, fill)
		}
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 90}); err != nil {
		t.Fatalf("encode fixture jpeg: %v", err)
	}
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, buf.Bytes(), 0o600); err != nil {
		t.Fatalf("write fixture jpeg: %v", err)
	}
	return path
}
