.PHONY: build build-slim run test test-integration test-e2e test-release test-live lint fmt clean update-readme validate-progress generate-progress orchestrator-test orchestrator-test-all orchestrator-lint

VERSION ?= $(shell sed -nE 's/^[[:space:]]*(var[[:space:]]+)?Version[[:space:]]*=[[:space:]]*"([^"]+)".*/\2/p' cmd/gormes/version.go | head -n1)
GIT_COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
GIT_DIRTY ?= $(shell git diff --quiet 2>/dev/null && git diff --cached --quiet 2>/dev/null && echo false || echo true)
BUILD_FLAGS := -trimpath -ldflags="-s -w -X main.Version=$(VERSION) -X main.GitCommit=$(GIT_COMMIT) -X main.GitDirty=$(GIT_DIRTY)"
BINARY_PATH := bin/gormes
SLIM_BINARY_PATH := bin/gormes-slim

build: validate-progress $(BINARY_PATH)
	@$(call record-benchmark)
	$(MAKE) -s generate-progress
	@$(call update-readme)

$(BINARY_PATH):
	CGO_ENABLED=0 go build $(BUILD_FLAGS) -o $(BINARY_PATH) ./cmd/gormes

build-slim: $(SLIM_BINARY_PATH)
	@echo "Built slim binary (excludes TTS, transcription, voice mode, image generation)"

$(SLIM_BINARY_PATH):
	CGO_ENABLED=0 go build $(BUILD_FLAGS) -tags slim -o $(SLIM_BINARY_PATH) ./cmd/gormes

validate-progress:
	@echo "Validating progress.json..."
	@go run ./cmd/progress validate

generate-progress:
	@echo "Regenerating progress-driven markdown..."
	@go run ./cmd/progress write

define record-benchmark
	@echo "Recording benchmark..."
	@go run ./cmd/repoctl benchmark record
endef

define update-readme
	@echo "Updating README.md..."
	@go run ./cmd/repoctl readme update
endef

update-readme:
	@$(call update-readme)

run: build
	./bin/gormes

test:
	go test ./... -count=1

test-integration:
	go test ./internal/provider/router ./internal/support/testutil/... ./internal/persistence/session ./internal/memory ./internal/gateway ./internal/adapters/channels/... ./internal/tools/... ./internal/llm/learning -run 'Test.*(Contract|Integration|Fake|Fixture|SQLite|Gateway|Webhook|Migration|Dashboard|Navivox|Tool|Schema|Learning)' -count=1

test-e2e:
	go test ./cmd/gormes ./internal/support/e2e ./internal/tui -run 'Test.*(E2E|EndToEnd|WideE2E|MultiTurn)' -count=1

test-release: validate-progress
	go test ./internal/platform/installtest ./webpages/docs/install ./cmd/gormes -run 'Test.*(Release|Install|Version|Checksum|Uninstall|PublicInstall)' -count=1
	CGO_ENABLED=0 go build $(BUILD_FLAGS) -o $(BINARY_PATH) ./cmd/gormes
	./$(BINARY_PATH) version --json >/tmp/gormes-version.json

test-live:
	go test -tags=live ./...

lint:
	golangci-lint run

fmt:
	gofmt -w .
	goimports -w .

clean:
	rm -rf bin/ coverage.out

orchestrator-test:
	@bash scripts/orchestrator/tests/run.sh unit

orchestrator-test-all:
	@bash scripts/orchestrator/tests/run.sh unit integration

orchestrator-lint:
	@if command -v shellcheck >/dev/null 2>&1; then \
	  shellcheck scripts/gormes-auto-codexu-orchestrator.sh \
	    scripts/gormes-builder-loop.sh \
	    scripts/gormes-builder-cron.sh \
	    scripts/orchestrator/audit.sh \
	    scripts/orchestrator/daily-digest.sh \
	    scripts/orchestrator/install-service.sh \
	    scripts/orchestrator/install-audit.sh \
	    scripts/orchestrator/disable-legacy-timers.sh \
	    testdata/legacy-shell/scripts/gormes-auto-codexu-orchestrator.sh \
	    testdata/legacy-shell/scripts/orchestrator/lib/*.sh; \
	else \
	  echo "shellcheck not installed; skipping"; \
	fi
