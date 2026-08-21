# RELAY Architecture

This document is a map of this repository, not a normative source. The
normative source is [`spec/relay-spec.md`](spec/relay-spec.md); if anything
here disagrees with it, the spec wins.

## What RELAY is

RELAY is the shared specification and canonical-type library for the
SoundMatt embedded network protocol ecosystem: CAN, DDS, LIN, MQTT, RCP, and
SOME/IP, each implemented independently in Go, C++, Rust, and (RCP only) C.
It defines:

- **Canonical frame/message types** per protocol, with lossless conversion
  to and from a common `relay.Message` envelope (§15).
- **Application interfaces** (`relay.Node`, `relay.Caller`) that let a
  consumer talk to any conformant protocol implementation without knowing
  which protocol it is (§10).
- **A conformance contract** — what a protocol implementation must
  provide, and expose through a `version`/`capabilities`/`status` CLI, to
  be RELAY-conformant (§17, §11-12).

RELAY does **not** implement a network driver and does not itself transmit
messages to a bus — it defines the contract; the protocol-specific `go-*`,
`cpp-*`, `rust-*`, `c-RCP` repos fulfil it (see the README's protocol-coverage
table).

## Repository layout

```
spec/            The normative specification (relay-spec.md), its
                 CHANGELOG, JSON schemas, and golden conformance vectors.
can/ dds/ lin/    Canonical Go types, ValidateFrame, and ToMessage/
mqtt/ someip/     FromMessage for each protocol (§15).
rcp/
router/          relay.Node/relay.Caller application-interface plumbing
                 and the crossbar message router (§10, §11.2.1's
                 "crossbar").
cmd/relay/       The relay CLI: version/capabilities/status/conform/
                 interop/convert/crossbar/probe/trace/report/sbom/
                 safety-case/audit-pack/compare/versions/serve.
docs/            Non-normative reference material:
                 - RCP-ARCHITECTURE.md: the canonical cross-language
                   architecture the four RCP ports (go/c/cpp/rust-RCP)
                   converge on.
                 - KNOWN_GAPS.md: specific implementations' current,
                   independently-tracked deviations from the spec.
                 - tool-safety-manual.md, asil-d-uplift.md: ISO 26262
                   tool-qualification evidence for RELAY itself.
                 - formal/: TLA+ models checked against §6's lifecycle
                   invariants.
.fusa-*.json     x-FuSa safety/security evidence: requirements registry,
.fusa.json       HARA, TARA — see tool-safety-manual.md §7.
```

## The x-FuSa safety-evidence layer

Every requirement-bearing behavior in this repo is tagged with a
`//fusa:req REQ-RELAY-NNN` at its implementation site and a matching
`//fusa:test REQ-RELAY-NNN` at the test verifying it, with a corresponding
entry in `.fusa-reqs.json`. CI's `gofusa` pipeline gates on 100% requirement
traceability, 0 ERROR-severity findings, and generates the ISO 26262 /
ISO 21434 evidence bundle `relay audit-pack` ships. See
`docs/tool-safety-manual.md` for the full qualification scope and
`docs/asil-d-uplift.md` for the path beyond RELAY's current ASIL-C/TCL2
qualification.

## Related documents

- [`spec/relay-spec.md`](spec/relay-spec.md) — the normative specification.
- [`docs/RCP-ARCHITECTURE.md`](docs/RCP-ARCHITECTURE.md) — the shared
  architecture the four RCP implementation repos converge on (a design
  target for those repos, not for this one).
- [`CONTRIBUTING.md`](CONTRIBUTING.md) — how to propose and land changes.
- [`SECURITY.md`](SECURITY.md) — how to report a vulnerability.
