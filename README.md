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

Milestone 0 (foundation): a single static binary serves the API and a server-rendered UI;
the full CI/DevSecOps toolchain is wired; the container builds to a shell-less, non-root
distroless image. Feature milestones (transcription, SOAP, capture UI) follow.

The repository is a single language and toolchain: **Go**. The frontend is a Go
`html/template` page plus two hand-written vanilla assets (`internal/api/assets/app.js`
and `app.css`) embedded via `embed.FS` — there is no Node/npm/bundler and no build step.
The only browser dependency is the native MediaRecorder JS API in that single asset.

## Requirements to build

- Go 1.26+
- Docker (for the container build)
- Chrome/Chromium (only for the `-tags e2e` browser test)

## Build and run

```bash
# Build and run the single binary (template + assets are embedded from source).
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

# Browser end-to-end (chromedp; needs Chrome + a running server)
./vetscribe &
VETSCRIBE_BASE_URL=http://localhost:8080 go test -tags e2e ./internal/api/ -run TestE2E
```

`scripts/check.sh` runs the fast local set in one command. Pre-commit and pre-push hooks are
wired via `lefthook` (`lefthook install`).

## Acceptance checks

A change is acceptable when: all CI jobs are green; the Go coverage floor holds
(`internal/...` ≥ 80%); the container image builds, passes Trivy + Grype + Dockle with no
HIGH/CRITICAL, runs as non-root with no shell, and stays within its size budget; the browser
end-to-end test passes. See [CONTRIBUTING.md](CONTRIBUTING.md) for the full gate list and merge
governance.
