# Milestone 1 — Plan: Backend domain, provider contracts, API skeleton

Status: plan, pre-execution. Builds on M0 (`v0.0.0`). Lands via a pull request against a
branch-protected `main`.

## Goal

Establish every seam the feature milestones fill in: the domain types, the JSON schemas, the
provider interfaces (transcription + SOAP), config-driven provider selection, the
provenance-tracked artifact store, the REST API shape, and the HTTP middleware. Ship exactly
one real provider — the deterministic SOAP template that needs no model and never fails — so
the end-to-end path (create visit → attach a transcript → draft SOAP → export) works entirely
offline with no external dependencies.

Transcription (whisper) is **out of scope** (M2). Real LLM SOAP providers are **out of scope**
(M3). The capture UI is **out of scope** (M4). M1 is contracts + one offline provider + the
API, verified by tests.

## Scope

In:
- Domain types: `Visit`, `Transcript` (segments, speakers, timings), `SoapNote` (S/O/A/P with
  per-section provenance), `Provenance`/visit metadata (app version, provider ids + versions,
  timestamps, whether cloud was used).
- JSON schemas in `schemas/` (`transcript.schema.json`, `soap-draft.schema.json`, `visit.schema.json`),
  ported from the prior `vet-transcriber` repo as the reference spec, with a Go validation test.
- Provider interfaces: `TranscriptionProvider`, `SoapProvider`. Stub/mock transcription impl
  (returns a canned transcript) + **real** `DeterministicTemplateSoap` provider.
- Config-driven provider selection (env → which provider; validated, with clear errors).
- Artifact store: interface + filesystem impl; preserves original audio, transcript, SOAP,
  edits, and metadata; edited artifacts never overwrite originals.
- REST API:
  - `POST /api/visits` — create a visit, accept an audio upload (stored + hash-recorded;
    transcription deferred to M2, so status = `awaiting-transcription`).
  - `GET /api/visits/{id}` — visit + manifest.
  - `PUT /api/visits/{id}/transcript` — attach/replace a reviewed transcript (speaker labels).
  - `POST /api/visits/{id}/soap` — draft SOAP from the reviewed transcript via the configured
    `SoapProvider` (template in M1).
  - `GET /api/visits/{id}/soap` and `GET /api/visits/{id}/export?format=md|docx`.
- HTTP middleware: request-body size limits (`MaxBytesReader`), security headers, structured
  request logging with a request id, panic recovery.
- Carry-over fixes from the M0 review (see below).

Out: whisper transcription (M2), real LLM SOAP (M3), browser capture + review UI (M4),
cloud-provider implementations beyond interface + contract test doubles (M5).

## M0 review carry-overs folded into M1

1. Fix `main.go` to exit non-zero on a startup/serve failure (capture the serve error).
2. Enable branch protection on `main` (require CI + Container + Security checks and a PR)
   before the M1 PR merges; M1 lands through that PR.
3. Add HTTP security headers + body-size middleware (needed for the upload endpoint anyway).
4. SPA deep-link fallback: serve the rendered index page for unmatched non-API GET paths.
5. Frontend health status is exercised by the Go `chromedp` browser e2e (the vanilla
   `internal/api/assets/app.js` sets `data-testid="health-status"` from `/healthz`), which
   replaces the former vitest/Preact unit test of the health branch.

## Deliverables (packages)

```
internal/
  domain/        # Visit, Transcript, SoapNote, Provenance types + validation
  transcribe/    # TranscriptionProvider interface + Stub impl
  soap/          # SoapProvider interface + DeterministicTemplate impl + MD/DOCX render
  store/         # ArtifactStore interface + filesystem impl (provenance-preserving)
  config/        # extend: provider selection + validation
  api/           # extend: /api/visits routes, middleware (size, headers, logging, recover)
schemas/         # transcript.schema.json, soap-draft.schema.json, visit.schema.json
testdata/        # a reviewed-transcript fixture + expected template SOAP (golden, synthetic)
```

## Provider contracts (target Go signatures)

