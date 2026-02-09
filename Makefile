SHELL := /bin/bash

.PHONY: build test lint vuln sbom fmt ui-build

build: ui-build
	go build -v ./...

ui-build:
	cd web && npm install && npm run build
	mkdir -p internal/webui/dist
	rm -rf internal/webui/dist/*
	cp -r web/dist/* internal/webui/dist/

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
