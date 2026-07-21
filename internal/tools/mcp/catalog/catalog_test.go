package catalog

import (
	"bytes"
	"testing"
	"testing/fstest"
)

func TestLoadParsesAndSortsApprovedManifests(t *testing.T) {
	catalog := Load(fstest.MapFS{
		"zeta/manifest.yaml": {Data: []byte(`
manifest_version: 1
name: zeta
description: Zeta MCP
source: https://example.com/zeta
transport:
  type: stdio
  command: npx
  args: ["-y", "zeta-mcp"]
auth:
  type: api_key
  env:
    - name: ZETA_API_KEY
      prompt: Zeta API key
      required: true
      secret: true
tools:
  default_enabled: [search, get]
`)},
		"alpha/manifest.yaml": {Data: []byte(`
manifest_version: 1
name: alpha
description: Alpha MCP
source: https://example.com/alpha
transport:
  type: http
  url: https://mcp.example.com/mcp
auth:
  type: oauth
  scopes: [read, write]
  env_var: ALPHA_TOKEN
post_install: Restart your session.
`)},
	})

	entries := catalog.List()
	if len(entries) != 2 {
		t.Fatalf("List() count = %d, want 2; diagnostics=%+v", len(entries), catalog.Diagnostics())
	}
	if entries[0].Name != "alpha" || entries[1].Name != "zeta" {
		t.Fatalf("List() names = %q, %q; want alpha, zeta", entries[0].Name, entries[1].Name)
	}
	if entries[0].Transport.Type != TransportHTTP || entries[0].Transport.URL != "https://mcp.example.com/mcp" {
		t.Fatalf("alpha transport = %+v", entries[0].Transport)
	}
	if entries[0].Auth.Type != AuthOAuth || entries[0].Auth.EnvVar != "ALPHA_TOKEN" || len(entries[0].Auth.Scopes) != 2 {
		t.Fatalf("alpha auth = %+v", entries[0].Auth)
	}
	if entries[1].Transport.Type != TransportStdio || entries[1].Transport.Command != "npx" || len(entries[1].Transport.Args) != 2 {
		t.Fatalf("zeta transport = %+v", entries[1].Transport)
	}
	if len(entries[1].Auth.Env) != 1 || entries[1].Auth.Env[0].Name != "ZETA_API_KEY" {
		t.Fatalf("zeta auth = %+v", entries[1].Auth)
	}
	if got, ok := catalog.Get("official/alpha"); !ok || got.Name != "alpha" {
		t.Fatalf("Get(official/alpha) = %+v, %v", got, ok)
	}
	if len(catalog.Diagnostics()) != 0 {
		t.Fatalf("Diagnostics() = %+v, want empty", catalog.Diagnostics())
	}
}

func TestLoadSkipsInvalidAndFutureManifestsWithDiagnostics(t *testing.T) {
	catalog := Load(fstest.MapFS{
		"valid/manifest.yaml": {Data: []byte(validHTTPManifest("valid"))},
		"future/manifest.yaml": {Data: []byte(`
manifest_version: 99
name: future
description: Future
transport: {type: http, url: https://example.com}
`)},
		"broken/manifest.yaml": {Data: []byte(`manifest_version: [`)},
	})

	entries := catalog.List()
	if len(entries) != 1 || entries[0].Name != "valid" {
		t.Fatalf("List() = %+v, want only valid", entries)
	}
	diagnostics := catalog.Diagnostics()
	if len(diagnostics) != 2 {
		t.Fatalf("Diagnostics() = %+v, want 2", diagnostics)
	}
	if diagnostics[0].Entry != "broken" || diagnostics[0].Kind != DiagnosticInvalid {
		t.Fatalf("first diagnostic = %+v, want broken/invalid", diagnostics[0])
	}
	if diagnostics[1].Entry != "future" || diagnostics[1].Kind != DiagnosticFutureManifest {
		t.Fatalf("second diagnostic = %+v, want future/future_manifest", diagnostics[1])
	}
}

