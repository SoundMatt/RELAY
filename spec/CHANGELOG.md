# RELAY Spec Changelog

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
