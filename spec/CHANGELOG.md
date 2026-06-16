# RELAY Spec Changelog

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
