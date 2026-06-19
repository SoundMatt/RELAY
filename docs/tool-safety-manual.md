# RELAY Tool Safety Manual

**Version:** 0.3.0  
**Module:** `github.com/SoundMatt/RELAY`  
**License:** Mozilla Public License 2.0  
**Development ASIL:** ISO 26262 ASIL-C  
**Spec version:** v0.2

---

## 1. Purpose

This is the Tool Safety Manual for RELAY. It is intended for:

- Teams integrating RELAY canonical frame types, validators, or conversion
  helpers into x-Net implementations for safety-critical vehicle systems
- CI architects using `relay conform` as a release gate for x-Net packages
- Auditors assessing conformance of x-Net implementations against the RELAY spec

## 2. Tool overview

RELAY is the specification and shared-type library for the SoundMatt embedded
network protocol ecosystem. It provides:

- Canonical Go types for CAN, DDS, LIN, MQTT, RCP, and SOME/IP frames
- `ValidateFrame` functions enforcing all structural constraints from spec §15
- `ToMessage` / `FromMessage` lossless conversion to/from `relay.Message`
- Common error sentinels (`ErrClosed`, `ErrNotConnected`, `ErrTimeout`, `ErrPayloadTooLarge`)
- Application interfaces `relay.Node` and `relay.Caller`
- The RELAY specification (`spec/relay-spec.md`) defining what constitutes a
  conformant x-Net implementation
- The `relay` CLI (`version`, `capabilities`, `status`)

RELAY is **not** a network driver and does **not** transmit messages to a bus.
It defines the contract; x-Net implementations fulfil it.

## 3. Intended use

### 3.1 Frame validation

Import `github.com/SoundMatt/RELAY/can` (or `lin`) and call `ValidateFrame`
before transmitting a frame. `ValidateFrame` returns `ErrInvalidFrame` for any
structural violation; the caller MUST NOT transmit a frame that fails validation.

### 3.2 Envelope conversion

Use `Frame.ToMessage()` / `FromMessage()` to convert between protocol frames and
`relay.Message`. Round-trips are lossless for all mandatory fields.

### 3.3 Error sentinel detection

Use `errors.Is(err, relay.ErrClosed)` etc. to detect connection state. The four
sentinels are mutually distinct — `errors.Is(a, b)` returns false for any
cross-sentinel pair (§5.1 / REQ-RELAY-012).

### 3.4 Conformance verification

`relay conform <binary>` (v0.5+) verifies an x-Net binary against mandatory
conformance requirements §17.1–17.12.

## 4. Assumptions of use (AoU)

The safety argument depends on these assumptions. Violating them invalidates the
qualification.

- **AoU-1** — `ValidateFrame` is called on every frame before transmission. RELAY
  cannot prevent transmission of an unvalidated frame.
- **AoU-2** — `FromMessage` error return is checked before using the parsed frame.
  Discarding the error and using a zero-value frame is a misuse.
- **AoU-3** — Protocol-specific error types wrap the closest RELAY sentinel via
  `%w`. Callers relying on `errors.Is` for sentinel detection depend on correct
  wrapping in the x-Net implementation.
- **AoU-4** — The RELAY spec version declared in the capabilities document
  (`spec_version`) matches the version of this module imported by the x-Net
  package. Mismatched versions may result in undeclared contract gaps.
- **AoU-5** — The RELAY Go module is imported at a tagged release. Development
  HEAD is not qualified.

## 5. Hazards and mitigations

See `.fusa-hara.json` for the full HARA. Summary:

| Hazard | Description | Safety goal |
|---|---|---|
| H-001 | ValidateFrame accepts an invalid frame | SG-001 |
| H-002 | ToMessage / FromMessage loses a mandatory field | SG-002 |
| H-003 | Error sentinels not distinct — wrong errors.Is result | SG-003 |
| H-004 | MatchTopic wildcard error — wrong subscription delivery | SG-004 |
| H-005 | Spec defect — all x-Nets implement the same wrong behaviour | SG-005 |
| H-006 | Conformance false positive — non-conformant x-Net passes relay conform | SG-006 |

## 6. Known limitations

- `ValidateFrame` for CAN does not check SOME/IP payload framing. SOME/IP
  validation is the responsibility of the `someip` sub-package (`Message.Validate`).
- `MatchTopic` is tested against MQTT §4.7 cases but is not qualified against
  every broker's wildcard interpretation. Broker-specific extensions (e.g.
  shared subscriptions) are out of scope.

## 7. Evidence

| Artifact | Location | Purpose |
|---|---|---|
| Requirements | `.fusa-reqs.json` | Requirements REQ-RELAY-001..081 (incl. 5 cybersecurity), all traced + tested, each with a `verification` method |
| HARA | `.fusa-hara.json` | Hazard analysis and safety goals |
| TARA | `.fusa-tara.json` | Cybersecurity threat analysis (ISO/SAE 21434); mitigations M-001..005 traced to REQ-077..081 |
| Formal model | `docs/formal/RelayLifecycle.tla` | TLC-checked §6 lifecycle invariants |
| Specification | `spec/relay-spec.md` | Normative interface and frame type definitions (embedded evidence) |
| Test suite | `*_test.go` | Requirement traceability via `//fusa:test` |
| CI | `.github/workflows/ci.yml` | Lint, test (80% total + per-package coverage gate), go-FuSa (`check`, `cyber`, `trace -req-coverage 100`), CodeQL, Docker |
| Changelog | `spec/CHANGELOG.md` | Version history and breaking changes |

All evidence is embedded in the `relay` binary (`relay.Evidence`) and bundled by
`relay audit-pack`.

## 8. Qualification level and uplift path

RELAY is qualified as an ISO 26262 **ASIL-C** tool at **TCL2**. The evidence
path to ISO 26262 **ASIL-D** / DO-178C **DAL-A** (via DO-330) is documented in
`docs/asil-d-uplift.md`, which contains the gap analysis and the
Tool-error-Detection / Error-Measure (TD/EM) table. That uplift is incremental
and is not required for the current ASIL-C/TCL2 qualification.
