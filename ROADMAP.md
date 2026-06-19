# RELAY Roadmap

## Vision

RELAY is the shared specification and tooling layer for the SoundMatt embedded
network protocol ecosystem. Every implementation of CAN, DDS, LIN, MQTT, RCP,
and SOME/IP in Go, Rust, and C++ builds against the RELAY spec — so they share
common types, interfaces, error semantics, and lifecycle guarantees, and any
conformant implementation of the same protocol is interchangeable.

RELAY does **not** route messages at runtime. It defines the contract, validates
conformance, and provides observability tooling.

---

## Guiding Principles

1. **Spec before implementation.** Every Go type, interface, and CLI contract in
   RELAY is a specification artifact first. Implementations follow; RELAY does not
   follow implementations.
2. **Grounded in existing code.** Canonical types and interface signatures are
   derived from what the projects already implement. Breaking changes to existing
   projects are minimised; gaps are filled, not inventions added.
3. **Cross-language by design.** Go is canonical; C++ and Rust are first-class.
   Every spec decision is evaluated against all three languages.
4. **Conformance is machine-checkable.** `relay conform` MUST be able to verify
   any binary without access to source code.
5. **Safety from the start.** go-FuSa is wired into CI from the first commit.
   Requirements are traced and tested before any feature ships.
6. **No runtime dependency.** Implementations do not link against RELAY at runtime
   unless they choose to import canonical Go types. The spec is the dependency.

---

## Phase 1 — Foundation

### v0.1 — Core types and CI ✦ in progress

**Goal:** The RELAY Go module exists, defines the universal envelope, protocol
enum, and application interfaces, and has CI running with quality gates.

- ✅ `relay.Protocol` int enum (CAN=1 … SOMEIP=6) with `String()`
- ✅ `relay.Version` struct with `String()`
- ✅ `relay.Message` universal envelope (§4)
- ✅ `relay.ErrClosed`, `ErrNotConnected`, `ErrTimeout`, `ErrPayloadTooLarge` (§5)
- ✅ `relay.Node` and `relay.Caller` application interfaces (§10)
- ✅ `SubscriberConfig`, `SubscriberOption`, `ApplySubscriberOpts`, `ChanDepth` (§14)
- ✅ `relay.SpecVersion = "0.1"`
- ✅ `relay version` CLI command (text and JSON)
- ✅ DCO check, lint (golangci-lint), 80 %+ coverage gate, CodeQL
- ✅ Requirements registry (`.fusa-reqs.json`, REQ-RELAY-001 … REQ-RELAY-022)
- ✅ GitHub repo (`github.com/SoundMatt/RELAY`)
- ✅ go-FuSa self-check in CI — `gofusa check` gates on ERROR findings, `gofusa trace` reports matrix
- ✅ Docker quickstart (`Dockerfile`, `alpine:3.20` runtime, smoke-tested in CI)

Deliverables: `relay version`

---

### v0.2 — Interface contracts ✦ in progress

**Goal:** Optional interfaces defined in RELAY; x-Net compile-time assertions
filed as issues; `relay capabilities` CLI command shipping.

Per §13.6, the protocol-specific interface types (`Bus`, `Participant`, etc.)
live in each x-Net package — not in RELAY — to avoid circular imports. RELAY
defines the interfaces whose types are entirely within RELAY or stdlib.

