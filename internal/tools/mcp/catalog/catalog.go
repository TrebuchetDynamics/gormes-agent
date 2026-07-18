package catalog

import (
	"errors"
	"io"
	"io/fs"
	"path"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	ManifestVersion     = 1
	maxManifestFileSize = 256 << 10

	TransportStdio TransportType = "stdio"
	TransportHTTP  TransportType = "http"
	AuthAPIKey     AuthType      = "api_key"
	AuthOAuth      AuthType      = "oauth"
	AuthNone       AuthType      = "none"

	DiagnosticInvalid        DiagnosticKind = "invalid"
	DiagnosticFutureManifest DiagnosticKind = "future_manifest"
	DiagnosticDuplicate      DiagnosticKind = "duplicate"
)

var (
	entryNamePattern = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)
	envNamePattern   = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
)

type TransportType string
type AuthType string
type DiagnosticKind string

type EnvVar struct {
	Name     string `json:"name"`
	Prompt   string `json:"prompt"`
	Required bool   `json:"required"`
	Secret   bool   `json:"secret"`
	Default  string `json:"default,omitempty"`
}

type Auth struct {
	Type     AuthType `json:"type"`
	Env      []EnvVar `json:"env,omitempty"`
	Provider string   `json:"provider,omitempty"`
	Scopes   []string `json:"scopes,omitempty"`
	EnvVar   string   `json:"env_var,omitempty"`
}

type Transport struct {
	Type    TransportType `json:"type"`
	Command string        `json:"command,omitempty"`
	Args    []string      `json:"args,omitempty"`
	URL     string        `json:"url,omitempty"`
	Version string        `json:"version,omitempty"`
}

type Install struct {
	Type      string   `json:"type"`
	URL       string   `json:"url"`
	Ref       string   `json:"ref"`
	Bootstrap []string `json:"bootstrap,omitempty"`
}

type Tools struct {
	DefaultEnabled []string `json:"default_enabled,omitempty"`
}

type Entry struct {
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Source      string    `json:"source,omitempty"`
	Transport   Transport `json:"transport"`
	Auth        Auth      `json:"auth"`
	Tools       Tools     `json:"tools,omitempty"`
	Install     *Install  `json:"install,omitempty"`
	PostInstall string    `json:"post_install,omitempty"`
}

type Diagnostic struct {
	Entry   string         `json:"entry"`
	Kind    DiagnosticKind `json:"kind"`
	Message string         `json:"message"`
}

type Catalog struct {
	entries     []Entry
	diagnostics []Diagnostic
}

func Load(fsys fs.FS) Catalog {
	if fsys == nil {
		return Catalog{diagnostics: []Diagnostic{{Kind: DiagnosticInvalid, Message: "catalog unavailable"}}}
	}

	var parsed []Entry
	var diagnostics []Diagnostic
	_ = fs.WalkDir(fsys, ".", func(filePath string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			diagnostics = append(diagnostics, invalidDiagnostic(path.Base(path.Dir(filePath))))
			return nil
		}
		if entry.IsDir() || path.Base(filePath) != "manifest.yaml" {
			return nil
		}
		hint := path.Base(path.Dir(filePath))
		raw, err := readManifest(fsys, filePath)
		if err != nil {
			diagnostics = append(diagnostics, invalidDiagnostic(hint))
			return nil
		}
		item, failure := parseManifest(raw)
		if failure != nil {
			diagnostics = append(diagnostics, Diagnostic{Entry: hint, Kind: failure.kind, Message: failure.message})
			return nil
		}
		parsed = append(parsed, item)
		return nil
	})

	counts := make(map[string]int, len(parsed))
	for _, item := range parsed {
		counts[item.Name]++
	}
	entries := make([]Entry, 0, len(parsed))
	for _, item := range parsed {
		if counts[item.Name] > 1 {
			continue
		}
		entries = append(entries, item)
	}
	for name, count := range counts {
		if count > 1 {
			diagnostics = append(diagnostics, Diagnostic{Entry: name, Kind: DiagnosticDuplicate, Message: "duplicate catalog entry omitted"})
		}
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name < entries[j].Name })
	sort.Slice(diagnostics, func(i, j int) bool {
		if diagnostics[i].Entry != diagnostics[j].Entry {
			return diagnostics[i].Entry < diagnostics[j].Entry
		}
		return diagnostics[i].Kind < diagnostics[j].Kind
	})
	return Catalog{entries: entries, diagnostics: diagnostics}
}

