# RELAY Spec Changelog

## v1.10 — 2026-06-19 (stable)

Continuous conformance — conformance must now be *continuously proven*, not just
declared. Additive (MINOR).

- New **§20 Continuous Conformance**, making the CI process normative:
  - **§20.1 CI gates** — a conformant implementation's CI MUST gate on
    `relay conform --strict`, the **full x-FuSa lifecycle** (check + 100%
    traceability + cyber + vuln + qualification), and `relay interop`
    behavioural conformance; releases tagged only from green CI.
  - **§20.2** behavioural conformance against the embedded golden vectors via
    `convert`.
  - **§20.3** core- vs **tooling-conformant** tiers (the latter adds `convert`,
    `subscribe`/`send --format json`); an advertised-but-erroring `convert` is a
    conformance failure.
  - **§20.4** mandatory evidence (requirements registry, HARA, dFMEA, TARA where
    untrusted input is processed) — schemas owned by x-FuSa, presence required.
  - **§20.5** supply-chain: SBOM + build provenance (SLSA), signed releases.
- RELAY's own CI now runs the **full go-FuSa lifecycle** (adds `vuln`, `qualify`,
  `verify`, `fmea`, `boundary`, `coupling`, `release`, and the ISO 26262 /
  DO-178C / SLSA / ISO 21434 gap reports + `audit-pack`) — the reference exemplar.
- `SpecVersion = "1.10"`; REQ-RELAY-088/089/090.

## v1.9 — 2026-06-19 (stable)

Cross-language library-architecture convention. Additive (MINOR).

- New **§13.7**: a normative module taxonomy with **names identical across
  languages** (idiomatic packaging per language). Mandates the `adapt` adapter
  module (not protocol-prefixed names like `can_relay`), `mock`, and a standard
  module-name registry (`virtual` not `virtual_bus`, `socketcan`, `safety`,
  `dbc`, `isotp`, `j1939`, `obdii`, `uds`, `recorder`, `codegen`, the RCP
  control-plane set, bridges, …) so the same protocol in Go/Rust/C++ is
  structurally consistent and interchangeable to maintainers.
- §13.7.3: until the `relay-rs`/`relay.hpp` binding is published, an
  implementation MUST bundle the RELAY core types in a single `relay` module.
- `SpecVersion = "1.9"`; REQ-RELAY-087.

## v1.8.1 — 2026-06-19 (doc clarification; no normative change)

- §15.7.1 (CAN `ToMessage`/`FromMessage`) now documents the CAN XL Meta keys
  (`can.esi/xl/sdt/vcid/af/sec`) and the **emission rule**: the four classic
  flags are always emitted; the CAN-FD/XL fields are emitted only when set, so
  classic/CAN-FD output is unchanged from v1.0. Documents existing v1.1
  behaviour (RELAY#42). `SpecVersion` unchanged (`1.8`).

## v1.8 — 2026-06-19 (stable)

The RELAY **crossbar** — a central protocol router. Additive (MINOR).

- New **`router` package** (`router.Router` over `relay.Node`): a zero-dependency
  switch fabric. Register named spokes + routes; each route forwards a source
  spoke's messages to one or more destinations with an optional filter and
  converter. Embeddable in-process with `Adapt()`ed implementations.
- **Converters** (`router`): `Identity` (repeat), `Retag` (cross-protocol),
  a named registry + `Lookup`, and `DefaultConverter` (identity for
  same-protocol routes, re-tag otherwise).
- New **`relay crossbar --config FILE`** command: builds the router from a JSON
  config of spokes (CLI-backed nodes) and routes; runs until interrupted or
  `--duration`; reports forwarded/filtered/error stats.
- New **streaming sink**: `send --format json` reads `relay.Message` NDJSON on
  stdin (the egress dual of `subscribe --format json`) — the portable,
  protocol-uniform sink the crossbar uses, avoiding per-protocol send flags.
- `SpecVersion = "1.8"`; REQ-RELAY-084/085/086; `crossbar` added to capabilities.
- **Safety note:** runtime routing introduces new hazards (drop/mis-route/
  mistranslate) not yet in the HARA, so the crossbar requirements are **QM**
  pending hazard analysis.

## v1.7 — 2026-06-19 (stable)

Interoperability build-out — the v1.6 interop harness implemented in Go, with a
reference `convert` driver. Additive (MINOR).

- **`relay convert --protocol P`** (§11.2): RELAY's reference canonical-value →
  `relay.Message` conversion over the canonical Go types — the golden oracle for
  interop. Reads a canonical value as JSON on stdin, validates it, writes the
  lossless `relay.Message` on stdout (normalised timestamp).
- **`relay interop <binary>...`** (§11.2.1): drives each binary's `convert` and
  diffs its output against RELAY's in-process reference for every golden vector,
  reporting a per-vector equivalence matrix (text/json/markdown). The reference
  is an implicit participant, so a single implementation can be checked without a
  second present. `--strict` fails on missing `convert`.
- Both commands added to the `relay capabilities` command set.
- Golden vectors are now embedded (`relay.Vector`/`relay.VectorNames`).
- Added Go fuzz/property tests: CAN `ValidateFrame` totality + `FromMessage∘
  ToMessage = id` losslessness (5.9M execs clean), LIN/SOME/IP validator
  totality — discharging an ASIL-D-uplift work item in Go.
- `SpecVersion = "1.7"`; REQ-RELAY-082/083 traced and tested.

## v1.6.3 — 2026-06-19 (evidence/metadata; no normative change)

Per-requirement ASIL allocation. Specification unchanged (`SpecVersion` stays
`1.6`); module PATCH.

- Every requirement now carries an `asil` field, **inherited from the safety
  goal it implements** (SG-001/005 → ASIL-C; SG-002/003/004/006 → ASIL-B);
  requirements not allocated to a safety goal are **QM** (quality-managed).
  Cybersecurity requirements are QM on the ASIL axis (their axis is the
  cybersecurity assurance level, tracked via the TARA).
- Distribution: 12 ASIL-C, 23 ASIL-B, 46 QM. 35 requirements now carry a
  `safety_goal` link (up from 8), so ASIL is traceable rather than asserted.
- This aligns RELAY's requirement schema with the x-Net libraries (which carry
  per-requirement `asil`) and lets `gofusa` report the worst-case tool ASIL.

