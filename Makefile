SHELL := /bin/bash

.PHONY: build test lint vuln sbom fmt

build:
	go build ./...

test:
	go test -race -cover ./...

lint:
	golangci-lint run ./...

vuln:
	govulncheck ./...

sbom:
	@command -v syft >/dev/null 2>&1 || { echo "syft not found"; exit 1; }
	mkdir -p artifacts
	syft packages dir:. -o spdx-json > artifacts/sbom.spdx.json

fmt:
	gofmt -w .
