# Contributing to VetScribe

## Merge governance

- No direct pushes to `main`; all changes go through a pull request.
- Rebase or update each PR onto the current `main` before final verification.
- Every CI gate must be green **and** inspected before merge.
- The integrator inspects the full diff for behavioral risk, security, privacy, and release
  impact.
- Squash-merge by default so `main` stays linear and readable.
- Conventional Commits are enforced (`feat:`, `fix:`, `chore:`, …).

## Required gates

Backend (Go):
- `gofumpt` formatting (no diff), `go vet`, `golangci-lint` (staticcheck, gosec, revive,
  errcheck, ineffassign, misspell, bodyclose, noctx, unconvert, errorlint).
- `go test -race`; union line coverage ≥ 80%.
- `govulncheck` clean.
- Cross-compiles for `linux/amd64` and `linux/arm64`.

Frontend (TypeScript):
- `prettier --check`, `eslint` (typed rules), `tsc --noEmit`.
- `vitest` coverage ≥ 80%.
- Playwright end-to-end passes in Chromium and Firefox.

Container:
- `hadolint` clean; Trivy **and** Grype report 0 HIGH/CRITICAL; Dockle passes.
- Image within its size budget (api profile ≤ 25 MB); runs as non-root with a read-only root
  filesystem and no shell.
- SBOM generated (SPDX + CycloneDX); release images are signed and attested.

Supply chain:
- `gitleaks`, `osv-scanner`, CodeQL (Go + JS/TS), OpenSSF Scorecard.
- All external binaries and models pinned by URL + SHA256; lockfiles committed.

## Privacy / medical-safety checklist (PRs touching audio, transcript, or SOAP)

- No real client audio or PII in the repo, tests, logs, or artifacts (synthetic/consented
  golden data only).
- Generated SOAP remains a draft requiring veterinarian approval; no invented facts.
- Offline mode makes no outbound network connections; cloud providers are explicit and opt-in.

## Milestones

Work proceeds one milestone at a time (see [PLAN.md](PLAN.md)). Each ends with green gates, a
signed tag, and a `docs/milestones/Mx-report.md` recording what was built and the gate outputs.
