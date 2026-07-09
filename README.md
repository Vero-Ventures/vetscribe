# VetScribe

Cross-platform veterinary visit transcription and SOAP-note drafting. The browser captures
audio; a single static Go binary (with the embedded frontend) transcribes and drafts the note.
It runs identically on free cloud infrastructure and on an old, cheap local computer — one
container image, `linux/amd64` and `linux/arm64`.

This repository is a ground-up rebuild of the earlier Windows desktop app, whose failures
(missing runtime DLLs, file locking, audio-device config) came from shipping native binaries
into uncontrolled client environments. VetScribe ships the *environment* (a container) and
moves capture into the browser. See [PLAN.md](PLAN.md) for the full architecture, milestones,
and gates.

## Status

Milestone 0 (foundation): a single static binary serves the API and an embedded Preact UI;
the full CI/DevSecOps toolchain is wired; the container builds to a shell-less, non-root
distroless image. Feature milestones (transcription, SOAP, capture UI) follow.

## Requirements to build

- Go 1.24+
- Node 22+ with pnpm (via corepack)
- Docker (for the container build)

## Build and run

```bash
# 1. Build the frontend into the Go embed directory
cd web && pnpm install --frozen-lockfile && pnpm build && cd ..

# 2. Build and run the single binary
CGO_ENABLED=0 go build -o vetscribe ./cmd/vetscribe
./vetscribe                       # serves on :8080
```

Or the container:

```bash
docker build -f deploy/Dockerfile -t vetscribe .
docker run --rm -p 8080:8080 vetscribe
```

Open <http://localhost:8080>. Health: `GET /healthz`.

## Quality gates (run locally)

```bash
# Go
gofumpt -l .            # formatting (no output = clean)
go vet ./...
golangci-lint run ./...
go test -race -shuffle=on ./...
govulncheck ./...

# Frontend (in web/)
pnpm format:check
pnpm lint
pnpm typecheck
pnpm test               # unit + coverage floor
pnpm e2e                # browser end-to-end (needs a running server)
```

`scripts/check.sh` runs the fast local set in one command. Pre-commit and pre-push hooks are
wired via `lefthook` (`lefthook install`).

## Acceptance checks

A change is acceptable when: all CI jobs are green; coverage floors hold (Go and frontend
≥ 80%); the container image builds, passes Trivy + Grype + Dockle with no HIGH/CRITICAL, runs
as non-root with no shell, and stays within its size budget; the browser end-to-end test
passes. See [CONTRIBUTING.md](CONTRIBUTING.md) for the full gate list and merge governance.