func TestLoadRejectsInvalidManifestShapes(t *testing.T) {
	cases := map[string]string{
		"bad_name": `
manifest_version: 1
name: "bad name"
description: Bad
transport: {type: http, url: https://example.com}
`,
		"missing_description": `
manifest_version: 1
name: demo
transport: {type: http, url: https://example.com}
`,
		"stdio_without_command": `
manifest_version: 1
name: demo
description: Demo
transport: {type: stdio}
`,
		"http_without_url": `
manifest_version: 1
name: demo
description: Demo
transport: {type: http}
`,
		"bad_auth": `
manifest_version: 1
name: demo
description: Demo
transport: {type: http, url: https://example.com}
auth: {type: password}
`,
		"bad_env_name": `
manifest_version: 1
name: demo
description: Demo
transport: {type: stdio, command: demo}
auth:
  type: api_key
  env: [{name: "BAD-NAME"}]
`,
		"git_without_ref": `
manifest_version: 1
name: demo
description: Demo
transport: {type: stdio, command: demo}
install: {type: git, url: https://example.com/demo.git}
`,
	}
	for name, manifest := range cases {
		t.Run(name, func(t *testing.T) {
			catalog := Load(fstest.MapFS{"demo/manifest.yaml": {Data: []byte(manifest)}})
			if len(catalog.List()) != 0 {
				t.Fatalf("List() = %+v, want empty", catalog.List())
			}
			if diagnostics := catalog.Diagnostics(); len(diagnostics) != 1 || diagnostics[0].Kind != DiagnosticInvalid {
				t.Fatalf("Diagnostics() = %+v, want invalid", diagnostics)
			}
		})
	}
}

func TestLoadFailsClosedForDuplicateNamesAndOversizedFiles(t *testing.T) {
	catalog := Load(fstest.MapFS{
		"one/manifest.yaml":  {Data: []byte(validHTTPManifest("duplicate"))},
		"two/manifest.yaml":  {Data: []byte(validHTTPManifest("duplicate"))},
		"huge/manifest.yaml": {Data: bytes.Repeat([]byte("x"), maxManifestFileSize+1)},
	})
	if len(catalog.List()) != 0 {
		t.Fatalf("List() = %+v, want empty", catalog.List())
	}
	kinds := map[DiagnosticKind]bool{}
	for _, diagnostic := range catalog.Diagnostics() {
		kinds[diagnostic.Kind] = true
	}
	if !kinds[DiagnosticDuplicate] || !kinds[DiagnosticInvalid] {
		t.Fatalf("Diagnostics() = %+v, want duplicate and invalid", catalog.Diagnostics())
	}
}

func TestEmbeddedCatalogPreservesApprovedHermesMetadata(t *testing.T) {
	catalog := Embedded()
	if diagnostics := catalog.Diagnostics(); len(diagnostics) != 0 {
		t.Fatalf("Embedded diagnostics = %+v", diagnostics)
	}
	entries := catalog.List()
	if len(entries) != 3 || entries[0].Name != "linear" || entries[1].Name != "n8n" || entries[2].Name != "unreal-engine" {
		t.Fatalf("Embedded entries = %+v", entries)
	}
	linear, _ := catalog.Get("linear")
	if linear.Transport.Type != TransportHTTP || linear.Auth.Type != AuthOAuth || linear.Source != "https://linear.app/docs/mcp" {
		t.Fatalf("linear = %+v", linear)
	}
	n8n, _ := catalog.Get("n8n")
	if n8n.Transport.Type != TransportStdio || n8n.Auth.Type != AuthAPIKey || n8n.Install == nil || n8n.Install.Type != "git" || len(n8n.Tools.DefaultEnabled) != 8 {
		t.Fatalf("n8n = %+v", n8n)
	}
	unreal, _ := catalog.Get("unreal-engine")
	if unreal.Transport.Type != TransportHTTP || unreal.Auth.Type != AuthNone || unreal.Transport.URL != "http://127.0.0.1:8000/mcp" {
		t.Fatalf("unreal-engine = %+v", unreal)
	}
}

func TestCatalogListAndGetReturnDefensiveCopies(t *testing.T) {
	catalog := Load(fstest.MapFS{"demo/manifest.yaml": {Data: []byte(`
manifest_version: 1
name: demo
description: Demo
transport: {type: stdio, command: demo, args: [one]}
auth: {type: oauth, scopes: [read]}
tools: {default_enabled: [search]}
install: {type: git, url: https://example.com/demo.git, ref: v1, bootstrap: [build]}
`)}})
	entries := catalog.List()
	entries[0].Transport.Args[0] = "changed"
	entries[0].Auth.Scopes[0] = "changed"
	entries[0].Tools.DefaultEnabled[0] = "changed"
	entries[0].Install.Bootstrap[0] = "changed"
	got, _ := catalog.Get("demo")
	if got.Transport.Args[0] != "one" || got.Auth.Scopes[0] != "read" || got.Tools.DefaultEnabled[0] != "search" || got.Install.Bootstrap[0] != "build" {
		t.Fatalf("catalog mutated through List(): %+v", got)
	}
}

func validHTTPManifest(name string) string {
	return "manifest_version: 1\nname: " + name + "\ndescription: Demo\ntransport: {type: http, url: https://example.com/mcp}\nauth: {type: none}\n"
}
