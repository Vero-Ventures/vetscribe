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

echo "== Frontend =="
(
  cd web
  pnpm install --frozen-lockfile
  pnpm format:check
  pnpm lint
  pnpm typecheck
  pnpm test
  pnpm build
)

echo "== Go: build with embedded frontend =="
CGO_ENABLED=0 go build -o /dev/null ./cmd/vetscribe

echo "All local checks passed."
