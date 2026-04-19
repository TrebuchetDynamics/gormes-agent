.PHONY: build test test-live lint fmt clean

build:
	go build -o bin/gormes ./cmd/gormes

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
