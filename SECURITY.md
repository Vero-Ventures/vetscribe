# Security Policy

## Reporting a vulnerability

Report suspected vulnerabilities privately to the maintainers (GitHub Security Advisories on
this repository). Do not open a public issue for undisclosed vulnerabilities. We aim to
acknowledge within a few business days.

## Data handling

VetScribe processes veterinary visit audio, which may contain owner personally identifiable
information (owner name, contact details). Veterinary records are not regulated under HIPAA
(animals are not "patients"), but this PII is still handled with care.

- **On-prem / offline mode (default for sensitive use):** the bundled deployment runs entirely
  on the operator's own machine. Audio and derived artifacts never leave the host. An automated
  test asserts no outbound network connections in offline mode.
- **Cloud-provider mode (opt-in):** when configured to use a cloud transcription or
  SOAP-drafting API, the configuration names exactly what data leaves the machine. Operators
  should prefer no-retention API settings. This mode is never the silent default.
- Generated SOAP notes are drafts requiring veterinarian review and approval before becoming
  part of a medical record.

## Supply chain

- External binaries and models are pinned by URL and SHA256 and verified at build time and
  again at runtime before use.
- Container images are built from minimal distroless bases, run as non-root with no shell, are
  scanned by two independent CVE scanners, and (for releases) are signed with an attached SBOM
  and build provenance.
