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

- `relay serve [--addr :8080]` — web dashboard
- Per-implementation status cards (protocol, version, conformance level)
- Live message rate and error counters from running `relay trace` sessions
- `/api/v1/status`, `/api/v1/implementations`, `/api/v1/trace`
- SVG status badge at `/badge/status.svg`
- Webhook on conformance status transitions

Deliverables: `relay serve`

---

## Phase 8 — Stability

### v1.0 — API and spec stability

**Goal:** RELAY spec v1.0. No breaking changes to canonical types or interfaces
without a MAJOR version increment.

- All §14 conformance requirements machine-checkable by `relay conform`
- RELAY spec promoted from v0.1 to v1.0
- Stable Go module API (no breaking changes post-1.0)
- Certification evidence reviewed and signed off

---

## Future (post v1.0)

- **v1.1 — RELAY spec v1.1:** First protocol extension (e.g. CAN XL support)
- **v1.2 — Rust reference:** `relay-rs` crate with all canonical types and trait definitions
- **v1.3 — C++ reference:** `relay.hpp` header with all canonical types and abstract base classes
- **v1.4 — Formal verification:** Model-checked lifecycle invariants (§6)
- **v1.5 — Certification uplift:** ISO 26262 ASIL-D / DO-178C DAL-A evidence path
