<!--
VetScribe PR checklist. All gates are enforced by CI; this checklist is the
procedural record that the author verified them and considered the impact.
-->

## What and why

<!-- One or two sentences: what changes and the motivation. -->

## Milestone

<!-- e.g. M1 — backend domain + provider contracts. Link the plan/report. -->

## Gate checklist

- [ ] `go build ./...`, `go test -race ./...` pass; `internal/...` coverage ≥ 80%.
- [ ] `gofumpt -l .` empty, `go vet ./...`, `golangci-lint run ./...` clean.
- [ ] `govulncheck ./...` clean; `go mod tidy` produces no diff.
- [ ] Container builds; Trivy + Grype 0 HIGH/CRITICAL; Dockle clean; image within budget;
      non-root, no shell.
- [ ] Browser `chromedp` e2e passes (if the change touches the UI or API surface).
- [ ] Single-language rule holds: no Node/TS/bundler toolchain introduced.

## Privacy / medical-safety (if this touches audio, transcript, or SOAP)

- [ ] No real client audio or PII in the repo, tests, logs, or artifacts.
- [ ] Generated SOAP remains a draft; no invented facts.
- [ ] Offline mode makes no outbound connections; cloud providers are explicit + opt-in.

## Reviewer notes

<!-- Behavioral risk, security, privacy, and release impact for the integrator audit. -->
