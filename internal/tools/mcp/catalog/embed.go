package catalog

import (
	"embed"
	"io/fs"
)

//go:embed manifests/*/manifest.yaml
var manifestFiles embed.FS

func Embedded() Catalog {
	root, err := fs.Sub(manifestFiles, "manifests")
	if err != nil {
		return Catalog{diagnostics: []Diagnostic{{Kind: DiagnosticInvalid, Message: "catalog unavailable"}}}
	}
	return Load(root)
}