## v1.6.2 — 2026-06-19 (tooling/tests; no normative change)

Test-coverage maximisation. Specification unchanged (`SpecVersion` stays `1.6`);
module PATCH.

- **Coverage:** every canonical-type package (`can`, `dds`, `lin`, `mqtt`,
  `rcp`, `someip`) is now at **100%**; the root package is 98%; `cmd/relay`
  rose 83%→90% (the remainder is `main()`/`http.ListenAndServe` and
  build-metadata branches that cannot execute under `go test`). **Aggregate
  86.7% → 92.2%.**
- Added branch/error-path tests for `LoanedFrame.Return`, LIN checksum carry,
  MQTT wildcard matching, RCP/SOME/IP message-conversion error paths, the
  JSON-schema validator helpers, the CLI dispatcher, and the
  compare/probe/report renderers.
- **CI gates raised:** total coverage **80% → 90%**, plus an **85% per-package
  floor** (was 80%). All new requirement-tests remain traced.

## v1.6.1 — 2026-06-19 (tooling/evidence; no normative change)

Requirements-traceability, test-coverage, and cybersecurity hardening. The
specification text is unchanged (`SpecVersion` stays `1.6`); this is a module
PATCH.

- **Traceability:** closed the 6 untraced requirements — REQ-047/048/049 (CLI
  document field validation) and REQ-079 annotated on the conform validators and
  tests; REQ-050 (HARA) and REQ-045/046 (the spec's §7 constructor contract,
  levied on implementations) traced to the embedded HARA and specification with
  content tests. All requirements are now traced **and** tested; CI gates this
  via `gofusa trace -req-coverage 100`.
- **Cybersecurity:** added 5 cybersecurity requirements (REQ-077…081) derived
  from the TARA mitigations M-001…M-005 (build provenance, dependency-free build
  + SBOM, structural document validation, tamper-evident audit pack,
  least-privilege probing), each traced to code and tested. CI now runs
  `gofusa cyber` explicitly alongside the CYBER static-analysis rules in
  `gofusa check`.
- **Coverage:** raised `rcp` 52%→98%, and `relay`/`lin`/`someip` above 80%;
  aggregate 82.7%→86.7%. CI adds a **per-package 80% floor** so a weak unit can
  no longer hide behind high-coverage packages.
- **Requirement metadata:** every requirement now carries a `verification`
  method; safety and cybersecurity requirements link to their safety goal /
  threat. The normative specification is embedded as the `specification`
  evidence artifact and bundled by `relay audit-pack`.
- **Known limitation:** `gofusa` reports `Sec-Tested: 0` because it measures
  *independent* (different-author) verification, which a single-maintainer
  project cannot satisfy; this is documented in `docs/asil-d-uplift.md`.

## v1.6 — 2026-06-18 (stable)

