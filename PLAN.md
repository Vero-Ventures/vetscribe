# VetScribe — Cross-Platform Web Rebuild Plan

Status: proposal, pre-approval. No code is written until Milestone 0 is approved.
Working name: **VetScribe** (rename freely). New repo, from scratch. Reuses the prior
`vet-transcriber` C# project only as a *reference spec* for the SOAP schema, transcript
merge rules, prompt templates, and golden test fixtures — not as a code dependency.

## 1. Why we are rebuilding

The WPF desktop app kept failing on environment specifics that cannot be guaranteed on a
clinic's machine, discovered only by running it on a clean Windows VM:

1. WAV file-handle lock on save (Windows mandatory locking) — fixed, but only found in a VM.
2. `whisper-cli.exe` needs `MSVCP140.dll` (VC++ redistributable) absent on a clean box.
3. WASAPI multichannel + audio-device configuration is per-machine and fragile.
4. Windows-only; no path to Mac/Linux/tablet.

Root cause: shipping native binaries to an *uncontrolled* client environment is an
environment lottery. The fix is to ship the **environment**, not just the binary, and to
move capture into the browser (one standardized cross-OS API).

## 2. Goal and non-negotiables

Build the smallest, most reliable veterinary visit → transcript → SOAP-draft tool that runs
with near-100% certainty regardless of client OS/hardware/config.

Non-negotiables (ranked; reliability wins ties):

- **R1 Reliability first.** No dependency on client OS version, installed runtimes, drivers,
  or audio stack beyond "a modern browser with a microphone."
- **R2 One artifact.** A single container image is the unit of deployment. It runs
  identically on free cloud infra and on an old cheap local box, `linux/amd64` and
  `linux/arm64`.
- **R3 Smallest viable footprint.** Static binaries, minimal base image, models mounted at
  runtime (not baked into the image).
- **R4 Cross-platform frontend.** Zero install. Any browser, any OS, desktop or tablet.
- **R5 Draft-not-record.** Generated SOAP is a draft; the veterinarian reviews and approves.
  No invented medical facts; every claim traceable to the transcript or flagged uncertain.
- **R6 Provenance.** Original audio, transcript, SOAP draft, and edits are preserved with
  model/version metadata.
- **R7 Data control.** Support a fully local/on-prem deployment where audio never leaves the
  clinic, and an optional cloud-API mode that is explicit and opt-in.
- **R8 Verifiable quality.** Every milestone has computer-verifiable and human-verifiable
  gates. Nothing is "done" until its gates are green and auditable.

## 3. Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│ Browser (any OS/device)                                              │
│  - MediaRecorder captures visit audio (Opus/WebM)                    │
│  - Uploads audio, shows transcript, edits + speaker labels,          │
│    triggers SOAP, edits SOAP, exports Markdown / DOCX                 │
└───────────────▲───────────────────────────────┬─────────────────────┘
                │ HTTPS (same-origin)            │
                │                                 ▼
┌───────────────┴─────────────────────────────────────────────────────┐
│ Single Go binary (static, CGO_ENABLED=0)                            │
│  - Serves embedded frontend assets (embed.FS)                        │
│  - REST API: /api/visits, /api/visits/{id}/transcript, .../soap      │
│  - Orchestrates providers; writes provenance-tracked artifacts       │
│                                                                      │
│  TranscriptionProvider (interface)                                   │
│    ├─ LocalWhisperCpp   → shells to static whisper.cpp binary        │
│    ├─ GroqWhisperAPI    → HTTPS (optional, opt-in)                   │
│    └─ OpenAIWhisperAPI  → HTTPS (optional, opt-in)                   │
│                                                                      │
│  SoapProvider (interface)                                            │
│    ├─ LocalLlamaCpp     → shells to static llama.cpp binary          │
│    ├─ GeminiAPI / OpenRouterAPI → HTTPS (optional, opt-in)           │
│    └─ DeterministicTemplate → offline fallback, never fails          │
└───────────────┬───────────────────────────┬─────────────────────────┘
                │ subprocess                 │ mounted, SHA256-verified
                ▼                            ▼
   static whisper.cpp / llama.cpp   models volume (ggml-*.bin, *.gguf)
   (musl static, no shared libs)    downloaded on first run, pinned
