package tuigateway

import (
	"bufio"
	"errors"
	"image"
	_ "image/gif"  // header decoders for ReadImageMetadata
	_ "image/jpeg" // header decoders for ReadImageMetadata
	_ "image/png"  // header decoders for ReadImageMetadata
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

// imageHeaderReadCap bounds the number of bytes ReadImageMetadata feeds to
// image.DecodeConfig. PNG/JPEG/GIF dimensions live in the first few hundred
// bytes; 64 KiB is enough for any practical header while keeping the read
// O(1) regardless of the underlying file size. Mirrors the bounded-read
// invariant in the task contract: no full-image decode happens here.
const imageHeaderReadCap = 64 * 1024

// EstimateImageTokens mirrors hermes-agent/tui_gateway/server.py:
// _estimate_image_tokens. It returns a rough cross-provider token estimate
// for an image of the given pixel dimensions:
//
//   - 0 or negative width/height collapse to 0 tokens (the upstream guard).
//   - Otherwise the image is tiled into 512px squares using ceil-division
//     and each tile is worth 85 tokens.
//
// The helper is pure: no allocations, no I/O, no global state.
func EstimateImageTokens(width, height int) int {
	if width <= 0 || height <= 0 {
		return 0
	}
	tilesW := (width + 511) / 512
	tilesH := (height + 511) / 512
	if tilesW < 1 {
		tilesW = 1
	}
	if tilesH < 1 {
		tilesH = 1
	}
	return tilesW * tilesH * 85
}

// ImageMetadata is the structured shape returned by ReadImageMetadata.
// Mirrors the JSON keys upstream's _image_meta surfaces over JSON-RPC, but
// declared here so callers can compose it into transport-layer events
// without depending on the transport itself.
type ImageMetadata struct {
	Name          string `json:"name"`
	Width         int    `json:"width,omitempty"`
	Height        int    `json:"height,omitempty"`
	TokenEstimate int    `json:"token_estimate,omitempty"`
}

// ReadImageMetadata mirrors hermes-agent/tui_gateway/server.py:_image_meta:
// it reads only the image header to extract pixel dimensions and computes a
// token estimate, returning a metadata struct ready for transport.
//
// Behavioural contract:
//
//   - The file is opened with os.Open and read through an io.LimitReader
//     capped at imageHeaderReadCap bytes — no full-image decode happens, so
//     a 12 MiB pseudo-PNG with a junk body still completes in O(header).
//   - Recognised formats are PNG, JPEG, and GIF (the std-library default
//     decoders registered above). Anything image.DecodeConfig cannot parse
//     falls back to the upstream "name only" branch (zero dimensions and
//     zero token_estimate, no error).
//   - A genuinely missing file returns the underlying *fs.PathError so
//     callers can surface a typed I/O failure upstream; only decode errors
//     are swallowed, mirroring the upstream `except Exception: pass`
//     scope.
//
// The helper performs no network I/O, never starts goroutines, and does
// not consult any process-wide globals.
func ReadImageMetadata(path string) (ImageMetadata, error) {
	meta := ImageMetadata{Name: filepath.Base(path)}

	f, err := os.Open(path)
	if err != nil {
		var pe *fs.PathError
		if errors.As(err, &pe) {
			return ImageMetadata{}, pe
		}
		return ImageMetadata{}, err
	}
	defer f.Close()

	cfg, _, decodeErr := image.DecodeConfig(bufio.NewReader(io.LimitReader(f, imageHeaderReadCap)))
	if decodeErr != nil {
		return meta, nil
	}
	meta.Width = cfg.Width
	meta.Height = cfg.Height
	meta.TokenEstimate = EstimateImageTokens(cfg.Width, cfg.Height)
	return meta, nil
}
