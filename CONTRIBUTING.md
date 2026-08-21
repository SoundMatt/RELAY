# Contributing to RELAY

RELAY is the shared specification and canonical-type library that
CAN/DDS/LIN/MQTT/RCP/SOME-IP implementations in Go, C++, Rust, and C build
against. Because every x-Net implementation depends on this repo staying
correct and stable, changes here — especially to `spec/relay-spec.md` — have
a wider blast radius than a typical library change. This document describes
how to propose and land changes safely.

## Before you start

- **Spec changes land here first.** Per §5.5 and similar sections of
  `spec/relay-spec.md`, a new requirement, error sentinel, or protocol
  addition ships as a spec PR *before* any implementation depends on it —
  the spec is the ecosystem-consistency contract a single implementation's
  own PR can't establish on its own.
- **Read §19 (Versioning) first** if your change touches normative
  (MUST/SHOULD/MAY) text. It determines whether the change is a PATCH
  (editorial, no behavior change), MINOR (additive — a new optional
  interface, protocol, CLI command, canonical field, or a genuinely new
  conformance requirement), or MAJOR (breaking change to canonical types,
  interface signatures, or lifecycle requirements). Get this right —
  under-declaring a MINOR bump as a PATCH silently breaks `relay conform`'s
  own version-acceptance logic (§19.3).
- **Check `spec/CHANGELOG.md`** for the style of entry expected: what
  changed, why, explicit scope notes for anything deliberately *not*
  addressed, and — for any version bump — why that bump level was chosen.

## Development

```console
$ go build ./...
$ go vet ./...
$ go test ./...
```

CI (`.github/workflows/ci.yml`) additionally runs:

- **DCO** — every commit must be signed off (see below).
- **Lint** — `gofmt`/`go vet` cleanliness.
- **Test** — build, vet, full test suite, 90% total / 85% per-package
  coverage gates.
- **go-FuSa full safety lifecycle** — `gofusa check` (0 ERROR findings),
  `gofusa trace` (100% requirement traceability via `//fusa:req`/`//fusa:test`
  tags), plus cybersecurity, vulnerability, qualification, dFMEA, and
  supply-chain (SBOM + SLSA provenance) evidence generation.
- **Self-conformance** — `relay conform`/`relay interop` run against RELAY's
  own CLI.
- **CodeQL** and **Docker build**.

All of these must be green before merge; none are optional or skippable via
`|| true`.

### Requirement tags

New or changed behavior that's part of the safety-evidence set needs a
`//fusa:req REQ-RELAY-NNN` tag at its implementation site and a matching
`//fusa:test REQ-RELAY-NNN` at the test that verifies it, plus an entry in
`.fusa-reqs.json`. Never attach an existing requirement ID to new behavior —
give it a new ID with its own genuine implementation site.

## Sign-off (DCO)

Every commit must carry a `Signed-off-by:` trailer certifying the
[Developer Certificate of Origin](https://developercertificate.org/):

```console
$ git commit -s -m "..."
```

CI's DCO check fails the build on any commit missing this trailer.

## Pull requests

- One logical change per PR — a single spec-guidance fix, a single code
  fix, not a bundle of unrelated changes.
- State explicitly what's in scope and what's deliberately left out (a
  finding often turns out broader or narrower than its own filing text
  suggests — investigate against current `HEAD`, not just the issue text,
  and say what you found).
- Reference the issue you're closing (`Closes #N`).
- Squash-merge is the norm; keep the PR's own commit history clean but
  don't worry about it being pristine — the squash commit message is what
  matters.

## Reporting a vulnerability

See [SECURITY.md](SECURITY.md) — do not open a public issue for a security
report.

## License

By contributing, you agree your contributions are licensed under the
[Mozilla Public License 2.0](LICENSE), the same license as the rest of this
repository.