Cross-implementation interoperability harness. Additive (MINOR release);
specification only — the `relay interop` command and the per-implementation
`convert` driver are tracked as issues.

- New optional CLI command `convert --protocol P` (§11.2): reads a canonical-type
  value as JSON on stdin, runs it through the implementation's own `ToMessage()`,
  and writes the resulting `relay.Message` as JSON — the black-box driver surface
  for interop testing.
- New tooling command `interop <binaryA> <binaryB> …` (§11.2.1): feeds shared
  golden vectors to each implementation via `convert`, normalises timestamps, and
  produces a pairwise equivalence matrix with field-level diffs. The canonical
  `relay.Message` is the cross-language equality oracle, so conforming
  `cpp-CAN`/`rust-CAN`/`go-CAN` MUST produce identical output for identical input.
- Complements `compare` (declared-capability interchange) with behavioural
  interchange.
- `SpecVersion = "1.6"`.

## v1.5 — 2026-06-18 (stable)

Certification uplift evidence path. Documentation/evidence only — no normative
or API change (MINOR release).

- Added `docs/asil-d-uplift.md`: the evidence path from the current ISO 26262
  ASIL-C / TCL2 qualification to ASIL-D and DO-178C DAL-A (via DO-330), with a
  gap analysis (coverage, independence, fault injection, formal methods,
  configuration management) and a tool-error Detection/Error-Measure (TD/EM)
  table mapping each HARA hazard to its detection measure and evidence.
- Embedded as the `asil-d-uplift` evidence artifact and bundled by
  `relay audit-pack`.
- Tool safety manual refreshed: corrected stale limitations (relay conform is
  shipped), evidence index now lists REQ-RELAY-001…076, TARA and the formal
  model; new §8 documents the qualification level and uplift path.
- `SpecVersion = "1.5"`; REQ-RELAY-076 traced and tested.

The document is explicitly a **path**, not a claim that RELAY is currently
qualified at ASIL-D / DAL-A; the uplift work items are tracked as issues when
scheduled.

## v1.4 — 2026-06-18 (stable)

Formal verification of the §6 node lifecycle. Additive (MINOR release).

- Added a TLA+ model `docs/formal/RelayLifecycle.tla` (+ `RelayLifecycle.cfg`,
  `README.md`) that model-checks the §6 lifecycle as a state machine: TLC
  verifies invariants for zero-value safety, send/receive-after-close,
  channels-closed-on-close, and the no-auto-reconnect policy.
- `docs/formal/README.md` gives the full requirement→invariant mapping for all
  ten §6 requirements.
- The model and its documentation are embedded in the binary as evidence
  (`relay.Evidence("formal-model")`, `"formal-model-doc"`) and bundled by
  `relay audit-pack`.
- New spec §6.1; `SpecVersion = "1.4"`; REQ-RELAY-074/075 traced and tested
  (`TestFormalModelCoversLifecycle` asserts the mapping covers §6.1…§6.10).

## v1.3 — 2026-06-18 (stable)

C++ reference binding. Documentation-only — no normative or Go API change, so
this is a MINOR release.

- §18.2 (C++) completed: the `relay.hpp` header-only layout, the core types
  (`Protocol`, `Version`, `Message`), and **all six** canonical frame types
  (`can`, `dds`, `lin`, `mqtt`, `rcp`, `someip`) with their enums, constants,
  and validators — including the v1.1 CAN XL fields.
- C++ types reuse the §18.2 `to_message`/`from_message` convention with the
  §15.7 Meta-key mappings identical to Go and Rust (cross-language trace
  interchangeability across all three reference languages).
- `SpecVersion = "1.3"`.

The `relay.hpp` header implementation is tracked as a RELAY issue (spec defines
the binding; the header is a separate deliverable).

## v1.2 — 2026-06-18 (stable)

Rust reference binding. Documentation-only — no normative or Go API change, so
this is a MINOR release.

- §18.3 (Rust) completed: the `relay-rs` crate layout, the core types
  (`Protocol`, `Version`, `Message`), and **all six** canonical frame types
  (`can`, `dds`, `lin`, `mqtt`, `rcp`, `someip`) with their enums, constants,
  and serde field mappings — including the v1.1 CAN XL fields.
- Rust `to_message`/`from_message` conversion contract documented, with the same
  Meta-key field mappings as Go so traces are interchangeable across languages.
- `SpecVersion = "1.2"`.

The `relay-rs` crate implementation is tracked as RELAY issue (spec defines the
binding; the crate is a separate deliverable).

## v1.1 — 2026-06-18 (stable)

