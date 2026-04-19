.PHONY: build run test test-live lint fmt clean

# Strict static build: CGO disabled so the binary has zero C-library
# dependencies and runs on Termux, Alpine, minimal Ubuntu LTS, etc.
build:
	CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o bin/gormes ./cmd/gormes

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