func (catalog Catalog) List() []Entry {
	out := make([]Entry, len(catalog.entries))
	for i, item := range catalog.entries {
		out[i] = cloneEntry(item)
	}
	return out
}

func (catalog Catalog) Get(identifier string) (Entry, bool) {
	name := strings.TrimSpace(identifier)
	name = strings.TrimPrefix(name, "official/")
	index := sort.Search(len(catalog.entries), func(i int) bool { return catalog.entries[i].Name >= name })
	if index >= len(catalog.entries) || catalog.entries[index].Name != name {
		return Entry{}, false
	}
	return cloneEntry(catalog.entries[index]), true
}

func (catalog Catalog) Diagnostics() []Diagnostic {
	out := make([]Diagnostic, len(catalog.diagnostics))
	copy(out, catalog.diagnostics)
	return out
}

func readManifest(fsys fs.FS, name string) ([]byte, error) {
	file, err := fsys.Open(name)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	raw, err := io.ReadAll(io.LimitReader(file, maxManifestFileSize+1))
	if err != nil {
		return nil, err
	}
	if len(raw) > maxManifestFileSize {
		return nil, errors.New("manifest too large")
	}
	return raw, nil
}

type rawManifest struct {
	ManifestVersion int          `yaml:"manifest_version"`
	Name            string       `yaml:"name"`
	Description     string       `yaml:"description"`
	Source          string       `yaml:"source"`
	Transport       rawTransport `yaml:"transport"`
	Auth            rawAuth      `yaml:"auth"`
	Tools           rawTools     `yaml:"tools"`
	Install         *rawInstall  `yaml:"install"`
	PostInstall     string       `yaml:"post_install"`
}

type rawTransport struct {
	Type    string   `yaml:"type"`
	Command string   `yaml:"command"`
	Args    []string `yaml:"args"`
	URL     string   `yaml:"url"`
	Version string   `yaml:"version"`
}

type rawAuth struct {
	Type     string      `yaml:"type"`
	Env      []rawEnvVar `yaml:"env"`
	Provider string      `yaml:"provider"`
	Scopes   []string    `yaml:"scopes"`
	EnvVar   string      `yaml:"env_var"`
}

type rawEnvVar struct {
	Name     string `yaml:"name"`
	Prompt   string `yaml:"prompt"`
	Required *bool  `yaml:"required"`
	Secret   *bool  `yaml:"secret"`
	Default  string `yaml:"default"`
}

type rawTools struct {
	DefaultEnabled []string `yaml:"default_enabled"`
}

type rawInstall struct {
	Type      string   `yaml:"type"`
	URL       string   `yaml:"url"`
	Ref       string   `yaml:"ref"`
	Bootstrap []string `yaml:"bootstrap"`
}

type manifestFailure struct {
	kind    DiagnosticKind
	message string
}