First protocol extension. Fully additive over v1.0 — no breaking changes to any
stable surface, so this is a MINOR release.

**CAN XL (ISO 11898-1:2024):**
- `can.Frame` gains `XL`, `SDT`, `VCID`, `AF`, and `SEC` fields for the CAN XL
  format (payloads up to 2048 bytes; 11-bit Priority ID carried in `ID`).
- `can.Frame` gains `ESI` (Error State Indicator), valid for CAN-FD and CAN XL.
- New limits `CANXLMinDataLen` (1), `CANXLMaxDataLen` (2048), `CANXLMaxPrioID`
  (0x7FF), and a format-aware `Frame.MaxDataLen()` method (`MaxDataLen(fd bool)`
  is retained for back-compat).
- `ValidateFrame` rejects: FD and XL both set; ESI without FD/XL; and XL frames
  that set Ext/RTR/BRS, exceed the 11-bit Priority ID, or fall outside the
  1…2048-byte payload range.
- `ToMessage`/`FromMessage` round-trip the new fields losslessly via `can.esi`,
  `can.xl`, `can.sdt`, `can.vcid`, `can.af`, `can.sec` Meta keys (emitted only
  when set, so classic/FD frame output is unchanged).
- Updated `spec/schemas/can-frame.json`; new golden vector `can-xl-frame` and
  error vectors `can-fd-xl-mutually-exclusive`, `can-xl-priority-id-overflow`.

**Evidence:** requirements extended to REQ-RELAY-001…073 (new REQ-RELAY-070…073
for CAN XL/ESI), all traced and tested.

**Implementation note:** the CAN XL transceiver, segmentation, and `Adapt()`
work lives in the x-CAN implementations (go-CAN / rust-CAN / cpp-CAN), tracked
as issues there.

## v1.0 — 2026-06-17 (stable)

First **stable** release. No normative changes from v0.3; this release promotes
the specification and the Go module API to v1.0 and establishes the stability
guarantee.

**Stability guarantee:**
- The canonical types (§15), application interfaces (§10), error sentinels (§5),
  and CLI document schemas (§12) are now stable. Breaking changes to any of them
  require a MAJOR version increment (v2.0).
- Additive changes (new optional fields, new protocols, new CLI commands) ship in
  MINOR releases; clarifications and fixes in PATCH releases.
- `relay conform` validates any binary against the §12 schemas without source
  access; the full conformance surface is machine-checkable.

**Evidence:** requirements REQ-RELAY-001…069 are traced and tested; HARA
(`.fusa-hara.json`), TARA (`.fusa-tara.json`), and the tool safety manual are
maintained and bundled by `relay audit-pack`.

---

## v0.3 — 2026-06-16 (draft)

Incremented from v0.2. Contains a breaking change to the SOME/IP `Message`
`Meta` format; additive changes elsewhere. SOME/IP implementations MUST update
their `ToMessage()` / `FromMessage()` mappings before declaring
`"spec_version": "0.3"`.

**Breaking changes:**
- §15.7.6 / §4.3: SOME/IP `Meta["someip.msg_type"]` now carries the **numeric**
  `MessageType` (decimal uint8) instead of the string name, so the round-trip is
  lossless. The human-readable label moves to `Meta["someip.msg_type_name"]`
  (diagnostic only; ignored by `FromMessage`). `ToMessage()` now also emits
  `someip.client_id` and `someip.session_id`, and `FromMessage()` restores
  `ClientID`, `SessionID`, and `MessageType`. The conversion is now lossless per
  §15.7 (hazard H-002).

