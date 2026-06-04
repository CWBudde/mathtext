.PHONY: all test race cover bench lint fmt tidy

GO ?= go

all: test

test:
	$(GO) test ./...

race:
	$(GO) test -race ./...

cover:
	$(GO) test -coverprofile=cover.out ./...
	$(GO) tool cover -func=cover.out

bench:
	$(GO) test -bench=. -benchmem -run=^$$ ./...

lint:
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run --timeout=5m; \
	else \
		echo "golangci-lint not installed."; \
		echo "Install: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
		exit 1; \
	fi

fmt:
	$(GO) fmt ./...

tidy:
	$(GO) mod tidy