```go
// transcribe
type Request struct {
    AudioPath   string
    Language    string
    Vocabulary  []string
}
type Segment struct {
    Speaker   string
    StartMs   int64
    EndMs     int64
    Text      string
}
type Result struct {
    Segments   []Segment
    ModelID    string
    ModelSHA   string
    Backend    string
}
type Provider interface {
    Transcribe(ctx context.Context, req Request) (Result, error)
    Info() ProviderInfo // id, version, whether it is cloud-backed
}

// soap
type DraftRequest struct {
    Transcript domain.Transcript
    Vocabulary []string
}
type Provider interface {
    Draft(ctx context.Context, req DraftRequest) (domain.SoapNote, error)
    Info() ProviderInfo
}
```

Both interfaces expose `Info()` so provenance records exactly which engine and version produced
each artifact (R6). Cloud providers report `IsCloud=true` so the offline-mode egress test can
assert none are active.

## API contracts (per endpoint, tested)

- `POST /api/visits`: `multipart/form-data` audio ≤ configurable cap → `201` with visit id +
  manifest; oversized or wrong content-type → `413`/`415`; audio stored + SHA256 recorded.
- `PUT /api/visits/{id}/transcript`: schema-valid transcript JSON → `200`; invalid → `422`.
- `POST /api/visits/{id}/soap`: requires a reviewed transcript → `200` with schema-valid SOAP;
  missing transcript → `409`.
- `GET /api/visits/{id}/export?format=docx`: returns a DOCX that opens and contains S/O/A/P.
- All endpoints: body-size limited, security headers present, structured access log emitted,
  panics recovered as `500` without leaking internals.

## Gates

### Computer-verifiable
- All M0 gates stay green (format, lint, vet, govulncheck, tests+coverage, e2e, container
  scans + size + posture, gitleaks, OSV).
- Domain/schema: every schema in `schemas/` validates its golden fixture; round-trip
  (Go type → JSON → schema) passes.
- Providers: interface conformance tests for the stub transcriber and the template SOAP
  provider; the template provider produces **schema-valid** SOAP with S/O/A/P from the golden
  transcript, deterministically, and **never returns an error**.
- API: `httptest` contract tests for every endpoint including the error codes above; a single
  integration test drives create → attach transcript → draft SOAP → export DOCX and asserts a
  valid DOCX with the four sections.
- Store: artifacts persisted with provenance; editing a transcript leaves the original intact
  (asserted).
- Middleware: an over-cap upload is rejected `413`; security headers asserted; a handler panic
  yields `500` and a logged, non-leaking error.
- Coverage floor 80% on `internal/...` holds with the new packages.
- `main` exits non-zero on a simulated bind failure (test the wiring or a scripted check).

### Human-verifiable
- Reviewer reads the provider interfaces and API shape and confirms the seams fit M2
  (drop-in whisper `TranscriptionProvider`) and M3 (drop-in cloud `SoapProvider`) with no API
  change.
- Reviewer POSTs the golden transcript and reads the template SOAP: a sane, structured,
  non-hallucinated draft; confirms it degrades gracefully (no model needed).
- Reviewer inspects a completed visit's artifact folder and can answer "what was captured,
  what was generated, what changed," from the provenance metadata.
- Reviewer confirms branch protection is active (the M1 PR itself is the evidence).

## Verification approach

- Golden fixtures are synthetic/consented only (enforced by the privacy checklist).
- The template SOAP is the M1 correctness anchor: deterministic, schema-valid, offline,
  error-free — so the whole pipeline is testable before any model exists.
- The offline-egress test asserts no provider with `IsCloud=true` is active in the default
  (bundled/offline) configuration.

## Exit criteria

All gates green; the M1 PR reviewed and merged through branch-protected `main`; signed tag
`v0.1.0`; `docs/milestones/M1-report.md` written with gate outputs and any new known gaps.

## Risks

| Risk | Mitigation |
|---|---|
| DOCX generation in Go (no C# OpenXML) | Chosen library: **`github.com/gomutex/godocx`** (MIT, pure Go, no external deps, python-docx-inspired API for headings/paragraphs/tables — fits the S/O/A/P structure). Not yet wired; M1 adds it, pinned + license-checked. Rejected: `fumiama/go-docx` (AGPL-3.0), `nguyenthenguyen/docx` (template-fill, unmaintained since 2017), `unidoc/unipdf` (PDF, commercial license). The golden export test guards structure. |
| Schema drift from the prior C# spec | Port the schemas verbatim as the source of truth; validate Go types against them in tests. |
| Middleware over-blocking legitimate uploads | Size cap is configurable; tested at boundary. |
| Scope creep into M2/M3 | Transcription and real LLM providers are interfaces + doubles only in M1. |
