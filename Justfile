go := env_var_or_default("GO", "go")

default: test

all: test

test:
	{{go}} test ./...

race:
	{{go}} test -race ./...

cover:
	{{go}} test -coverprofile=cover.out ./...
	{{go}} tool cover -func=cover.out

bench:
	{{go}} test -bench=. -benchmem -run='^$' ./...

lint:
	#!/usr/bin/env sh
	if command -v golangci-lint >/dev/null 2>&1; then
		golangci-lint run --timeout=5m
	else
		echo "golangci-lint not installed."
		echo "Install: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"
		exit 1
	fi

fmt:
	{{go}} fmt ./...

tidy:
	{{go}} mod tidy