**Additive changes:**
- §14.1: `WithTopic(name string) SubscriberOption` and `SubscriberConfig.TopicName`
  added — DDS adapters read it to route subscriptions to a topic; all other
  adapters ignore it (resolves RELAY issue #13)
- `spec/schemas/`: JSON Schema (draft 2020-12) published for every canonical type
  (§15) and every CLI document (§12.1 version, §12.2 capabilities, §12.3 status,
  conform-result). Embedded in the `relay` binary and exposed via `relay.Schema`
- `spec/vectors/`: golden reference vectors for every canonical type (deterministic
  `ToMessage()` output) and error-condition vectors under `spec/vectors/errors/`
- `relay conform` now validates target output against the embedded §12 schemas

---

## v0.2 — 2026-06-16 (draft)

Incremented from v0.1. Contains breaking changes to CAN and LIN interface
signatures; additive changes elsewhere. Implementations targeting v0.1 MUST
update their `Subscribe` signatures before declaring `"spec_version": "0.2"`.

**Breaking changes:**
- CAN `Bus.Subscribe` signature changed from `Subscribe(filters ...Filter)` to
  `Subscribe(filters []Filter, opts ...SubscriberOption)` — separates content
  filtering from channel delivery configuration (§8.1)
- LIN `Bus.Subscribe` signature changed identically (§8.3)

**Additive changes:**
- §1.1: Scope boundary table — what belongs in RELAY vs each x-Net implementation
- §6.10: Reconnection policy — implementations MUST NOT reconnect automatically
- §8.3: `MasterBus.SetSchedule(entries []ScheduleEntry) error` added to LIN
- §10.5: `Adapt()` goroutine model — lifecycle, back-pressure, channel ownership
- §13.5: Docker image base standardised (`golang:1.25-alpine` / `alpine:3.20`)
- §13.6: Package layout — interface types live in x-Net, not re-exported from RELAY
- §15.7: Complete `ToMessage()` / `FromMessage()` field mappings for all 6 protocols
- §18.2: `relay::SubscriberOptions` C++ type defined with concurrency note
- §18.3: `SubscriberOptions` Rust type defined
- Appendix A: CAN/LIN Subscribe breaking-change rows added; SetSchedule gap tracked
- Out-of-scope items explicitly listed in §1: wire formats, SOME/IP-SD, security,
  `relay conform` CLI internals

---

## v0.1 — 2026-06-16 (draft)

Initial draft. Derived from go-CAN, go-DDS, go-LIN, go-mqtt, go-RCP,
go-SOMEIP, and cpp-RCP at their current HEAD revisions.

**Established:**
- Protocol integer enum (CAN=1 … SOMEIP=6)
- Universal `Message` envelope with per-protocol ID mapping and Meta keys
- Four common error sentinels: `ErrClosed`, `ErrNotConnected`, `ErrTimeout`, `ErrPayloadTooLarge`
- Six lifecycle invariants (idempotent close, send-after-close, concurrent close, etc.)
- Constructor contract (Form 1–3, mock sub-package requirement)
- Per-protocol interface contracts: `Bus` (CAN, LIN), `Participant`/`Publisher`/`Subscriber` (DDS),
  `Client`/`Subscription` (MQTT), `Controller`/`Registry`/`LoaningController` (RCP),
  `Service`/`Server`/`Subscription` (SOMEIP)
- Optional interfaces: `LoaningBus` (CAN), `LoaningPublisher` (DDS), `HealthProvider`,
  `MetricsProvider`, `Drainer` — all protocols
- CLI contract: `version`, `capabilities`, `status`, `connect`, `send`, `subscribe`
- Capability discovery document schema (§11)
- Subscriber defaults: depth=64, back-pressure=DropNewest, `ApplySubscriberOpts`, `ChanDepth`
- Canonical frame types for all six protocols including `GUID`, `BackPressurePolicy`,
  TSN QoS fields, SOME/IP-TP variants, `Loan.Return()`, `MaxDataLen()`
- `relay::Context` C++ type formally defined
- Rust async-primary model decided
- Extension mechanism for new protocols (§3)
- Deprecation policy: minimum one MINOR version notice before removal
- Compatibility range syntax: `"spec_version": "0.1"` in capabilities document
- Application interface: `relay.Node` (pub/sub) and `relay.Caller` (request/response)
  with `Adapt()` contract and routing rules per protocol (§10)
- Cross-language binding for `relay.Node` and `relay.Caller` in C++ and Rust (§18)
- `"adapt": true` conformance flag in capabilities document (§12.2, §17 req 6)
- RELAY vs x-Net scope boundary table (§1.1)
- Reconnection policy: no automatic reconnect; return `ErrNotConnected` (§6.10)
- CAN/LIN `Subscribe` signature resolved: `Subscribe(filters []Filter, opts ...SubscriberOption)` — breaking change from current go-CAN/go-LIN (Appendix A)
- LIN `MasterBus.SetSchedule(entries []ScheduleEntry) error` added (§8.3)
- `Adapt()` goroutine model: lifecycle, back-pressure, channel ownership (§10.5)
- Complete `ToMessage()` / `FromMessage()` field mappings for all 6 protocols (§15.7)
- `relay::SubscriberOptions` type defined for C++ and Rust (§18.2, §18.3)
- Package layout clarified: interface types live in x-Net, not re-exported from RELAY (§13.6)
- Docker image base standardised: `golang:1.25-alpine` build, `alpine:3.20` runtime (§13.5)
- Out-of-scope items explicitly listed: wire formats, SOME/IP-SD, reconnection, security, `relay conform` CLI internals