- ✅ Optional interfaces: `relay.HealthProvider`, `relay.MetricsProvider`, `relay.Drainer`
- ✅ Supporting types: `relay.Health`, `relay.HealthStatus`, `relay.Metrics`
- ✅ `relay capabilities` CLI command
- ✅ REQ-RELAY-023 … REQ-RELAY-029 added to requirements registry
- ✅ SpecVersion bumped to `"0.2"`; spec issues #2–#9 fixed
- ⬜ Protocol-specific interface compile-time assertions — tracked as x-Net issues
  (go-CAN#15, go-DDS#54, go-LIN#17, go-mqtt#21, go-RCP#51, go-SOMEIP#31, cpp-RCP#5)

Deliverables: `relay version`, `relay capabilities`

---

### v0.3 — Canonical frame types and application interface ✦ in progress

**Goal:** All six protocol canonical types defined in RELAY sub-packages with
validation and envelope conversion; `relay status` CLI command shipping.

Types live in sub-packages (`github.com/SoundMatt/RELAY/can` etc.) so x-Net
packages can import them without circular dependencies. `Adapt()` functions
live in x-Net packages (tracked as issues there) because they wrap x-Net's
Bus/Participant/etc. types which RELAY cannot import.

- ✅ `github.com/SoundMatt/RELAY/can` — Frame, Filter, LoanedFrame, ValidateFrame, MaxDataLen, ToMessage/FromMessage; REQ-RELAY-030..032
- ✅ `github.com/SoundMatt/RELAY/dds` — Sample, QoS, GUID, enums, ValidateDomain, ToMessage/FromMessage; REQ-RELAY-033..034
- ✅ `github.com/SoundMatt/RELAY/lin` — Frame, Filter, ScheduleEntry, ValidateFrame, ProtectID, VerifyPID, CalcChecksum, ToMessage/FromMessage; REQ-RELAY-035..037
- ✅ `github.com/SoundMatt/RELAY/mqtt` — Message, UserProperty, QoS, MatchTopic, ToMessage/FromMessage; REQ-RELAY-038..039
- ✅ `github.com/SoundMatt/RELAY/rcp` — Command, Response, Status, Loan, Zone (PascalCase String()), Priority, CommandType, ResponseStatus, ToMessage/FromMessage; REQ-RELAY-040..041
- ✅ `github.com/SoundMatt/RELAY/someip` — Message, MessageType (MsgType* prefix), ReturnCode (Ret* prefix), SOMEIPProtocolVersion, Validate(), ToMessage/FromMessage; REQ-RELAY-042..043
- ✅ `relay status` CLI command; REQ-RELAY-044
- ✅ REQ-RELAY-030 … REQ-RELAY-044 added to requirements registry
- ⬜ `Adapt()` in each x-Net package — tracked as x-Net issues
- ✅ JSON schemas for canonical types in `spec/schemas/` — delivered in v0.6
- ⬜ C++ and Rust type implementations — tracked as x-Net issues

Deliverables: `relay version`, `relay capabilities`, `relay status`

---

## Phase 2 — Safety Groundwork

### v0.4 — Requirements and HARA ✦ in progress

**Goal:** RELAY is developed as an ASIL-C tool. Full requirements traceability
and hazard analysis in place before conformance tooling ships.

- ✅ `.fusa-hara.json` — 6 hazards (H-001..H-006), 6 safety goals (SG-001..SG-006), ASIL-C worst case
- ✅ `docs/tool-safety-manual.md` — 7-section tool safety manual with AoU, hazard table, evidence index
- ✅ REQ-RELAY-045..050 — §7 constructor contract and §12 schema requirements added
- ⬜ `gofusa trace --strict` CI gate — deferred to v0.5 once relay conform is implemented and 100% traceability is verified
- ⬜ `.fusa-fmea.json` — go-FuSa FMEA schema not yet published; deferred to v0.9

---

## Phase 3 — Conformance Tooling

### v0.5 — `relay conform`

**Goal:** Any RELAY-conformant binary can be verified without source access.

- ✅ `relay conform <binary>` — invokes `version --format json`, `capabilities`,
  `status --format json`; validates each document against §12 schemas
- ✅ Validates sentinel errors via golden error vectors (`spec/vectors/errors/`)
  exercised by `TestErrorVectors` (invalid frame IDs, RTR+FD, wrong protocol
  version, domain out of range)
- ✅ Conformance report: text and JSON (`--format`)
- ✅ Exit 0 on PASS/WARN, exit 1 on any FAIL
- ✅ `relay conform --strict` — also fails on WARN
- ✅ REQ-RELAY-052 … REQ-RELAY-055 added to requirements registry
- ⬜ HTML report renderer — deferred to v0.7 alongside `relay trace` renderers
- ⬜ Live send/subscribe round-trip via known endpoints — deferred (needs
  running x-Net binaries; covered by golden vectors in the interim)

Deliverables: `relay conform` ✅

---

### v0.6 — JSON schemas and spec vectors

**Goal:** Machine-readable spec artifacts that conformance tooling and test
suites can consume directly.

- ✅ JSON Schema (draft 2020-12) for every canonical type (§15): can-frame,
  dds-sample, lin-frame, mqtt-message, rcp-command, rcp-status, someip-message,
  relay-message
- ✅ JSON Schema for version (§12.1), capabilities (§12.2), status (§12.3), and
  conform-result CLI documents
- ✅ Golden reference vectors — one per canonical type with deterministic
  `ToMessage()` output, plus error-condition vectors under `spec/vectors/errors/`
- ✅ Schemas embedded in the binary (`relay.Schema`) and `relay conform`
  validates live output against them via a dependency-free draft-2020-12 subset
  validator
- ✅ `spec/schemas/` and `spec/vectors/` committed and CI-tested
  (`TestGoldenVectorsRoundTrip`, `TestErrorVectors`, `TestGoldenVectorsConformToSchemas`)
- ✅ Fixed SOME/IP `ToMessage`/`FromMessage` lossiness (client_id, session_id,
  message_type now preserved); REQ-RELAY-056 … REQ-RELAY-058 added

Deliverables: `spec/schemas/`, `spec/vectors/` ✅

---

## Phase 4 — Observability

### v0.7 — `relay probe` and `relay trace`

**Goal:** Live message capture and cross-protocol observability for systems
using multiple RELAY-conformant implementations.

- ✅ `relay probe [--scan] [--match glob] [binary...]` — discovers RELAY-conformant
  binaries (explicit or by scanning PATH); reports tool, protocol, version, spec
  version, transports, and adapt
- ✅ `relay trace <binary> [--protocol P] [--count N] [--output FILE]`
  — spawns `<binary> subscribe --format json` and captures the `relay.Message`
  NDJSON stream to stdout or file
- ✅ `relay trace --replay --from FILE` — replays a captured trace
- ✅ Text / JSON / NDJSON renderers; `--protocol P` filter
- ✅ `relay.ParseProtocol` added (REQ-RELAY-059); REQ-RELAY-060/061 for probe/trace
- ⬜ HTML renderer — deferred to v0.8 (`relay report`), which is the HTML/report milestone

Deliverables: `relay probe`, `relay trace` ✅

---

### v0.8 — `relay report`

**Goal:** Cross-protocol conformance and interchangeability report.

- ✅ `relay report [--scan] [--match glob] [--strict] [binary...]` — runs the
  conformance checks across all discovered implementations and produces a
  unified PASS/WARN/FAIL report with per-implementation pass/warn/fail counts
- ✅ Per-implementation conformance table (tool, protocol, result)
- ✅ `relay report --format html` — self-contained HTML dashboard
- ✅ `relay report --format markdown` — GFM for PR comments and wikis
- ✅ `--format text|json` as well; `--strict` escalates WARN to FAIL
- ✅ REQ-RELAY-062; refactored shared `conformBinary` out of `relay conform`

Deliverables: `relay report` ✅

---

## Phase 5 — Safety Evidence

### v0.9 — Full safety evidence set

**Goal:** RELAY carries the same safety evidence as go-FuSa and FuSaOps.

- ✅ TARA (`.fusa-tara.json`) — cybersecurity threat analysis (ISO/SAE 21434):
  3 assets, 5 threats, 5 mitigations, 4 controls
- ✅ SBOM — `relay sbom` derives it from build info (module, Go toolchain, VCS
  revision, dependency components); JSON `relay-sbom/1` or text
- ✅ Build provenance — VCS revision/time/modified surfaced via `relay sbom`
  (full SLSA attestation remains a CI concern)
- ✅ Audit pack — `relay audit-pack` bundles all embedded evidence + schemas
  into a zip with a SHA-256 `manifest.json`
- ✅ `relay safety-case` — assembles requirements + HARA + TARA into a summary
  (text/json/markdown)
- ✅ Evidence embedded in the binary (`relay.Evidence`/`EvidenceNames`)
- ✅ REQ-RELAY-063..065, traced and tested
- ⬜ TCL2 qualification report — narrative document, deferred to the v1.0 docs pass

---

## Phase 6 — Version Compatibility

### v0.10 — `relay compare`

**Goal:** Machine-checked interchangeability between implementations of the
same protocol.

- ✅ `relay compare <binaryA> <binaryB>` — determines whether two
  implementations are interchangeable: same protocol, same spec version, and
  identical command/feature/interface sets; lists the deltas otherwise
- ✅ `relay compare --format json` — machine-readable delta report; exit 1 when
  incompatible
- ✅ `relay versions [--scan] [--match] [binary...]` — lists implementations and
  whether each is aligned with the spec version this relay tool implements
- ✅ REQ-RELAY-066/067, traced and tested
- ⬜ Live `relay.Message` equivalence for identical inputs — deferred (needs
  running x-Net binaries; capability/feature interchangeability covered here)

Deliverables: `relay compare`, `relay versions` ✅

---

## Phase 7 — Dashboard

### v0.11 — `relay serve`

**Goal:** Real-time observability dashboard for systems using multiple
RELAY-conformant implementations.

- ✅ `relay serve [--addr :8080] [--scan] [--match] [--strict] [binary...]` — web dashboard
- ✅ Per-implementation status cards (tool, protocol, version, conformance level)
- ✅ `/api/v1/status`, `/api/v1/implementations` (JSON)
- ✅ SVG status badge at `/badge/status.svg` (green PASS / amber WARN / red FAIL)
- ✅ REQ-RELAY-068/069, traced and tested
- ⬜ Live message-rate/error counters and `/api/v1/trace` — deferred (need
  long-running trace sessions against live x-Net binaries)
- ⬜ Webhook on conformance status transitions — deferred (outbound integration)

Deliverables: `relay serve`

---

## Phase 8 — Stability

### v1.0 — API and spec stability

**Goal:** RELAY spec v1.0. No breaking changes to canonical types or interfaces
without a MAJOR version increment.

- ✅ All §12 conformance documents machine-checkable by `relay conform`; the
  §14 subscriber surface is validated by the canonical-type tests and vectors
- ✅ RELAY spec promoted to **v1.0 (stable)**; `SpecVersion = "1.0"`,
  `spec/version.json` status `stable`
- ✅ Stability guarantee documented (§19.3 / CHANGELOG): canonical types,
  interfaces, error sentinels, and CLI schemas are stable; breaking changes
  require a MAJOR increment
- ✅ Certification evidence (REQ-RELAY-001…069 traced+tested, HARA, TARA, tool
  safety manual) maintained and bundled by `relay audit-pack`

---

## Phase 9 — Protocol Extension

### v1.1 — CAN XL ✦ done

**Goal:** First additive protocol extension under the v1.0 stability guarantee.

- ✅ `can.Frame` extended with CAN XL fields (`XL`, `SDT`, `VCID`, `AF`, `SEC`)
  and the CAN-FD/XL `ESI` flag — all additive, defaulting to zero/false
- ✅ `CANXLMinDataLen`/`CANXLMaxDataLen`/`CANXLMaxPrioID` limits and a
  format-aware `Frame.MaxDataLen()` method
- ✅ `ValidateFrame` XL/ESI constraints (FD⊕XL, 11-bit priority ID, 1…2048-byte
  payload, no Ext/RTR/BRS on XL)
- ✅ Lossless `ToMessage`/`FromMessage` round-trip via `can.{esi,xl,sdt,vcid,af,sec}`
- ✅ Updated `spec/schemas/can-frame.json`; golden vector `can-xl-frame`;
  error vectors `can-fd-xl-mutually-exclusive`, `can-xl-priority-id-overflow`
- ✅ Spec §15.1 + §15.7.1 updated; `SpecVersion = "1.1"`
- ✅ REQ-RELAY-070…073 traced and tested
- ⬜ CAN XL transceiver / segmentation / `Adapt()` — filed as x-CAN issues
  (go-CAN, rust-CAN, cpp-CAN); RELAY defines the contract only

Deliverables: CAN XL canonical type, schema, vectors ✅

---

## Phase 10 — Language References

### v1.2 — Rust reference ✦ done

**Goal:** A complete Rust binding of the spec so a `relay-rs` crate can be built
against §18.3 without ambiguity.

- ✅ §18.3 (Rust) completed: crate layout, core types (`Protocol`, `Version`,
  `Message`), and all six canonical frame types with enums, constants, and serde
  field mappings (incl. v1.1 CAN XL fields)
- ✅ Rust `to_message`/`from_message` conversion contract with Go-identical
  Meta-key mappings (cross-language trace interchangeability)
- ✅ `SpecVersion = "1.2"`
- ⬜ `relay-rs` crate implementation + publish to crates.io — filed as a
  tracking issue; spec defines the binding, the crate is a separate deliverable

Deliverables: complete Rust binding (§18.3) ✅

---

### v1.3 — C++ reference ✦ done

**Goal:** A complete C++ binding of the spec so a `relay.hpp` header can be built
against §18.2 without ambiguity.

- ✅ §18.2 (C++) completed: header-only `relay.hpp` layout, core types
  (`Protocol`, `Version`, `Message`), and all six canonical frame types with
  enums, constants, and validators (incl. v1.1 CAN XL fields)
- ✅ C++ `to_message`/`from_message` convention reuses the §15.7 Meta-key
  mappings identical to Go and Rust (cross-language trace interchangeability)
- ✅ `SpecVersion = "1.3"`
- ⬜ `relay.hpp` header implementation — filed as a tracking issue; spec defines
  the binding, the header is a separate deliverable

Deliverables: complete C++ binding (§18.2) ✅

---

## Phase 11 — Formal Verification

### v1.4 — Model-checked §6 lifecycle ✦ done

**Goal:** The §6 lifecycle requirements are machine-checked, not just prose.

- ✅ TLA+ model `docs/formal/RelayLifecycle.tla` (+ `.cfg`) modelling the node
  lifecycle state machine and its invariants (zero-value safety,
  send/receive-after-close, channels-closed-on-close, no-auto-reconnect)
- ✅ `docs/formal/README.md` requirement→invariant mapping for all ten §6 rules
- ✅ Model + doc embedded as evidence (`relay.Evidence("formal-model")`) and
  bundled by `relay audit-pack`
- ✅ Spec §6.1; `SpecVersion = "1.4"`; REQ-RELAY-074/075 traced and tested
  (`TestFormalModelCoversLifecycle` guards the §6.1…§6.10 mapping)

Deliverables: formal lifecycle model (§6.1) ✅

---

## Phase 12 — Certification Uplift

### v1.5 — ASIL-D / DAL-A evidence path ✦ done

**Goal:** Document the path from the current ASIL-C / TCL2 qualification to the
highest integrity levels.

- ✅ `docs/asil-d-uplift.md`: gap analysis (ISO 26262 ASIL-C → ASIL-D and
  DO-178C DAL-A via DO-330) with a tool-error Detection/Error-Measure (TD/EM)
  table mapping each HARA hazard to its detection measure and evidence
- ✅ Embedded as the `asil-d-uplift` evidence artifact; bundled by `relay audit-pack`
- ✅ Tool safety manual refreshed (stale limitations corrected; evidence index
  updated to REQ-RELAY-001…076 + TARA + formal model; new §8 uplift section)
- ✅ `SpecVersion = "1.5"`; REQ-RELAY-076 traced and tested
- ⬜ The uplift work items (MC/DC coverage, mutation testing, independent
  verification, formal TOR/TQP, SLSA provenance) are tracked as issues when
  scheduled — incremental and not required for the current ASIL-C/TCL2 level

Deliverables: ASIL-D / DAL-A evidence path (`docs/asil-d-uplift.md`) ✅

---

## Phase 13 — Interoperability

### v1.6 — Cross-implementation interop harness ✦ spec done

**Goal:** Verify implementations of the same protocol are *behaviourally*
interchangeable (not just declared-capability compatible, as `compare` checks).

- ✅ Spec §11.2 optional `convert --protocol P` driver command: canonical value
  in (stdin JSON) → that implementation's `relay.Message` out (stdout JSON)
- ✅ Spec §11.2.1 `interop <binaryA> <binaryB> …`: drives each impl with shared
  golden vectors via `convert`, diffs the canonical output pairwise, reports an
  equivalence matrix; canonical `relay.Message` is the cross-language oracle
- ✅ `SpecVersion = "1.6"`; CHANGELOG/ROADMAP
- ⬜ `relay interop` command implementation — filed as a RELAY tracking issue
- ⬜ `convert` per-impl driver — filed as x-Net issues (go-CAN, go-DDS, go-LIN,
  go-mqtt, go-RCP, go-SOMEIP, cpp-RCP), prerequisite for live interop

Deliverables: interop harness spec (§11.2 `convert`, §11.2.1 `interop`) ✅

---

### v1.6.1 — Traceability / coverage / cyber hardening ✦ done

Quality hardening (tooling/evidence PATCH; no normative spec change):
- ✅ All requirements traced **and** tested (closed 6 untraced); CI gates via
  `gofusa trace -req-coverage 100`
- ✅ 5 cybersecurity requirements (REQ-077…081) from the TARA mitigations,
  traced + tested; CI runs `gofusa cyber`
- ✅ Per-package 80% coverage floor (rcp 52%→98%; aggregate 86.7%)
- ✅ Requirement `verification` method + safety-goal/threat links; spec embedded
  as evidence

---

## Phase 14 — Interoperability Build-out

### v1.7 — `relay interop` + reference `relay convert` ✦ done

**Goal:** Implement the v1.6 interop harness in Go, with RELAY as the reference
participant.

- ✅ `relay convert --protocol P` — reference canonical → `relay.Message`
  conversion over the canonical Go types (golden oracle); stdin → stdout
- ✅ `relay interop <binary>...` — diffs each binary's `convert` against the
  in-process reference for every golden vector; per-vector equivalence matrix
  (text/json/markdown); `--strict`; reference is an implicit participant
- ✅ Golden vectors embedded (`relay.Vector`/`relay.VectorNames`)
- ✅ Go fuzz/property tests: CAN validation totality + conversion losslessness,
  LIN/SOME/IP validator totality (discharges an ASIL-D-uplift item)
- ✅ Spec §11.2/§11.2.1 + capabilities; `SpecVersion = "1.7"`; REQ-082/083
- ⬜ x-Net `convert` drivers remain tracked on each implementation (go-CAN#31,
  rust-CAN#2, cpp-CAN#2, …) — once shipped, `relay interop` checks them against
  the reference and each other

Deliverables: `relay convert`, `relay interop` ✅

---

## Phase 15 — Routing

### v1.8 — Crossbar router ✦ done

**Goal:** A central switch fabric that routes/repeats/bridges `relay.Message`
between any protocol spokes.

- ✅ `router` package: `Router` over `relay.Node` (zero-dependency), named spokes
  + routes, filter + converter per route, forwarded/filtered/error stats
- ✅ Converters: `Identity` (repeat), `Retag` (bridge), named registry + `Lookup`,
  `DefaultConverter`
- ✅ `relay crossbar --config FILE` — CLI-backed spokes (subscribe/send pipes),
  JSON config, runs until interrupt/`--duration`, stats on exit
- ✅ Streaming sink `send --format json` (egress dual of subscribe) spec'd
- ✅ Spec §11.2/§11.2.1 + capabilities; `SpecVersion = "1.8"`; REQ-084/085/086
- ⬜ x-Net `send --format json` streaming sink — filed as issues (the existing
  ad-hoc `send` CLIs are non-conformant to §11.2; crossbar needs the JSON sink)
- ⬜ Runtime-routing HARA (new hazards: drop/mis-route/mistranslate) — crossbar
  is QM pending that analysis

Deliverables: `router` library, `relay crossbar`, streaming sink ✅

---

## Phase 16 — Cross-language consistency

### v1.9 — Library architecture convention ✦ done

**Goal:** Make the same protocol in different languages as structurally common
as possible.

- ✅ Spec §13.7: normative cross-language module taxonomy + identical module
  names (the `adapt` adapter, `mock`, `virtual`, standard module-name registry),
  and the local-`relay`-module rule pending published bindings
- ✅ `SpecVersion = "1.9"`; REQ-RELAY-087
- ⬜ x-Net alignment (rename `virtual_bus`→`virtual`, `can_relay`→`adapt`, C++
  feature parity), spec-version uplift, and rust-RCP/rust-DDS conform fixes —
  filed as issues

Deliverables: §13.7 architecture convention ✅

---

## Roadmap complete

All planned phases (1–16, v0.1 → v1.9) are delivered. Future work is demand-driven:
new protocol extensions (additive MINOR releases), reference-implementation
crates/headers (tracked issues), the `relay interop` build-out and its `convert`
driver surface (tracked issues), and the incremental ASIL-D/DAL-A uplift work
items above.
