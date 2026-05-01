.PHONY: build build-slim run test test-live lint fmt clean update-readme validate-progress generate-progress orchestrator-test orchestrator-test-all orchestrator-lint

VERSION ?= 0.2.0-scout
BUILD_FLAGS := -trimpath -ldflags="-s -w -X main.Version=$(VERSION)"
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
	go test ./...

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
