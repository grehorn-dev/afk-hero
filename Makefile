# Makefile for AFK-Hero. Mirrors scripts/verify.ps1 so the same feedback
# loop is available on Unix-like shells (git bash, WSL, macOS, Linux CI).
# On native Windows prefer scripts/verify.ps1 — it handles PATH discovery
# for the WinLibs toolchain and golangci-lint that make install side-effects.

SHELL := /usr/bin/env bash

# frontend/node_modules still contains a stray Go helper that npm publishes
# for some packages (e.g. `flatted`). The postinstall hook in
# frontend/package.json patches those files with `//go:build ignore`, but
# keep this filter as a safety net so ad-hoc `make test` still stays clean
# even before `npm ci` has run.
GO_PACKAGES := $(shell go list ./... 2>/dev/null | grep -v '/frontend/node_modules/')

.PHONY: help verify tidy fmt vet lint test test-race build build-wails \
        frontend-install frontend-typecheck frontend-test frontend-lint clean

help:
	@echo "AFK-Hero Makefile targets:"
	@echo "  make verify           - run the full quality gate (fmt, vet, test, lint, frontend, wails build)"
	@echo "  make fmt              - gofmt -w ."
	@echo "  make vet              - go vet ./..."
	@echo "  make lint             - golangci-lint run"
	@echo "  make test             - go test ./... (excludes frontend/node_modules)"
	@echo "  make test-race        - go test -race ./... (requires CGO/gcc)"
	@echo "  make build            - go build ./..."
	@echo "  make build-wails      - wails build (production bundle)"
	@echo "  make tidy             - go mod tidy"
	@echo "  make frontend-install - npm ci (applies the postinstall patch)"
	@echo "  make frontend-typecheck - npx tsc --noEmit"
	@echo "  make frontend-test    - npm test"
	@echo "  make frontend-lint    - npm run lint"
	@echo "  make clean            - remove build/bin and frontend/dist"

verify: fmt vet test test-race lint frontend-install frontend-typecheck frontend-test frontend-lint build-wails

fmt:
	gofmt -w .

vet:
	go vet $(GO_PACKAGES)

lint:
	golangci-lint run

test:
	go test $(GO_PACKAGES)

test-race:
	CGO_ENABLED=1 go test -race $(GO_PACKAGES)

build:
	go build $(GO_PACKAGES)

build-wails:
	wails build

tidy:
	go mod tidy

frontend-install:
	cd frontend && npm ci

frontend-typecheck:
	cd frontend && npx tsc --noEmit

frontend-test:
	cd frontend && npm test

frontend-lint:
	cd frontend && npm run lint

clean:
	rm -rf build/bin frontend/dist
