.PHONY: build test lint clean

BIN := node-watchdog

build:
	go build -ldflags="-s -w" -o bin/$(BIN) ./cmd/watchdog

test:
	go test ./... -v -race

lint:
	golangci-lint run ./...

clean:
	rm -rf bin/
