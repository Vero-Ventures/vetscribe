#!/usr/bin/env bash
# Fast local gate: mirrors the CI checks for quick pre-push feedback.
# CI remains authoritative.
set -euo pipefail
cd "$(dirname "$0")/.."

echo "== Go: gofumpt =="
test -z "$(gofumpt -l .)" || { echo "unformatted:"; gofumpt -l .; exit 1; }

echo "== Go: vet =="
go vet ./...

echo "== Go: golangci-lint =="
golangci-lint run ./...

echo "== Go: test (race) =="
go test -race -shuffle=on ./...

echo "== Go: govulncheck =="
govulncheck ./...

echo "== Go: build with embedded frontend =="
CGO_ENABLED=0 go build -o /dev/null ./cmd/vetscribe

echo "All local checks passed."
echo
echo "Note: the browser end-to-end test needs Chrome + a running server and is not"
echo "part of this fast gate. Run it manually with:"
echo "  ./vetscribe & VETSCRIBE_BASE_URL=http://localhost:8080 go test -tags e2e ./internal/api/ -run TestE2E"
