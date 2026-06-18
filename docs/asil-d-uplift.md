# RELAY Certification Uplift — ASIL-D / DAL-A Evidence Path

## Status

This document defines the **evidence path** for qualifying RELAY at
ISO 26262 ASIL-D and DO-178C DAL-A. It is a gap analysis and plan, **not** a
claim that RELAY is currently qualified at those levels.

- **Current qualification:** RELAY is developed as an ISO 26262 **ASIL-C**
  software tool, qualified as **TCL2** (Tool Confidence Level 2) per the tool
  safety manual (`docs/tool-safety-manual.md`).
- **Target:** ISO 26262 **ASIL-D** tool confidence and an equivalent
  **DO-178C DAL-A** evidence set, so RELAY can be relied upon in the highest
  integrity automotive and airborne programmes.

## Why a tool, not an item

RELAY is a verification/observability **tool**, not deployed software in a
vehicle or aircraft. The relevant ISO 26262 question is therefore Tool
Confidence Level (Part 8 §11), and for DO-178C it is tool qualification
(DO-330). The uplift is about the **confidence** that RELAY does not introduce
or fail to detect errors, evidenced to ASIL-D / DAL-A rigour.

## Current evidence baseline

| Evidence | Artifact | Level today |
|---|---|---|
| Requirements | `.fusa-reqs.json` (REQ-RELAY-001…076), all traced + tested | ASIL-C |
| Hazard analysis | `.fusa-hara.json` (HARA, ASIL-C worst case) | ASIL-C |
| Cybersecurity | `.fusa-tara.json` (TARA, ISO/SAE 21434) | — |
| Lifecycle proof | `docs/formal/RelayLifecycle.tla` (TLC model-checked) | formal |
| Tool safety manual | `docs/tool-safety-manual.md` (AoU, hazards, limitations) | TCL2 |
| Test + coverage | `*_test.go`, CI 80 % coverage gate | ASIL-C |
| Static analysis | golangci-lint, CodeQL, `gofusa check` | ASIL-C |

## Gap analysis — ISO 26262 ASIL-C → ASIL-D

| Area | ASIL-C today | ASIL-D target | Gap / action |
|---|---|---|---|
| Requirements coverage | Statement + branch via tests | MC/DC-equivalent coverage on safety-critical functions (validators, conversions) | Add MC/DC measurement; raise the CI gate from 80 % to ≥95 % on `can`/`lin`/`someip`/conversion paths |
| Structural coverage | Line coverage reported | Decision + condition coverage, with justified exclusions | Adopt a coverage tool reporting decision/condition coverage; document exclusions |
| Independence | Single-author review | Independent verification (different person/team for V&V) | Define an independent-review gate for safety-critical PRs |
| Fault injection | Error vectors (golden) | Systematic fault-injection / mutation testing of validators | Add mutation testing (e.g. `go-mutesting`) with a surviving-mutant budget of 0 on validators |
| Formal methods | §6 lifecycle model-checked | Extend formal coverage to validation invariants and conversion losslessness | Add TLA+/property models or exhaustive property tests for `ValidateFrame` and `ToMessage`∘`FromMessage = id` |
| Tool error detection | Limitations documented | Each potential tool error has a detection or avoidance measure with evidence | Complete the TD/EM (Tool error Detection / Error Measure) table below |
| Configuration mgmt | Git + signed tags | Baselined, immutable, reproducible builds with provenance | SBOM + SLSA provenance already partial (`relay sbom`); add reproducible-build attestation |

## Mapping to DO-178C DAL-A (via DO-330)

| DO-330 objective | RELAY evidence | Gap |
|---|---|---|
| Tool Operational Requirements (TOR) | §3 Intended use + §11.x CLI contract | Formalise TOR as a standalone document |
| Tool Qualification Plan (TQP) | This document + tool safety manual | Promote to a formal TQP with roles |
| Tool verification results | CI logs, test suite, TLC output | Archive per-release verification records |
| Tool Operational V&V | `relay conform` against reference binaries | Add a qualified reference test suite per protocol |
| Configuration management / QA | Git, DCO, signed tags, releases | Add a QA records index |

## Tool error Detection / Error Measure (TD/EM)

For ASIL-D / DAL-A, every credible tool malfunction needs a detection or
avoidance measure. The HARA hazards (H-001…H-006) already enumerate the
malfunctions; the uplift completes the measure column with evidence:

| Malfunction | Detection / avoidance | Evidence |
|---|---|---|
| H-001 invalid frame accepted | Golden error vectors + (target) mutation testing of `ValidateFrame` | `spec/vectors/errors/`, `*_test.go` |
| H-002 conversion loses a field | Golden round-trip vectors + (target) property test `from∘to = id` | `spec/vectors/`, `TestGoldenVectorsRoundTrip` |
| H-003 sentinels not distinct | `errors.Is` vector tests | `TestErrorVectors` |
| H-004 wildcard mis-delivery | MQTT §4.7 match tests | `mqtt/*_test.go` |
| H-005 spec defect propagates | Formal model + conformance suite + independent review (target) | `docs/formal/`, `relay conform` |
| H-006 conformance false positive | Schema validation + (target) qualified reference suite | `cmd/relay/jsonschema*.go` |

## Work breakdown (post-v1.5)

The uplift is delivered incrementally; none of the items below are required for
the current ASIL-C/TCL2 qualification:

1. MC/DC-equivalent coverage measurement and a raised gate on safety-critical packages.
2. Mutation testing of validators with a zero-surviving-mutant budget.
3. Property/formal models for validation invariants and conversion losslessness.
4. An independent-verification gate for safety-critical changes.
5. Formal TOR/TQP documents and per-release verification record archival.
6. Reproducible-build provenance (SLSA) attestation in CI.

Each item is tracked as a GitHub issue when scheduled.