```

Design consequences that buy reliability:

- **Static Go binary** has zero shared-library dependencies → the `MSVCP140`/glibc-version
  class of failure is structurally impossible.
- **Static whisper.cpp/llama.cpp (musl)** → no runtime redistributable, runs on any Linux
  kernel, old or new.
- **Provider interfaces** → the same product runs fully offline (local providers) or with
  zero local compute (cloud providers) by config, not by rebuild. An old box with no GPU can
  use local Whisper (CPU-friendly) for transcription and a cloud LLM for SOAP if local llama
  is too slow.
- **DeterministicTemplate SOAP fallback** → the pipeline degrades to a structured,
  non-hallucinated template instead of failing when no model is available.

## 4. Stack decisions and rationale

**Single language, single toolchain: Go.** The repository has no Node/TypeScript/bundler
toolchain. The browser needs the MediaRecorder JS API (unavoidable in a browser), so the
frontend is a Go `html/template` page plus one hand-written vanilla `app.js` + `app.css` — real
committed source files with **no build step**, embedded in the binary via `embed.FS`. This is
the simplest stack that meets the reliability goal and keeps the repo one language end to end.

| Concern | Choice | Why |
|---|---|---|
| Language / toolchain | **Go 1.26** (only) | Single static binary (`CGO_ENABLED=0`), trivial cross-compile to amd64/arm64, no runtime, stdlib HTTP, `embed.FS` for single-artifact deploy. One language, one toolchain. |
| Inference | **whisper.cpp + llama.cpp**, static musl builds, version+SHA256 pinned | CPU-only, no GPU required, no shared-lib deps. Reused from prior project's proven manifest approach. |
| Models | Mounted volume, downloaded on first run, SHA256-verified | Keeps image tiny (R3); lets deployer pick model size for their hardware. |
| Frontend | **Go `html/template` + hand-written vanilla `app.js`/`app.css`**, MediaRecorder | No Node/bundler/transpile; server-rendered page, no-build JS. Single language. |
| Frontend delivery | Templates + assets **embedded in the Go binary** (`embed.FS`) | Single artifact; no separate web server; same-origin (no CORS surface). |
| Browser e2e | **`chromedp`** (Go) drives headless Chrome | Real-browser test stays in Go — no Playwright/Node. |
| DOCX export | **`github.com/gomutex/godocx`** (MIT, pure Go) | Create-from-scratch docx in Go; permissive license; keeps the single-language rule. |
| Container base | **`gcr.io/distroless/static-debian12`** (CA certs + tzdata) default; `scratch` for pure-offline profile | Shell-less, non-root, CVE-minimal. |
| Pre-commit hooks | **lefthook** | Single cross-platform binary, fast, language-agnostic. |

Two build profiles from one codebase:

- **`api` profile** — no bundled inference; cloud providers only. Image target **< 25 MB**.
- **`bundled` profile** — static whisper.cpp + llama.cpp included; fully offline. Image
  target **< 80 MB** (models mounted separately).

## 5. Repository layout

```
vetscribe/
├── PLAN.md                      # this file
├── README.md                    # reader-first onboarding + acceptance checks
├── LICENSE
├── CONTRIBUTING.md              # merge governance, gate list
├── SECURITY.md                  # disclosure + data-handling posture
├── .editorconfig
├── .gitleaks.toml
├── lefthook.yml                 # pre-commit/pre-push hook wiring
├── go.mod / go.sum
├── .golangci.yml
├── cmd/vetscribe/main.go        # entrypoint
├── internal/
│   ├── api/                     # HTTP handlers, routing, middleware
│   ├── config/                  # env/flags/config, provider selection
│   ├── transcribe/              # TranscriptionProvider + impls
│   ├── soap/                    # SoapProvider + impls, schema, DOCX/MD export
│   ├── merge/                   # channel transcript merge (ported spec)
│   ├── store/                   # visit artifact store + provenance
│   └── models/                  # pinned model/binary manifest + SHA256 verify
│   └── api/
│       ├── templates/           # Go html/template (index.html.tmpl)
│       ├── assets/              # hand-written app.js + app.css (no build step)
│       └── e2e_test.go          # chromedp browser e2e (build tag: e2e)
├── schemas/                     # soap-draft.schema.json, transcript.schema.json
├── testdata/golden/             # golden audio WAV + expected transcript/SOAP
├── deploy/
│   ├── Dockerfile               # multi-stage, distroless/scratch final
│   ├── docker-compose.yml       # on-prem one-command run
│   ├── fly.toml / render.yaml   # free-cloud deploy manifests
│   └── manifest.json            # pinned binaries + models (url + sha256)
├── scripts/                     # check.sh, verify.sh, size-gate.sh
└── .github/workflows/           # ci, security, container, release, nightly
```

## 6. CI / DevSecOps / quality toolchain

Every check runs in CI and (fast subset) in the lefthook pre-commit/pre-push hooks. CI is
the source of truth; local hooks are an accelerator, not a substitute.

### Go backend
- **Format:** `gofumpt` (verify, no diff).
- **Vet/lint:** `go vet`; `golangci-lint` bundling `staticcheck`, `gosec` (security),
  `revive`, `errcheck`, `ineffassign`, `unconvert`, `misspell`, `bodyclose`, `noctx`.
- **Types/build:** `go build ./...` for `linux/amd64` and `linux/arm64`.
- **Tests:** `go test -race -shuffle=on -coverprofile`; **coverage floor 80%** (union).
- **Vuln:** `govulncheck` (Go vuln DB).
- **Module hygiene:** `go mod verify`, `go mod tidy` diff check.

### Frontend (no separate toolchain)
- The UI is a Go `html/template` page plus hand-written vanilla `app.js`/`app.css` embedded via
  `embed.FS` — no Node, bundler, transpile, or separate lint/format/unit gate.
- **Browser E2E:** a Go `chromedp` test (build tag `e2e`) drives headless Chrome against the
  running binary; in later milestones it feeds a fake audio stream to exercise the
  record→transcribe→SOAP path deterministically in a real browser.

### Container / image
- **Dockerfile lint:** `hadolint`.
- **CVE scan:** `trivy image` **and** `grype` (two independent scanners; gate 0 HIGH/CRITICAL).
- **Config/IaC scan:** `trivy config` on Dockerfile + compose.
- **Image best-practices:** `dockle` (CIS-style, non-root, no setuid, minimal).
- **SBOM:** `syft` → SPDX + CycloneDX, attached to the release.
- **Signing + attestation:** `cosign` sign image; attest SBOM and SLSA build provenance.
- **Size gate:** assert final image ≤ profile target (25 MB api / 80 MB bundled).
- **Runtime posture check:** image runs as non-root, read-only rootfs, no shell present.

### Supply chain and repo hygiene
- **Secrets:** `gitleaks` (CI + pre-commit).
- **Cross-ecosystem vulns:** `osv-scanner`.
- **Static analysis:** **CodeQL** (Go + JavaScript/TypeScript).
- **Pinned downloads:** all external binaries + models pinned by URL + SHA256 in
  `deploy/manifest.json`; verified at build time and again at runtime before use.
- **Dependency updates:** Renovate (deps, GitHub Actions, Docker base) with grouped PRs.
- **Licenses:** `go-licenses` + `license-checker`; deny-list copyleft-incompatible licenses.
- **Commits:** Conventional Commits enforced by `commitlint`.
- **Posture:** OpenSSF **Scorecard** workflow (branch protection, pinned actions, etc.).

### Merge governance (ported from the prior project's discipline)
- No direct pushes to `main`; PRs only; rebased on current `main` before merge.
- All CI gates green **and** inspected; squash-merge for a linear history.
- Integrator-owned full-diff audit noting behavioral risk, security, privacy, release impact.
- Privacy/medical-safety checklist completed per PR touching audio, transcript, or SOAP.

## 7. Verification philosophy

- **Golden audio corpus** (`testdata/golden/`): a small set of consented or synthetic
  vet-visit WAVs with hand-verified reference transcripts and reference SOAP structure. Kept
  in-repo (synthetic/consented only — never real client data).
- **Transcription gate:** Word Error Rate (WER) against the reference, deterministic model +
  seed. Gate: **WER ≤ 0.15** on clean audio, **≤ 0.30** on the noisy fixture. Drug-name recall
  tracked separately (custom-vocabulary bias).
- **SOAP gate:** JSON schema-valid; contains S/O/A/P; a faithfulness check that every SOAP
  assertion maps to a transcript span or is explicitly flagged uncertain (automated heuristic
  + human spot-check on the golden set).
- **End-to-end gate (headline):** Playwright drives the real UI with fed audio → asserts the
  transcript contains expected keywords → generates SOAP with four sections → exports a DOCX
  that opens and contains the sections. Full stack, real browser, computer-verifiable.
- **Reproducibility:** deterministic image build; release pins the image digest; SBOM diff
  reviewed on change.
- **Deploy smoke:** post-deploy script hits `/healthz` and runs a trimmed e2e against the live
  instance (cloud and on-prem).
- **Low-end hardware gate:** the full golden e2e must pass inside a constrained VM
  (2 vCPU / 2 GB RAM, no GPU) within a documented time budget, proving "old cheap computer"
  viability.

## 8. Milestones

Each milestone ends with a signed git tag, a `docs/milestones/Mx-report.md` (the auditable
record: what was built, gate outputs, known gaps), and both gate types green. A milestone is
not complete until every gate below it passes and the report is written.

### M0 — Foundation and empty-but-green pipeline
Deliverable: repo initialized; licenses, README, CONTRIBUTING, SECURITY; all formatters,
linters, hooks, and CI wired; a hello-world Go binary that builds and containerizes to
distroless; Renovate/CodeQL/Scorecard/gitleaks active.
- Computer gate: CI green with every tool from §6 running (even on near-empty code);
  `docker build` produces a distroless image that runs and passes trivy+grype+dockle with 0
  HIGH/CRITICAL and meets the size gate; pre-commit hooks block a planted lint error.
- Human gate: reviewer confirms the gate list is complete and each tool actually executes
  (no silent skips); reads README and can state the build/run/test commands.
- Artifact: `M0-report.md` with raw CI logs linked; first signed tag `v0.0.0`.

### M1 — Backend skeleton + embedded frontend
Deliverable: `GET /healthz`; config loader; provider interfaces defined (stubs); Vite/Preact
app built and embedded; single binary serves API + UI.
- Computer gate: `/healthz` returns 200 with build metadata; Playwright loads the page and
  sees the app shell; binary is a single static file (ldd shows "not a dynamic executable");
  cross-compiles amd64+arm64.
- Human gate: reviewer opens the app in a browser, sees the shell, confirms one-binary run.
- Artifact: `M1-report.md`; tag `v0.1.0`.

### M2 — Transcription (local whisper.cpp, static)
Deliverable: `POST /api/visits` (audio) → transcript; static whisper.cpp integration; model
manifest + SHA256 verify; offline operation.
- Computer gate: golden WER ≤ thresholds; a network-disabled test proves offline
  transcription; malformed/oversized upload handled; model hash mismatch refuses to run.
- Human gate: reviewer records a phrase in the browser, sees a correct transcript; inspects a
  saved visit's original audio + transcript artifacts.
- Artifact: `M2-report.md` with WER numbers; tag `v0.2.0`.

### M3 — SOAP drafting (pluggable, local + deterministic fallback)
Deliverable: transcript → schema-valid SOAP JSON; LocalLlamaCpp provider;
DeterministicTemplate fallback; Markdown + DOCX export.
- Computer gate: SOAP is schema-valid with S/O/A/P; faithfulness heuristic passes on golden
  set; fallback path produces a valid note with no model present; DOCX opens and contains
  sections.
- Human gate: reviewer reads a generated SOAP for a golden visit and confirms it is a sane,
  non-hallucinated draft; triggers the fallback and confirms graceful degradation.
- Artifact: `M3-report.md`; tag `v0.3.0`.

### M4 — Frontend capture + review flow
Deliverable: MediaRecorder capture; upload; transcript display + inline edit + speaker labels;
SOAP trigger + edit; export; error/status surfaces.
- Computer gate: Playwright e2e (Chromium + Firefox) with fed fake audio drives
  record→transcript→edit labels→SOAP→export DOCX and asserts each step; a11y smoke
  (axe-core) passes; no console errors.
- Human gate: reviewer completes a full visit in the browser on a desktop **and** a tablet
  form factor; exports and opens the DOCX.
- Artifact: `M4-report.md` + screen recording; tag `v0.4.0`.

### M5 — Full golden e2e + provenance + optional cloud providers
Deliverable: end-to-end golden path; provenance-tracked artifact store (original audio,
transcript, SOAP, edits, model/version metadata); Groq/OpenAI/Gemini/OpenRouter providers
behind config with contract tests.
- Computer gate: single test runs known-audio → SOAP through the whole stack and validates
  every artifact + provenance field; provider matrix contract tests pass (mocked for cloud);
  switching providers needs config only, no rebuild.
- Human gate: reviewer inspects a completed visit's artifact folder and can answer "what did
  the model hear, what did it generate, what changed."
- Artifact: `M5-report.md`; tag `v0.5.0`.

### M6 — Container hardening + supply chain
Deliverable: final distroless/scratch images for both profiles; non-root; read-only rootfs;
SBOM; signed image + attestations; size gates enforced.
- Computer gate: trivy+grype+dockle 0 HIGH/CRITICAL; image ≤ profile target; runs non-root
  with read-only rootfs; `cosign verify` succeeds; SBOM attached; SLSA provenance attested;
  reproducible-build check (two builds → same digest, modulo timestamps).
- Human gate: reviewer verifies the signature + SBOM out-of-band and confirms the image has no
  shell and drops privileges.
- Artifact: `M6-report.md` with scan reports + SBOM; tag `v0.6.0`.

### M7 — Deploy verification (cloud + on-prem + low-end)
Deliverable: one-command deploy to a free cloud host (Fly/Render/Cloudflare) and
`docker-compose up` on a fresh Linux box; low-end hardware validation.
- Computer gate: post-deploy smoke (live `/healthz` + trimmed e2e) green on the cloud URL;
  compose smoke green on a fresh VM; full golden e2e passes inside the 2 vCPU / 2 GB VM within
  the documented time budget.
- Human gate: reviewer follows the README from zero on a clean machine and reaches a working
  transcript+SOAP; confirms on-prem mode keeps audio local (network egress check).
- Artifact: `M7-report.md` with timings + deploy logs; tag `v1.0.0-rc`.

### M8 — Production hardening (stretch, post-MVP)
Auth (per-clinic), rate limiting, structured logging + metrics, data-retention controls,
WCAG 2.1 AA a11y gate, i18n scaffolding, load test. Each with its own gates. Tag `v1.0.0`.

## 9. Data and privacy

- **On-prem default for sensitive use:** the `bundled` profile runs entirely on the clinic's
  box; audio and PII never leave. An automated egress test asserts no outbound connections in
  offline mode.
- **Cloud mode is explicit + opt-in:** provider config names exactly what leaves the machine;
  README states data-handling per provider; prefer no-retention API settings.
- Vet records are not HIPAA-regulated (animals are not "patients"), but owner PII exists;
  SECURITY.md documents handling and the retention controls (M8).
- Generated SOAP is always a draft requiring veterinarian approval (R5).

## 10. Risks and mitigations

| Risk | Mitigation |
|---|---|
| Local llama.cpp too slow on old CPU for SOAP | Pluggable SOAP provider → cloud LLM or deterministic template; document per-hardware guidance. |
| Browser audio format vs whisper input | Server transcodes WebM/Opus → 16 kHz WAV via a pinned static ffmpeg (or minimal decoder); covered by golden e2e. |
| Static musl builds of whisper/llama unavailable/broken | Build them in a pinned builder stage from source; SHA256-lock the outputs; CI reproduces. |
| Model download at first run fails offline | Ship a documented sideload path (mount a pre-fetched models dir); hash-verify either way. |
| Scope creep past MVP | M8 is explicitly post-MVP; M0–M7 define the shippable MVP. |
| Golden corpus contains real client data | Only synthetic/consented audio in-repo; enforced in review checklist. |

## 11. Locked decisions

1. **Name / host** — `vetscribe`, new repo under the `Vero-Ventures` GitHub org.
2. **Frontend framework** — Preact + TypeScript.
3. **Default SOAP provider (MVP)** — **Cloud LLM** (fast + reliable on any hardware); local
   llama.cpp and the deterministic template ship as selectable providers. Transcription stays
   local Whisper by default.
4. **First deploy target (M7)** — Render (`render.yaml`).
5. Module path: `github.com/Vero-Ventures/vetscribe`.

Execution begins at **M0** and proceeds one milestone at a time; each milestone's gates must be
green and its report written before the next begins.
