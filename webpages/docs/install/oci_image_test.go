package install_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestOCIImageContract proves the Gormes OCI image fixtures port the upstream
// Hermes Docker entrypoint/config-volume operational behavior into a Go-binary
// runtime path with no required Python runtime, and explicitly classify the
// upstream Honcho hosted compose/Prometheus/Grafana files as docs-only
// divergence rather than required local Goncho runtime dependencies.
//
// The test inspects file contents only — it must never invoke `docker build`,
// pull base images, contact registries, or require provider credentials.
func TestOCIImageContract(t *testing.T) {
	dockerfile := readRepoFile(t, "Dockerfile")
	entrypoint := readRepoFile(t, "docker/entrypoint.sh")

	tests := []struct {
		name string
		body string
		// wantAll: every substring must appear in body.
		wantAll []string
		// wantNone: none of these substrings may appear in body.
		wantNone []string
	}{
		{
			// Acceptance #1: Dockerfile fixtures prove the image describes a
			// Go-binary runtime path with no Hermes Python runtime dependency.
			name: "dockerfile_runs_go_binary_with_no_python_runtime",
			body: dockerfile,
			wantAll: []string{
				"FROM golang",
				"AS build",
				"go build",
				"./cmd/gormes",
				"FROM gcr.io/distroless/static",
				"COPY --from=build",
				"/usr/local/bin/gormes",
				"VOLUME [\"/opt/data\"]",
				"ENV GORMES_HOME=/opt/data",
				"ENTRYPOINT [\"/opt/gormes/docker/entrypoint.sh\"]",
				"CMD [\"doctor\", \"--offline\"]",
			},
			wantNone: []string{
				"python",
				"uv pip",
				"playwright",
				".venv",
				"PYTHONUNBUFFERED",
				"npm install",
				"apt-get install",
			},
		},
		{
			// Acceptance #2: Entrypoint preserves offline doctor behavior,
			// config-volume bootstrap, and deterministic command forwarding.
			name: "entrypoint_offline_doctor_config_volume_and_forwarding",
			body: entrypoint,
			wantAll: []string{
				"#!/bin/sh",
				"set -eu",
				"GORMES_HOME=\"${GORMES_HOME:-/opt/data}\"",
				"mkdir -p \"$GORMES_HOME\"",
				"\"$GORMES_HOME/config.yaml\"",
				"if [ \"$#\" -eq 0 ]; then",
				"exec /usr/local/bin/gormes doctor --offline",
				"exec /usr/local/bin/gormes \"$@\"",
			},
			wantNone: []string{
				// No live registry pulls or python venv activation.
				"docker pull",
				"docker push",
				"source",
				".venv/bin/activate",
				"playwright install",
				"uv pip",
			},
		},
		{
			// Acceptance #3: Honcho hosted compose/Prometheus/Grafana files
			// must be classified as docs-only divergence in the Dockerfile,
			// not pulled in as required local Goncho runtime dependencies.
			name: "honcho_hosted_stack_is_docs_only_divergence",
			body: stripDivergenceComments(dockerfile),
			wantAll: []string{},
			wantNone: []string{
				"prometheus.yml",
				"grafana-datasource.yml",
				"docker-compose",
				"postgres",
				"redis",
			},
		},
		{
			// Acceptance #3 positive half: the divergence classifications must
			// be present and explicit, so the row stays auditable.
			name: "honcho_hosted_stack_divergence_is_named_in_dockerfile",
			body: dockerfile,
			wantAll: []string{
				"# divergence: honcho docker-compose.yml.example",
				"# divergence: honcho docker/prometheus.yml",
				"# divergence: honcho docker/grafana-datasource.yml",
				"# divergence: hosted-honcho compose stack is docs-only",
			},
			wantNone: []string{},
		},
		{
			// Acceptance #4: A smoke command can run `gormes doctor --offline`
			// against fake config-volume inputs. The Dockerfile default CMD
			// and the entrypoint zero-arg branch must both wire that command,
			// and the entrypoint must seed config.yaml into $GORMES_HOME so
			// the command sees the fake volume contents.
			name: "offline_doctor_smoke_uses_fake_config_volume",
			body: dockerfile + "\n---\n" + entrypoint,
			wantAll: []string{
				"CMD [\"doctor\", \"--offline\"]",
				"exec /usr/local/bin/gormes doctor --offline",
				"\"$GORMES_HOME/config.yaml\"",
			},
			wantNone: []string{
				"--online",
				"http://",
				"https://",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			for _, want := range tc.wantAll {
				if !strings.Contains(tc.body, want) {
					t.Errorf("missing required fragment %q", want)
				}
			}
			for _, banned := range tc.wantNone {
				if strings.Contains(tc.body, banned) {
					t.Errorf("forbidden fragment present: %q", banned)
				}
			}
		})
	}
}

func readRepoFile(t *testing.T, rel string) string {
	t.Helper()
	// Tests under docs/install run with cwd there, so reach back
	// to the repo root for top-level fixtures like Dockerfile and docker/.
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", rel))
	if err != nil {
		t.Fatalf("read %s: %v", rel, err)
	}
	return string(raw)
}

// stripDivergenceComments drops any line whose first non-whitespace tokens are
// `# divergence:` so that wantNone checks only catch ACTIVE references to
// banned hosted-Honcho assets, not the auditable divergence classifications.
func stripDivergenceComments(body string) string {
	var out strings.Builder
	for _, line := range strings.Split(body, "\n") {
		trim := strings.TrimSpace(line)
		if strings.HasPrefix(trim, "# divergence:") {
			continue
		}
		out.WriteString(line)
		out.WriteByte('\n')
	}
	return out.String()
}
