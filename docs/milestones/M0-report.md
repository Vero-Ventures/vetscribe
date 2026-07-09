# Milestone 0 — Foundation and empty-but-green pipeline

Date: 2026-07-09

## Deliverable

Repository scaffolded from scratch: a single static Go binary serving a JSON API and an
embedded Preact/TypeScript frontend, with the full CI/DevSecOps/quality toolchain wired and a
minimal distroless container.

## What was built

- **Backend:** Go 1.24 (toolchain pinned `go1.24.8` for a patched stdlib), `cmd/vetscribe`
  with graceful shutdown, `internal/config`, `internal/api` (`/healthz` + embedded frontend
  with a fresh-checkout placeholder fallback).
- **Frontend:** Preact + TypeScript + Vite, built into the Go embed directory so one binary
  serves everything. Vitest unit tests, Playwright e2e scaffold (Chromium + Firefox).
- **Container:** multi-stage `deploy/Dockerfile` (node build → static Go build → distroless
  `static-debian12:nonroot`).
- **Quality config:** `.golangci.yml`, `gofumpt`, `lefthook` hooks, `.editorconfig`,
  `.gitleaks.toml`, `.prettierrc`, `eslint.config.js`.
- **CI:** `ci.yml` (Go + frontend + browser e2e), `security.yml` (gitleaks, OSV, CodeQL,
  Scorecard), `container.yml` (hadolint, Trivy, Grype, Dockle, SBOM, size + posture gates),
  `renovate.json`.
- **Docs:** README, CONTRIBUTING (gate list + merge governance), SECURITY (data handling), the
  full PLAN.md.

## Gate results (local)

| Gate | Result |
|---|---|
| `go build ./...` | pass |
| `go test -race ./...` | 4 tests pass |
| `gofumpt -l .` | clean |
| `go vet ./...` | clean |
| `golangci-lint run ./...` | no issues |
| `govulncheck ./...` | no called vulnerabilities |
| Cross-compile amd64 + arm64 | pass |
| Static binary check | `not a dynamic executable` |
| Frontend typecheck / lint / format | pass |
| Frontend unit tests | pass, 100% of `app.tsx` |
| Frontend build | pass (14 KB gzip 6 KB bundle) |
| Container build | pass |
| Container image size | **8.06 MB** (budget ≤ 25 MB) |
| Container serves API + real UI | verified via `curl` |
| Container non-root, no shell | verified (`id`/`sh` absent — distroless) |

## Computer-verifiable gate

CI runs the full toolchain above on every PR. The container job enforces the size budget,
dual CVE scan (Trivy + Grype), Dockle best-practices, SBOM generation, and the
non-root/no-shell posture check.

## Human-verifiable gate

A reviewer can: build per README and open the app in a browser; confirm a single binary serves
both API and UI; confirm the container is 8 MB, non-root, and shell-less.

## Known gaps (by design, deferred to later milestones)

- No transcription yet (M2), no SOAP drafting (M3), no capture UI (M4).
- Playwright e2e currently asserts only the app shell + health; the record→transcribe→SOAP
  flow lands in M4.
- Container is the `api` profile; the `bundled` (offline inference) image is M6.

## Verification note

The pnpm 11 pre-run deps-check misbehaves under the local sandboxed corepack; disabled via
`verifyDepsBeforeRun: false` in `pnpm-workspace.yaml`. CI uses an explicit
`pnpm install --frozen-lockfile`, so the check is not relied upon.