func parseManifest(raw []byte) (Entry, *manifestFailure) {
	var manifest rawManifest
	if err := yaml.Unmarshal(raw, &manifest); err != nil {
		return Entry{}, invalidFailure()
	}
	if manifest.ManifestVersion != ManifestVersion {
		kind := DiagnosticInvalid
		message := "unsupported manifest version"
		if manifest.ManifestVersion > ManifestVersion {
			kind = DiagnosticFutureManifest
			message = "requires a newer Gormes catalog"
		}
		return Entry{}, &manifestFailure{kind: kind, message: message}
	}
	manifest.Name = strings.TrimSpace(manifest.Name)
	manifest.Description = strings.TrimSpace(manifest.Description)
	if !entryNamePattern.MatchString(manifest.Name) || manifest.Description == "" {
		return Entry{}, invalidFailure()
	}

	transport := Transport{
		Type:    TransportType(strings.TrimSpace(manifest.Transport.Type)),
		Command: strings.TrimSpace(manifest.Transport.Command),
		Args:    append([]string(nil), manifest.Transport.Args...),
		URL:     strings.TrimSpace(manifest.Transport.URL),
		Version: strings.TrimSpace(manifest.Transport.Version),
	}
	switch transport.Type {
	case TransportStdio:
		if transport.Command == "" {
			return Entry{}, invalidFailure()
		}
	case TransportHTTP:
		if transport.URL == "" {
			return Entry{}, invalidFailure()
		}
	default:
		return Entry{}, invalidFailure()
	}

	authType := AuthType(strings.TrimSpace(manifest.Auth.Type))
	if authType == "" {
		authType = AuthNone
	}
	if authType != AuthAPIKey && authType != AuthOAuth && authType != AuthNone {
		return Entry{}, invalidFailure()
	}
	auth := Auth{
		Type:     authType,
		Provider: strings.TrimSpace(manifest.Auth.Provider),
		Scopes:   append([]string(nil), manifest.Auth.Scopes...),
		EnvVar:   strings.TrimSpace(manifest.Auth.EnvVar),
	}
	for _, spec := range manifest.Auth.Env {
		spec.Name = strings.TrimSpace(spec.Name)
		if !envNamePattern.MatchString(spec.Name) {
			return Entry{}, invalidFailure()
		}
		required := true
		if spec.Required != nil {
			required = *spec.Required
		}
		secret := true
		if spec.Secret != nil {
			secret = *spec.Secret
		}
		prompt := strings.TrimSpace(spec.Prompt)
		if prompt == "" {
			prompt = spec.Name
		}
		auth.Env = append(auth.Env, EnvVar{Name: spec.Name, Prompt: prompt, Required: required, Secret: secret, Default: spec.Default})
	}

	var install *Install
	if manifest.Install != nil {
		if strings.TrimSpace(manifest.Install.Type) != "git" || strings.TrimSpace(manifest.Install.URL) == "" || strings.TrimSpace(manifest.Install.Ref) == "" {
			return Entry{}, invalidFailure()
		}
		install = &Install{
			Type:      "git",
			URL:       strings.TrimSpace(manifest.Install.URL),
			Ref:       strings.TrimSpace(manifest.Install.Ref),
			Bootstrap: append([]string(nil), manifest.Install.Bootstrap...),
		}
	}

	return Entry{
		Name:        manifest.Name,
		Description: manifest.Description,
		Source:      strings.TrimSpace(manifest.Source),
		Transport:   transport,
		Auth:        auth,
		Tools:       Tools{DefaultEnabled: append([]string(nil), manifest.Tools.DefaultEnabled...)},
		Install:     install,
		PostInstall: strings.TrimSpace(manifest.PostInstall),
	}, nil
}

func invalidFailure() *manifestFailure {
	return &manifestFailure{kind: DiagnosticInvalid, message: "manifest invalid"}
}

func invalidDiagnostic(entry string) Diagnostic {
	return Diagnostic{Entry: entry, Kind: DiagnosticInvalid, Message: "manifest invalid"}
}

func cloneEntry(item Entry) Entry {
	item.Transport.Args = append([]string(nil), item.Transport.Args...)
	item.Auth.Scopes = append([]string(nil), item.Auth.Scopes...)
	item.Auth.Env = append([]EnvVar(nil), item.Auth.Env...)
	item.Tools.DefaultEnabled = append([]string(nil), item.Tools.DefaultEnabled...)
	if item.Install != nil {
		copy := *item.Install
		copy.Bootstrap = append([]string(nil), item.Install.Bootstrap...)
		item.Install = &copy
	}
	return item
}

func (kind DiagnosticKind) String() string { return string(kind) }

func (transport TransportType) String() string { return string(transport) }

func (auth AuthType) String() string { return string(auth) }
