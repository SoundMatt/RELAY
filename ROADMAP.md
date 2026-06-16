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

### v0.2 — Interface contracts

**Goal:** Every protocol interface from §7 is defined as a Go interface in the
RELAY module. Implementations can declare conformance by satisfying the interface.

- `relay.Bus` (CAN), `relay.MasterBus` (LIN)
- `relay.Participant`, `relay.Publisher`, `relay.Subscriber` (DDS)
- `relay.Client`, `relay.Subscription` (MQTT, SOMEIP)
- `relay.Controller`, `relay.Registry`, `relay.LoaningController` (RCP)
- `relay.Service`, `relay.Server` (SOMEIP)
- Optional interfaces: `relay.LoaningPublisher`, `relay.HealthProvider`,
  `relay.MetricsProvider`, `relay.Drainer`
- Interface compile-time assertions for all existing Go protocol packages
- `relay capabilities` CLI command (§9.2)

Deliverables: `relay version`, `relay capabilities`

---

### v0.3 — Canonical frame types and application interface

**Goal:** All six protocol canonical types are defined in the RELAY module with
validation and envelope conversion, and `relay.Node` / `relay.Caller` are
implemented so applications can program protocol-agnostically.

- `relay.Frame` (CAN), `relay.Filter` (CAN, LIN), `relay.Sample` (DDS),
  `relay.QoS` (DDS), `relay.LINFrame`, `relay.MQTTMessage`, `relay.UserProperty`,
  `relay.RCPCommand`, `relay.RCPResponse`, `relay.RCPStatus`, `relay.SOMEIPMessage`
- All enum types and constants
- `ValidateCANFrame`, `ValidateLINFrame` with full constraint enforcement (§15)
- `ToMessage()` and `FromMessage()` for all six protocols (§15)
- `relay.Node` interface — pub/sub protocols (§10.1)
- `relay.Caller` interface — request/response protocols (§10.2)
- `Adapt()` in each protocol package: `can.Adapt`, `dds.Adapt`, `lin.Adapt`,
  `mqtt.Adapt`, `rcp.Adapt`, `someip.Adapt` (§10.3)
- `relay.Node` and `relay.Caller` C++ abstract base classes (§18.2)
- `relay::Node` and `relay::Caller` Rust traits (§18.3)
- JSON schemas for all canonical types in `spec/schemas/`
- `relay status` CLI command (§11.1)

Deliverables: `relay version`, `relay capabilities`, `relay status`

---

## Phase 2 — Safety Groundwork

### v0.4 — Requirements and HARA

**Goal:** RELAY is developed as an ASIL-C tool. Full requirements traceability
and hazard analysis in place before conformance tooling ships.

- Expand `.fusa-reqs.json` to cover all §7 and §12 requirements
- HARA (`.fusa-hara.json`) — tool-failure hazards and safety goals
- `gofusa trace --strict` gates CI — all requirements traced and tested
- FMEA (`.fusa-fmea.json`)
- Tool Safety Manual (`docs/tool-safety-manual.md`)

---

## Phase 3 — Conformance Tooling

### v0.5 — `relay conform`

**Goal:** Any RELAY-conformant binary can be verified without source access.

- `relay conform <binary>` — invokes `version --format json`, `capabilities`,
  `status --format json`; validates schemas against §9
- Synthesises protocol-specific frames, sends them via `send`, reads them
  back via `subscribe` using known endpoints where available
- Validates sentinel errors via intentional misuse (`send` after `close`,
  invalid frame IDs, oversized payloads)
- Conformance report: text / JSON / HTML
- Exit 0 on PASS, exit 1 on any FAIL
- `relay conform --strict` — also fails on WARN

Deliverables: `relay conform`

---

### v0.6 — JSON schemas and spec vectors

**Goal:** Machine-readable spec artifacts that conformance tooling and test
suites can consume directly.

- JSON Schema (draft 2020-12) for every canonical type in §12
- JSON Schema for capability (§9.2) and version (§9.1) documents
- Golden reference vectors — one per canonical type, one per error condition,
  with pre-computed `ToMessage()` outputs
- `relay conform` updated to validate against schemas
- `spec/schemas/` and `spec/vectors/` committed and CI-tested

---

## Phase 4 — Observability

### v0.7 — `relay probe` and `relay trace`

**Goal:** Live message capture and cross-protocol observability for systems
using multiple RELAY-conformant implementations.

- `relay probe` — discovers RELAY-conformant binaries on PATH; reports
  protocol, version, spec version, and available transports
- `relay trace [--protocol CAN|DDS|...] [--count N] [--output FILE]`
  — subscribes via conformant binaries and captures `relay.Message`
  objects to stdout (NDJSON) or file
- `relay trace --replay --from FILE` — replays a captured trace
- Text / JSON / HTML renderers for captured sessions

Deliverables: `relay probe`, `relay trace`

---

### v0.8 — `relay report`

**Goal:** Cross-protocol conformance and interchangeability report.

- `relay report` — runs `relay conform` across all discovered implementations
  and produces a unified PASS/WARN/FAIL report
- Per-protocol conformance table (which implementations pass, which gap)
- `relay report --format html` — self-contained HTML dashboard
- `relay report --format markdown` — GFM for PR comments and wikis

Deliverables: `relay report`

---

## Phase 5 — Safety Evidence

### v0.9 — Full safety evidence set

**Goal:** RELAY carries the same safety evidence as go-FuSa and FuSaOps.

- TARA (`.fusa-tara.json`) — cybersecurity threat analysis (ISO 21434)
- SBOM (`sbom.json`) — software bill of materials
- Build provenance (`provenance.json`)
- Qualification report (TCL2)
- Audit pack (`audit-pack.zip`) — all evidence bundled with hashed manifest
- `relay audit-pack`, `relay sbom`, `relay safety-case`
- All requirements at 100 % trace + test coverage

---

## Phase 6 — Version Compatibility

### v0.10 — `relay compare`

**Goal:** Machine-checked interchangeability between implementations of the
same protocol.

- `relay compare <binaryA> <binaryB>` — determines whether two implementations
  expose the same capabilities, support the same features, and produce
  compatible `relay.Message` output for equivalent inputs
- Version compatibility matrix per protocol (which versions are interchangeable)
- `relay compare --format json` — machine-readable delta report
- `relay versions` — list all known implementations and their spec alignment

Deliverables: `relay compare`, `relay versions`

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
