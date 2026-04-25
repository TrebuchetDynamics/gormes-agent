package site

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"io/fs"
	"os"
	"path/filepath"
)

//go:embed templates/*.tmpl templates/partials/*.tmpl static/* installers/install.sh installers/install.ps1 installers/install.cmd
var siteFS embed.FS

//go:embed data/benchmarks.json
var benchmarksJSON []byte
var templateFS = siteFS

// installerSpec describes one served installer asset.
type installerSpec struct {
	Embed       string      // path inside siteFS
	ContentType string      // HTTP Content-Type when served
	ExportMode  os.FileMode // file mode used by static export
}

// installerSpecs lists every installer asset the site embeds, serves, and exports.
// Adding a new installer means adding one entry here plus the matching //go:embed line.
var installerSpecs = map[string]installerSpec{
	"install.sh":  {Embed: "installers/install.sh", ContentType: "text/x-shellscript; charset=utf-8", ExportMode: 0o755},
	"install.ps1": {Embed: "installers/install.ps1", ContentType: "text/plain; charset=utf-8", ExportMode: 0o644},
	"install.cmd": {Embed: "installers/install.cmd", ContentType: "text/plain; charset=utf-8", ExportMode: 0o644},
}

type Site struct {
	page       LandingPage
	templates  *template.Template
	static     fs.FS
	installers map[string][]byte
}

func parseTemplates() (*template.Template, error) {
	return template.ParseFS(
		siteFS,
		"templates/*.tmpl",
		"templates/partials/*.tmpl",
	)
}

func staticFS() (fs.FS, error) {
	return fs.Sub(siteFS, "static")
}

func loadInstallerAssets() (map[string][]byte, error) {
	assets := make(map[string][]byte, len(installerSpecs))
	for name, spec := range installerSpecs {
		body, err := siteFS.ReadFile(spec.Embed)
		if err != nil {
			return nil, fmt.Errorf("read embedded %s: %w", name, err)
		}
		assets[name] = body
	}
	return assets, nil
}

func loadSite() (*Site, error) {
	templates, err := parseTemplates()
	if err != nil {
		return nil, err
	}

	files, err := staticFS()
	if err != nil {
		return nil, err
	}

	installers, err := loadInstallerAssets()
	if err != nil {
		return nil, err
	}

	return &Site{
		page:       DefaultPage(),
		templates:  templates,
		static:     files,
		installers: installers,
	}, nil
}

// InstallScript returns the embedded bytes for a named installer (e.g. "install.sh").
// Returns nil if the installer is not registered.
func (s *Site) InstallScript(name string) []byte {
	return s.installers[name]
}

func (s *Site) RenderIndex() ([]byte, error) {
	var buf bytes.Buffer
	if err := s.templates.ExecuteTemplate(&buf, "layout", s.page); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func RenderIndex() ([]byte, error) {
	s, err := loadSite()
	if err != nil {
		return nil, err
	}
	return s.RenderIndex()
}

func (s *Site) ExportDir(root string) error {
	if err := os.RemoveAll(root); err != nil {
		return err
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return err
	}

	index, err := s.RenderIndex()
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(root, "index.html"), index, 0o644); err != nil {
		return err
	}

	for name, spec := range installerSpecs {
		body := s.installers[name]
		if err := os.WriteFile(filepath.Join(root, name), body, spec.ExportMode); err != nil {
			return err
		}
	}

	return fs.WalkDir(s.static, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		target := filepath.Join(root, "static", path)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}

		body, err := fs.ReadFile(s.static, path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, body, 0o644)
	})
}

func ExportDir(root string) error {
	s, err := loadSite()
	if err != nil {
		return err
	}
	return s.ExportDir(root)
}
