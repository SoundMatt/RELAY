# RELAY Spec Changelog

## v2.7 — 2026-08-21 (MINOR — new protocol/model retirement process, §17 Requirement 17)

- **New §3.2 "Retiring a protocol or model".** §19.2's existing deprecation
  policy governs only a single MUST requirement's removal; it had nothing
  to say about retiring a whole canonical model (the way §15.5's RCP types
  were rewritten for TC18 conformance). Defines a `retired[]`/`deprecated[]`
  array pair for `spec/version.json`, each entry a `{name, since, removal,
  reason}` record — `name` matching what an implementation would otherwise
  advertise in `features`/`commands` (§12.2).
- **New §17 Requirement 17 — no retired capabilities past removal.** A
  capabilities document whose declared `spec_version` is at or past a
  `retired[]` entry's `removal` version MUST NOT still list that entry's
  `name`. Unlike Requirements 13–16, this is fully within `relay conform`'s
  own reach: the check only needs two things it already fetches over the
  CLI (declared `spec_version`, declared `features`/`commands`) compared
  against `relay conform`'s own embedded `spec/version.json` — the same
  authoritative source §19.4 already points to. Full black-box coverage,
  like Requirement 7.
- **`retired[]`/`deprecated[]` start empty, deliberately not backfilled.**
  RELAY's own pre-TC18 RCP placeholder-model replacement (§15.5, v2.0)
  predates this section: it was an instant MAJOR-version breaking
  replacement with no compatibility window, not a `since`/`removal`-spaced
  retirement this mechanism is designed to track. Force-fitting it into
  `retired[]` would misrepresent history as having had a graceful
  deprecation window it never had. This section governs retirements
  declared from here onward.
- **Bug found and fixed while wiring Requirement 17's manifest entry**:
  `buildManifest`'s Requirement 6 and Requirement 1 statuses were computed
  from `hasFail(cFindings)`/an incomplete filter that didn't account for
  the capabilities command failing to run at all (§17.7) as a superset
  failure — a gap that would have let Requirement 17's new, differently-
  cited findings silently miscolor Requirement 6's own status once mixed
  into the same findings list. Introduced a shared `capsUnreachable` check
  used by Requirements 1, 6, and 17 alike; mutation-tested (reverted,
  confirmed `TestBuildManifestFailPropagatesToOverall` failed, restored).
- **Reference implementation**: `checkRetiredCapabilities`/
  `specVersionAtLeast`/`loadRetiredCapabilities` in `cmd/relay/conform.go`,
  wired into `validateCapabilitiesDoc`. New `REQ-RELAY-099`. 8 new tests,
  including a real mutation test on the check's core comparison logic.
- `SpecVersion` bumped `2.6` → `2.7` (MINOR, new §17 requirement). Closes
  [NEW-SPEC-5].

## v2.6 — 2026-08-21 (MINOR — new release conformance attestation format)

- **New §20.6 — Release conformance attestation (`relay-conform-attestation/1`).**
  Defines the *shape* an implementation MAY publish to make §20.1's CI
  gates externally verifiable rather than self-asserted: an
  [in-toto](https://in-toto.io/) Statement whose `subject` digest is the
  SHA-256 of the attested binary file, and whose predicate bundles the
  full §17.2 conformance manifest, the attestation-generating tool's own
  embedded `vectors_version` (§15.8 — explicitly **not** observed from the
  attested binary, which a black-box invocation cannot introspect), and an
  optional implementation-defined `safety_evidence_summary`.
- **Scope decision (documented, not silently narrowed)**: this is a
  spec-only release. The originating issue proposed cryptographic
  signing, publishing the attestation alongside a release's container
  image, surfacing its digest in `version --format json`, and having
  `relay probe`/`relay report` treat an unverifiable release as
  non-conformant. None of that ships here: RELAY's own CI does not
  currently push container images to any registry, and fabricating a
  "signature" without real key management would be worse than not
  signing at all. `predicate.signed` MUST be `false` on an unsigned
  attestation; this section defines shape only and does not mandate a
  signing mechanism or publication channel. Closing this properly is
  future work.
- **New embedded schema** `spec/schemas/relay-conform-attestation.json`.
- **Reference implementation**: `relay conform --attestation <binary>` in
  `cmd/relay/conform.go`, generating the unsigned predicate for a given
  binary. New `REQ-RELAY-098`.
- **Bug found and fixed while implementing this**: `buildManifest`'s
  `Requirements` slice was missing entry 16 (Vector manifest) — PR #181
  added the §17 Requirement 16 spec text and the verifier-table row but
  never added the corresponding code entry, so every `relay-conform/1`
  manifest generated since v2.4 silently omitted it despite the spec's
  own text mandating "exactly one entry per §17 requirement (1–16, ...)".
  Fixed; both manifest-shape tests that should have caught this
  (`TestBuildManifestSelf`, `TestRunConformManifestFlag`) were also
  stale at a hardcoded `15` and are corrected alongside it.
- `SpecVersion` bumped `2.5` → `2.6` (MINOR). Closes [NEW-SPEC-4] (partial
  — signing, publishing, and probe/report enforcement remain open).

## v2.5 — 2026-08-21 (MINOR — new optional capabilities field, tightened §17 Requirements 1 and 6)

- **New optional `multi_protocol` capabilities field (§12.2).** Defaults to
  `false` when absent. A tool that self-declares `multi_protocol: true`
  legitimately reports a null `protocol`/`protocol_int` and `adapt: false`:
  §10.3 scopes the `Adapt()` contract to protocol packages, so a
  multi-protocol aggregator (like RELAY's own reference CLI) has no single
  protocol to declare or per-protocol adapter to export.
- **§17 Requirements 1 and 6 tightened from WARN to FAIL.** A null
  `protocol`/`protocol_int`, or `adapt: false`, on a capabilities document
  that does not declare `multi_protocol: true` is now a conformance FAIL,
  closing a real audit-flagged gap (THEME-B, capabilities drifting silently
  from the shipped binary) — previously both were only WARN, so an
  implementation using `relay conform` without `--strict` never failed on
  either. Both requirements move from "Partial" to "Full" black-box coverage
  in the requirement-to-verifier table.
- **Scope decision**: the originating issue also proposed verifying every
  declared `commands` string is actually invocable, and "exercising" every
  declared `features` string with a CLI probe. Both are explicitly declined
  in this release: there is no existing, spec-grounded signal distinct from
  the generic "invalid arguments" exit code (§11.3) to detect an unrecognized
  command across four languages' worth of implementations, and §12.2 already
  states `features` are compiled-in and explicitly not runtime-probed. Adding
  either would mean inventing an unproven new convention under this issue's
  scope rather than tightening an existing one — deferred to a dedicated
  follow-up.
- **Design note**: the naive implementation (blindly turning both WARN cases
  into FAIL) would have broken RELAY's own reference CLI's passing
  self-conformance CI job, which is deliberately, legitimately
  multi-protocol and non-adapting. `multi_protocol` exists specifically to
  let `relay conform`'s black-box CLI distinguish that legitimate case from
  a genuine single-protocol implementation bug, which it previously could
  not do at all.
- **Reference implementation**: `cmd/relay`'s own `capabilities` output now
  declares `"multi_protocol": true`; `validateCapabilitiesDoc` in
  `cmd/relay/conform.go` implements the gated FAIL logic for both fields.
  New `REQ-RELAY-097`. `SpecVersion` bumped to `2.5`. Closes [NEW-SPEC-3]
  (partial — commands-invocability and feature-probing deferred).

## v2.4 — 2026-08-21 (MINOR — new §17 conformance requirement)

- **New §17 Requirement 16 — Vector manifest.** The canonical
  `spec/vectors/` distribution (new §15.8) is pinned by
  `spec/vectors/vectors_manifest.json`. A conformant implementation that
  embeds a local copy of these vectors MUST embed the exact pinned set,
  and MUST have a CI step that fails when its embedded copy's SHA-256
  diverges from the published manifest for the `vectors_version` it
  targets. Same MINOR-worthy precedent as Requirements 13, 14, and 15.
- **New §15.8 — Vector distribution.** Defines the `relay-vectors/1`
  manifest shape (`kind`, `manifest_version`, `vectors_version`,
  `vectors[]` — each a `{name, sha256}` pair). New embedded schema
  `spec/schemas/vectors-manifest.json`. Explicitly scoped to
  *distribution* of the existing envelope-level `relay.Message` fixtures
  only: per §1.1/§17.1, wire format remains out of scope, and this
  manifest MUST NOT be extended with byte-for-byte wire-format vectors.
  (NEW-SPEC-1's originating issue proposed extending the vector set
  itself to wire-format bug classes; that part is explicitly declined as
  contradicting RELAY's own pre-existing scope boundary, established in
  REL-SPEC-2/§17.1 — only the manifest/pinning/hashing mechanism ships
  here.)
- **New §20.1 CI gate**: recompute the SHA-256 of every embedded golden
  vector and fail the job on any divergence from
  `spec/vectors/vectors_manifest.json` for the `vectors_version` it
  targets.
- **Reference implementation**: `spec/vectors/vectors_manifest.json`
  (16 vectors, hashes independently verified via `sha256sum`),
  `VectorsManifest()`/`ParsedVectorsManifest()`/`VerifyVectorManifest()` in
  this repo's own `vectors.go`. `VectorNames()` now excludes the manifest
  file itself from the vector set it enumerates (it is metadata about the
  distribution, not a vector) — this also required a matching fix in two
  pre-existing test files (`spec_vectors_test.go`,
  `cmd/relay/jsonschema_test.go`) that globbed `spec/vectors/*.json`
  directly and would otherwise have tried to parse the manifest as a
  golden vector. New `REQ-RELAY-096`. `SpecVersion` bumped to `2.4`.
  Closes [NEW-SPEC-2].

## v2.3 — 2026-08-21 (MINOR — new §17 conformance requirement)

- **New §17 Requirement 15 — Conformance manifest.** The implementation
  MUST commit a `relay-conform/1` manifest (new §17.2), generated by
  `relay conform --manifest` against its own built binary; CI MUST
  regenerate the manifest on every change and fail the build on any diff
  from the committed copy, or on any requirement entry whose `status` is
  `FAIL`. New MINOR-worthy Requirement, following the same precedent as
  Requirements 13 (v2.1) and 14 (v2.2): a numbered §17 requirement is
  MINOR-worthy even where `relay conform` cannot fully observe it.
- **New §17.2 — Conformance manifest (`relay-conform/1`).** Defines the
  manifest JSON shape (`kind`, `manifest_version`, `tool`,
  `binary_version`, `spec_version`, `git_sha`, `capabilities_sha256`,
  `requirements[]`, `overall`) and a 4-value status vocabulary
  (`PASS`/`FAIL`/`SHAPE_ONLY`/`NOT_OBSERVABLE`) deliberately distinct from
  `relay conform`'s existing `PASS`/`WARN`/`FAIL` finding severities: a
  manifest MUST NOT report `PASS` for a requirement `relay conform`
  cannot actually observe. New embedded schema
  `spec/schemas/relay-conform-manifest.json`.
- **New optional `commit` field on the §12.1 version document** — the git
  SHA (short or full) the binary was built from, if the build embeds one.
  Absent or empty when unavailable; this is not itself a conformance
  failure, but it is what lets a manifest populate its own `git_sha`
  field. Backward-compatible: not added to the schema's `required` array.
- **New §20.1 CI gate**: regenerate the manifest and fail the job on any
  diff from the committed copy, or on any requirement entry whose
  `status` is `FAIL`.
- **Reference implementation**: `relay conform --manifest <binary>` in
  this repo's own CLI (`cmd/relay/conform.go`). Only Requirements 1, 6, 7,
  12 get real `PASS`/`FAIL`/`SHAPE_ONLY` statuses, mirroring §17's
  existing narrative of what `relay conform` can observe black-box;
  everything else — including Requirement 15's own manifest-generation
  entry — is honestly `NOT_OBSERVABLE`, per this release's own new rule.
  `SpecVersion` bumped to `2.3`. Closes [NEW-SPEC-1].

## v2.2.4 — 2026-08-21 (doc addition; no normative change to existing conformant implementations)

- **Fixed a real spec-internal inconsistency**: §8.3's Go `CalcChecksum`
  signature used a bare `ChecksumType` parameter type, contradicting
  §15.3's own canonical Go name `LINChecksumType` for the identical
  concept — found while investigating this issue's underlying research.
  Fixed to `LINChecksumType`, matching §15.3.
- **New §13.7.4 — Standard type-name registry**, extending §13.7.2's
  module-name-registry pattern to exported *types*. Independently
  re-verified across the actual ~17 x-Net repos (not just the audit's own
  claim) which of the finding's eight named concepts are genuinely
  divergent: four are (LIN checksum selector, ISO-TP connection, J1939
  bus/frame/PGN/priority, the in-process virtual-bus type); the DDS QoS
  preset constants turned out to differ only in per-language idiomatic
  casing (already tolerated by §13.7.2's own "idiomatic packaging aside"
  carve-out — not a real divergence); the MQTT health-status
  enum/struct-inversion claim was independently re-verified against all
  18 repos and found to be **stale** — every repo already pairs
  `HealthStatus` (enum) with `Health` (struct) consistently, no exception
  found; `SubscriberConfig`/`SubscriberOption` is already governed by
  §14.1, not duplicated; and the RC Server general-register-block naming
  is deliberately **excluded** — its current divergence includes a real
  wire-shape difference (single `u32` vs. split major/minor `u8` pair),
  not just a naming one, making it `docs/RCP-ARCHITECTURE.md`'s scope to
  reconcile, not a plain naming registry's.
- **New §13.7.2 keyword-escape clause**: `virtual` is a reserved word in
  both C++ and Rust — the existing module-name mandate was, as literally
  written, unsatisfiable in two of the three languages it applies to.
  New sentence: where the mandated name collides with a language keyword,
  the implementation MUST use the closest available escape (`virt` for
  `virtual`) rather than inventing an unrelated name.
- **Not part of the §17 conformance gate**, matching every other §13.7.x
  naming convention (including the pre-existing module-name registry):
  `relay conform`'s black-box CLI cannot introspect exported type names
  over a CLI interface. `SpecVersion` unchanged (`2.2`). Closes
  [REL-SPEC-4].

## v2.2.3 — 2026-08-21 (doc addition; no normative change to existing conformant implementations)

- **Fixed stale example literals.** §12.1/§12.2's `spec_version` JSON
  examples and §13.5's Docker `LABEL io.relay.spec-version` example still
  showed `"0.1"`, three major and several minor versions behind current —
  confusing as a worked example even though the surrounding prose already
  called it out as illustrative. Updated to `"2.2"`.
- **`docs/tool-safety-manual.md` header and CLI-coverage fix.** The
  document's own header block still said "Spec version: v0.2"; updated to
  v2.2. Separately, §2's tool-overview bullet listed only 3 of the CLI's
  now-16 subcommands (`version`, `capabilities`, `status`) — independently
  re-verified against `cmd/relay/main.go`'s actual command list before
  fixing (the audit flagged this claim FILED_UNCERTAIN, not confirmed).
  New bullet lists all 16, split into the 10 safety-relevant to conformance
  verification (§3.4) and the 6 that are convenience/dev tooling outside
  that path.
- **New `CONTRIBUTING.md`, `SECURITY.md`, `ARCHITECTURE.md`.** None
  existed despite this repo's own safety documentation making ASIL-C
  claims elsewhere. `CONTRIBUTING.md` covers the spec-PR-before-implementation
  workflow (§5.5 and similar), the §19 versioning decision, DCO sign-off,
  and requirement-tag discipline. `SECURITY.md` points to GitHub private
  vulnerability reporting — enabled on this repository as part of this
  change, since a policy pointing at a disabled feature would be exactly
  the kind of drift this fix is closing. `ARCHITECTURE.md` maps the
  repository layout and the x-FuSa safety-evidence layer, each claim
  checked against the actual current tree before writing it down (e.g. the
  crossbar router's actual section citation, §11.2.1, not guessed).
- **Deliberately out of scope**: `docs/RCP-ARCHITECTURE.md` and the
  ROADMAP.md phase-count/`v1.14`/`v2.0` narrative gap are already tracked
  separately (issue #79); not duplicated here, per this finding's own text.
  `.fusa-reqs.json`'s exact current `REQ-RELAY-NNN` range (the manual's §7
  Evidence table cites `001..081`, likely stale given multiple releases
  since) was not independently re-verified in this pass — flagging rather
  than silently leaving a claim in place that this PR didn't actually
  check. Closes [REL-SPEC-11].
- `SpecVersion` unchanged (`2.2`); every change here is either an example
  literal, a non-normative reference doc's header, or a new non-normative
  governance document — nothing in §17's conformance requirements changed.

## v2.2.2 — 2026-08-21 (doc addition; no normative change to existing conformant implementations)

- **New §17 requirement-to-verifier lookup table.** §17 already narrated,
  in prose, which of its 14 requirements `relay conform` can verify and
  which it can't — but the narrative was scattered across four paragraphs,
  with no single place to look up "does `relay conform` check Requirement
  N, and how?" New table collects it: one row per requirement, its
  verifier, and its coverage (full / partial / shape-only / not observable
  through the CLI). Purely descriptive — collects existing prose into a
  table, adds no new requirement and changes no existing one.
- **New §12.4 combined walkthrough.** §12.1–12.3 each show one document's
  JSON schema in isolation, as three unrelated fragments for hypothetical
  different tools. New subsection runs `version` → `capabilities` →
  `status` against the same example binary (`go-can`) in one sequence, so
  the fields shared across all three documents (`tool`, `version`,
  `spec_version`) and the superset relationship between `version` and
  `capabilities` are visible together, the way a reader actually
  encounters them at a real CLI. Supplements §12.1–12.3, doesn't replace
  them — each subsection's own schema reference is still the place to look
  up one document's shape in isolation. Closes [REL-SPEC-9].
- `SpecVersion` unchanged (`2.2`); no requirement text changed or was
  added — this documents existing, already-shipped behavior more
  legibly, it doesn't create new behavior for any implementation to
  satisfy.

## v2.2.1 — 2026-08-21 (doc addition; no normative change to existing conformant implementations)

- **New §13.8 — README conventions.** §3.5 of the audit's terminology matrix
  found no two x-Net implementation READMEs use the same section header for
  describing their RELAY adapter ("RELAY adapter", "RELAY Integration",
  "RELAY conformance", "RELAY-conformant CLI", "RELAY compliance (vX.Y)", or
  absent entirely). New requirement: this section MUST be headed exactly
  `## RELAY conformance`; a version number, if included, belongs in the
  body, not the heading, so the heading text itself never goes stale.
- **Quickstart round-trip requirement, same subsection.** Separately, the
  audit found some ports' documented two-shell CLI quickstart demo cannot
  actually round-trip a message because the CLI spins up a fresh,
  non-shared bus per invocation (reported for rust-LIN and go-SOMEIP) — so
  following the README literally does not work. New requirement: a
  documented two-shell round-trip demo MUST actually work as written,
  either because the transport shares persistent state across invocations,
  or the README MUST NOT claim the round-trip is possible with the
  mock/virtual transport alone. Closes [REL-SPEC-10].
- **Not part of the §17 conformance gate**, consistent with every other
  §13.x naming/documentation convention (repo names, CLI binary names,
  module names, …): `relay conform`'s black-box CLI never reads README
  content, so this MUST — like the rest of §13 — sits outside the
  "RELAY-conformant if and only if" checklist. `SpecVersion` unchanged
  (`2.2`); no existing conformant implementation is made non-conformant by
  a text convention `relay conform` was never going to check.
- **Pairs with [NEW-SPEC-8]** in the same audit batch, which proposes
  making README examples CI-executed so this specific class of defect
  (a quickstart that silently doesn't work) can't ship in the first place —
  not attempted here; this PR only fixes the normative text describing what
  "works" means.

## v2.2 — 2026-08-21 (stable)

Spec-version binding — a new normative conformance requirement. Additive
(MINOR).

- **New §17 Requirement 14 — spec-version binding.** THEME-A found 13 of 18
  ecosystem ports currently declare a stale `spec_version`. §17 Requirement
  12 already required a `SpecVersion` constant "equal to the spec version
  being targeted," but its own verification narrative already conceded
  `relay conform`'s black-box CLI can only shape-check the printed string,
  never observe whether it's genuinely bound to anything — so a hand-typed
  literal that's never revisited was always indistinguishable, from the
  outside, from a correctly-maintained one. New requirement: a Go
  implementation that depends on `github.com/SoundMatt/RELAY` MUST bind its
  declared `spec_version` directly to `relay.SpecVersion` rather than
  duplicating the string — a dependency bump alone then keeps it current.
  Any implementation that can't do that (a different language, or a Go
  implementation without the RELAY dependency) MUST have a CI step that
  fails the build when its declared value diverges from the authoritative
  `spec/version.json` at the RELAY revision it targets. Closes
  [REL-SPEC-6].
- **Scoped deliberately to the normative half only.** The finding's
  recommendation also proposed generating the spec document's own version
  literals (§19.4's embedded snippets, etc.) from `spec/version.json`
  itself — that's [NEW-SPEC-6]'s own scope (RELAY's own doc-build tooling),
  not duplicated here.
- Like Requirement 13, `relay conform` cannot observe *how* a declared
  `spec_version` was produced — a value bound to `relay.SpecVersion`, one
  checked by a CI script against `spec/version.json`, and one simply typed
  once and forgotten all look identical from outside the binary — so this
  MUST is verified by the implementation's own CI, not `relay conform`.
- `SpecVersion = "2.2"` (`version.go`, `spec/version.json`,
  `version_test.go`); `cmd/relay`'s `toolVersion` bumped `2.1.0` → `2.2.0`
  per its own drift-guard test. Per §19.3, `relay conform` continues
  accepting implementations declaring earlier `spec_version`s under their
  own rules — nothing currently conformant is retroactively broken.

## v2.1 — 2026-08-20 (stable)

Capabilities generation — a new normative conformance requirement, not just
editorial. Additive (MINOR).

- **New §17 Requirement 13 / §12.2 addition — capabilities generation.**
  Ecosystem audits found at least six ports' `capabilities` documents had
  drifted from what their binaries actually ship (THEME-B) — the class of
  bug §12.2's existing "set at build time" language described but never
  actually prevented, since nothing required the value to be *generated*
  from that build-time source rather than hand-copied from it once and left
  to rot. New requirement: the `commands`, `transports`, `features`,
  `interfaces`, and `optional_interfaces` fields MUST be generated from (or
  otherwise kept in lock-step with) the same source-of-truth that gates
  compilation — build tags, a CMake target list, Cargo feature flags, C
  preprocessor macros, or the implementation's equivalent — not
  hand-maintained independently of it. Like Requirements 2–5 and 8–11,
  `relay conform`'s black-box CLI cannot observe *how* a document was
  produced, only that it's shape-valid, so this MUST is verified by the
  implementation's own build process and test suite. Closes [REL-SPEC-3].
- **Deliberately narrower than the finding's full recommendation.** The
  finding also asked for a mandated CI self-check (every dispatched
  subcommand/feature present in `capabilities` and vice versa) and a
  reference generator in the RELAY CLI. The CI self-check is [NEW-SPEC-3]'s
  own scope (tightening `relay conform`'s existing `adapt: false` WARN to a
  FAIL is the same class of change, tracked there rather than duplicated
  here). A single reference generator isn't achievable in the Go-only
  `relay` CLI for a genuinely cross-language requirement — Go build tags,
  CMake target lists, Cargo features, and C preprocessor macros are four
  different mechanisms with no common generator to write once; each
  language ecosystem needs its own pattern, which is a larger, separate
  undertaking than this spec-guidance PR.
- **Why this is MINOR, not a patch-level doc addition** (unlike the
  preceding v2.0.x series): this is a genuine new MUST-level conformance
  requirement on already-shipped implementations, several of which are
  already known (per the finding) to hand-maintain their capabilities list.
  Per §19.3, `relay conform` continues to accept implementations declaring
  `spec_version: "2.0"` under the pre-existing 12-requirement rule — no
  currently-conformant implementation is retroactively broken. Only
  implementations that update their declared `spec_version` to `"2.1"` take
  on Requirement 13.
- `SpecVersion = "2.1"` (`version.go`, `spec/version.json`,
  `version_test.go`); `cmd/relay`'s `toolVersion` bumped `2.0.4` → `2.1.0`
  to stay in sync per its own drift-guard test
  (`TestToolVersionTracksSpecVersion`).

## v2.0.10 — 2026-08-20 (doc addition; no normative change to existing conformant implementations)

- **New §17.1 — Wire-format regression coverage (recommended)**: the
  finding's own recommendation — that RELAY ship byte-for-byte wire-encoding
  conformance vectors in `spec/vectors/` for recurring bug classes (MQTT
  varint boundaries, LIN diagnostic checksum selection, RCP register-map/PWM
  field width, J1939 PGN layout, SOME/IP-SD option runs) — conflicts with
  this document's own §1 scope note: "RELAY does **not** define... [w]ire
  formats... RCP's binary frame format, SOME/IP header layout, CAN bit
  timing, etc. are defined in each x-Net implementation." `spec/vectors/`
  already covers Requirement 9 (envelope conversion) at the canonical-type
  level; wire-byte encoding is a level below that boundary, and two of the
  finding's five named examples (J1939 PGN layout, SOME/IP-SD option-run
  structure) aren't even normatively defined by this spec's own text (§1;
  §12's feature-flag treatment of J1939 as a CAN capability, not a protocol
  with its own wire-format section) for RELAY to author conformance
  vectors against in the first place — the finding's other three examples
  (MQTT varint, LIN checksum selection, RCP register-map/PWM/ISELED fields)
  are real, well-evidenced gaps, but still sit below the wire-format
  boundary §1 draws.
- The actual gap the finding's own evidence supports — no shared regression
  discipline exists against a well-evidenced recurring bug class, encode/decode
  asymmetry hidden by round-trip-only tests — is real and addressed within
  RELAY's actual boundary: new §17.1 recommends (SHOULD, not MUST — `relay
  conform`'s black-box CLI has no mechanism to observe or gate on an
  implementation's internal test suite, the same reason Requirements 2–5 and
  8–11 are MUSTs the CLI itself cannot verify) that each implementation's own
  test suite maintain byte-for-byte encode/decode regression vectors for its
  own protocol's wire-format boundary conditions, asserting both directions
  independently rather than only `decode(encode(x)) == x`. Closes
  [REL-SPEC-2].
- `SpecVersion` unchanged (`2.0`); this is a SHOULD-level recommendation with
  no `relay conform` gate, so no existing conformant implementation is made
  non-conformant by it either way.

## v2.0.9 — 2026-08-20 (doc addition; no normative change to existing conformant implementations)

- **New `docs/KNOWN_GAPS.md`; stop embedding sibling-repo bug notes in
  normative spec prose**: four inline asides — go-CAN's `Subscribe()`
  slice-vs-variadic note (§8.1), the go-CAN `ValidateFrame` RTR/FD-check
  gap (§15.1), go-LIN's `SetSchedule` note (§8.3), and go-SOMEIP's
  `SubscriberConfig`/`SubscriberOption` naming note (§14.1) — all cited a
  specific implementation's then-current deviation and pointed at
  Appendix A, which the spec's own text already marks as a historical
  snapshot "not maintained thereafter" the v0.2 milestone. Mixing
  per-implementation status into normative requirement text meant that
  text either had to be re-reviewed every time the named implementation
  shipped a fix, or — as had already happened — silently went stale
  while still reading as current spec guidance. All four asides are
  removed, leaving each site purely normative; their content moves
  verbatim into new `docs/KNOWN_GAPS.md`, an explicitly non-normative
  document with its own staleness caveat (nothing in it gates `relay
  conform` or any other CI check). Appendix A's own preamble now points
  to `docs/KNOWN_GAPS.md` for current per-implementation deviations, so
  the pointer exists exactly once rather than being repeated at every
  future implementation-status callout. Closes [REL-SPEC-7].
- `SpecVersion` unchanged (`2.0`); no MUST/SHOULD/MAY requirement text
  changed — only the removal of non-normative asides and their
  relocation to a new non-normative document. No existing conformant
  implementation is affected either way.

## v2.0.8 — 2026-08-20 (doc addition; no normative change to existing conformant implementations)

- **§13.7.1 / §18.2 — C++ protocol-interface `I`-prefix**: every existing
  cpp-* port independently prefixes its §8 protocol interfaces (`Bus`,
  `Participant`, `Client`, …) with `I` — the same convention `relay::INode`
  already establishes for the application interface — but §8's own text
  ("C++ ... equivalents are in §18") never actually showed what that C++
  equivalent looked like: §18.2 covered `relay::Node`/`Caller` and the
  canonical types, but no protocol interface at all, in either prefixed
  or unprefixed form. New §13.7.1 paragraph documents the `I`-prefix as
  permitted, idiomatic C++ practice (not a MUST — the spec's own
  canonical names stay unprefixed, matching §15's own cross-language
  convention); new §18.2 "Protocol interfaces (C++)" subsection adds the
  actual worked example (CAN's `IBus`, the simplest protocol), including
  a minimal `Filter` struct so the example is genuinely self-contained
  rather than referencing a type this spec's own C++ section didn't
  define. Closes [REL-SPEC-5]/[THEME-G].
- Deliberately scoped to the mandatory interface only, not every optional
  extension (`LoaningBus` and siblings) — a full parallel C++
  canonical-type catalogue for CAN specifically is out of this
  subsection's own scope; the same `I`-prefix-or-not latitude applies to
  those too, noted but not separately worked through.
- `SpecVersion` unchanged (`2.0`); no existing conformant implementation
  is made non-conformant — `relay conform` (§17) cannot observe a C++
  interface's own name either way, so both prefixed and unprefixed forms
  were already, and remain, equally conformant.

## v2.0.7 — 2026-08-20 (doc addition; no normative change to existing conformant implementations)

- **§5.5 — Adding a protocol-specific error post-launch**: §3.1 already
  covered adding an entirely new protocol; there was no equivalent guidance
  for the narrower, more common case of adding a new protocol-specific
  error sentinel to a protocol that has already shipped. New subsection
  gives a three-part test for when a new §5.4 table entry is actually
  warranted vs. when the condition should just wrap an existing sentinel
  (§5.2), plus the process itself: the §5.4 table row lands via a spec PR
  *before* any implementation ships the sentinel, since the table is the
  ecosystem-consistency contract a single implementation's own PR can't
  establish on its own. Closes [REL-SPEC-8].
- Motivated by a concrete instance the finding names: a shipped, exported
  go-SOMEIP error sentinel with no corresponding §5.4 row, apparently
  because no documented process existed for adding one post-launch. Only
  the spec-level process gap is addressed here — the go-SOMEIP instance
  itself (filed separately as RELAY-TERM-04) is that repo's own follow-up,
  not fixed by this change.
- Inserted as a new §5.5 subsection immediately after §5.4 (not a trailing
  appendix like [REL-SPEC-1]/v2.0.6): a decimal subsection number under an
  existing top-level section doesn't renumber anything else, so this can
  live next to the material it governs without the cross-reference-breakage
  concern a new top-level section would raise.
- `SpecVersion` unchanged (`2.0`); no existing conformant implementation is
  made non-conformant — this documents a process for future additions, it
  does not itself add, remove, or rename any interface, type, or sentinel.

## v2.0.6 — 2026-08-20 (doc addition; no normative change to existing conformant implementations)

- **Appendix B — Quickstart for Implementers**: added a non-normative,
  linear build-order walkthrough (9 steps, worked example over CAN) tying
  together the ~20 sections a conformant implementation touches
  (constructor contract §7, protocol interfaces §8, error sentinels §5,
  lifecycle §6, `Adapt()` §10.3-10.5, canonical types §15, subscriber
  helpers §14.1, CLI contract §11, capability discovery §12, conformance
  §17, continuous conformance §20). Every requirement it walks through is
  cited to its own authoritative section — the appendix restates nothing
  and adds no new MUST/SHOULD text of its own.
- Motivated by [REL-SPEC-1]: THEME-A found 13 of 18 ports drifted off the
  current spec version, and implementers evidently find self-checking
  re-conformance hard today with the requirements scattered by topic
  rather than sequenced by build order. Placed as a trailing appendix
  (after Appendix A) rather than inserted early in the document, so no
  existing section number changes and no cross-reference anywhere in the
  ecosystem (docs, code comments, other repos' issues) breaks.
- `SpecVersion` unchanged (`2.0`); no existing conformant implementation
  is made non-conformant by this addition, since it adds no new
  requirement.

## v2.0.5 — 2026-08-20 (doc addition; no normative change to existing conformant implementations)

- **§15.7.5**: added guidance for protocol bindings whose underlying wire
  protocol has several independently fixed-shape request/response pairs
  and no shared generic envelope beneath the fields this section's own
  reference mapping already covers — the case RCP's own TC18 binding hits
  (13 heterogeneous endpoint/operation types: GPIO, SPI, I²C, UART, ADC,
  both PWM directions, LIN, CAN, ISELED, MDIO, both WakeUp operations,
  discovery). Closes #66.
- Deliberately does not mandate one answer: a cross-repo survey of all
  four independent TC18 implementations found real, working, divergent
  strategies. cpp-RCP, rust-RCP, and go-RCP independently (no
  coordination between them) kept `Adapt` at this section's own generic
  ACF-envelope layer, pushing endpoint-type-specific payload
  encoding/decoding out to the caller via each endpoint type's own module.
  c-RCP alone (`include/rcp/adapt.h`/`src/adapt.c`) built a richer,
  per-operation-opcode dispatcher inside `Adapt` itself, narrowing
  `Caller` scope to one endpoint-type family and exposing type-specific
  fields via a `"rcp.<endpoint-type>.<field>"` `Meta` convention. Both are
  now documented as conformant strategies, mirroring this section's own
  pre-existing permissive pattern for multi-stream `ID` encoding (same
  "document your choice in the `Adapt` doc comment" disclosure
  requirement, no single mandated format).
- `SpecVersion` unchanged (`2.0`); no existing conformant implementation
  (including all four current RCP bindings, using either strategy) is
  made non-conformant by this addition.

## v2.0.4 — 2026-07-30 (bug fix; no semantic change)

- **`go.mod`**: module path was still `github.com/SoundMatt/RELAY` after the
  v2.0 MAJOR bump. Per Go's semantic import versioning rules, a v2+ tagged
  module with a `go.mod` file must carry a `/v2` path suffix or it becomes
  uninstallable via ordinary `go install`/`go get` tooling
  (`go install .../RELAY/cmd/relay@v2.0.3` failed with `invalid version:
  module contains a go.mod file, so module path must match major version`).
  Renamed to `github.com/SoundMatt/RELAY/v2`; updated all 30 internal
  package imports, README install/usage snippets, and §13.4's module-path
  guidance accordingly. Closes #70 — independently rediscovered by three
  downstream repos (rust-RCP, rust-MQTT, go-mqtt), all of which had worked
  around it with a commit-hash pin instead of a normal version tag; they can
  now move to a standard `github.com/SoundMatt/RELAY/v2 v2.0.4` require.
- `SpecVersion` unchanged (`2.0`); this is a Go-tooling installability fix,
  not a wire-format or behavioral change.

## v2.0.3 — 2026-07-30 (bug fix + doc correction; no normative change)

- **`router/router.go`**: `Router.Run` leaked the goroutine(s) already
  started for earlier sources when a later source's `Subscribe` failed —
  it returned the error immediately without stopping them. Now derives a
  cancellable child context, cancels it and waits for all started
  goroutines to actually stop before returning the error. Added
  `TestRunSubscribeFailureStopsAlreadyStartedGoroutines`, confirmed to
  fail against the pre-fix code and pass against the fix.
- **README.md**: stale "Current: v1.13 (stable)" corrected to v2.0.
- **Dockerfile**: stale `org.opencontainers.image.version="1.13.0"` /
  `io.relay.spec-version="1.13"` OCI labels corrected to `2.0.0`/`2.0`.
- **§15.7.4**: added an explicit note that MQTT v5 properties
  (`PacketID`/`ResponseTopic`/`CorrelationData`/`UserProperties`/
  `ContentType`/`ExpiryInterval`) are intentionally not carried through
  `relay.Message`, matching the RCP §15.7.5 subset-mapping style — closes
  the doc gap NEW-R-02 flagged (the underlying field-dropping itself
  remains tracked separately as RELAY-13/#87).
- `SpecVersion` unchanged (`2.0`); this is a tooling/doc/bug-fix patch
  release.

## v2.0.2 — 2026-07-30 (doc correction; no normative change)

- **`rcp/rcp.go`**: the `ToMessage` doc comment claimed "ToMessage/FromMessage
  round-trip losslessly for any Message this package produces." False:
  `Control`'s `FlagAck`/`FlagResponse`/`FlagMoreSegments` bits and
  `Timestamp` were never carried (this was already true before v2.0.1;
  v2.0.1 fixed `TransactionNum`/`ReadSizeOrSegment` but kept the same
  overclaiming sentence). The *behavior* was already spec-conformant —
  §15.7.5 only ever mapped op/error/transaction_num/read_size_or_segment —
  the defect was the docstring contradicting both the code and the spec's
  deliberate subset mapping, dangerous in a functional-safety codebase
  where such invariants may be relied on. Softened the comment to state
  exactly which fields round-trip and which three Control bits don't.
- **`rcp/rcp_test.go`**: `TestMessageToMessageError` previously round-tripped
  `FlagResponse | FlagError` but only asserted `FlagError` survived,
  written in a way that didn't notice `FlagResponse` silently became
  `FlagRead`. Strengthened to assert the full `Control` value, so the
  documented gap stays visible instead of being accidentally hidden.
- `SpecVersion` unchanged (`2.0`); this is a doc-only patch release.

## v2.0.1 — 2026-07-30 (bug fix + doc correction; no normative change)

- **§15.7.5**: added the `Meta["rcp.transaction_num"]`/`Meta["rcp.read_size_or_segment"]`
  mapping rows and documented `ToMessage()`'s single, direction-agnostic
  behavior (it always sets both `rcp.op`/`rcp.error` together, and now also
  `rcp.transaction_num`/`rcp.read_size_or_segment`, regardless of request vs
  response direction). This closes a gap against §15.7's pre-existing
  "MUST be lossless for all mandatory fields" rule — `TransactionNum` and
  `ReadSizeOrSegment` are existing `rcp.Message` fields that were silently
  dropped by `ToMessage`/`FromMessage` and never round-tripped, though the
  general lossless mandate already covered them; no new field was added to
  `rcp.Message` and no existing conformant behavior is invalidated. Also
  documented that `Message.Timestamp` (the native AVTP presentation
  timestamp) is intentionally not carried into `relay.Message.Timestamp`,
  matching every other protocol in §15.7.
- **`rcp/rcp.go`**: `FromMessage`/`ToMessage` now carry `TransactionNum` and
  `ReadSizeOrSegment` through `Meta`; fixed the package doc comment's
  reference to a nonexistent "RELAY v1.15" (the TC18 replacement shipped in
  the v2.0 MAJOR release, per REQ-RELAY-094).
- **`spec/vectors/rcp-message.json`**: populated `transaction_num` and
  `read_size_or_segment` with non-zero values so `TestGoldenVectorsRoundTrip`
  actually exercises the previously-dropped fields instead of passing
  vacuously at their zero default.
- **README.md**: replaced the stale "RCP zone…" comment and the pre-v2.0
  symbolic `ID: "FrontLeft"` quickstart example (which `ParseEndpointID`
  rejects) with the current decimal `ByteBusID` encoding.
- `SpecVersion` unchanged (`2.0`); this is a tooling/doc patch release.

## v2.0 — 2026-07-29 (stable) — BREAKING CHANGE

Replaces §15.5/§15.7.5's RCP canonical types and conversion mapping
entirely. RCP now means the real OPEN Alliance TC18 Remote Control Protocol
Specification v0.5.1_RC (an IEEE 1722 AVTPDU/ACF wire protocol addressing
individually-configured Endpoints on a remote RC Server), not the earlier
RELAY-internal placeholder protocol (`Zone`/`Command`/`Response`/`Status`/
`Priority`/`CommandType`) those names described through v1.14. No
compatibility shim — this is a genuine MAJOR bump per §19.3's stability
guarantee, since canonical types are one of the sections that guarantee
covers.

**Why now:** go-RCP (one of four x-Net RCP implementations independently
replacing their protocol core against the same target spec) reached full
TC18 conformance cutover (v1.0.0) and explicitly disclosed that RELAY's own
RCP schemas/golden vector were stale as of v1.14 — correctly leaving the
RELAY-side update out of its own scope. rust-RCP also reached v1.0.0, but
was found to have a real gap (no RC Client/network controller was ever
scoped into its roadmap, only the RC Server side) and isn't a reliable
second reference yet (tracked: rust-RCP#87) — this rework is designed
against go-RCP alone.

**New canonical types** (§15.5): `StreamID` (AVTP stream_id), `ByteBusID`
(endpoint address, scoped to a stream), `TransactionNum`, `ControlFlags`
(Ack/Read/Write/Response/Error/MoreSegments bits), `Message` (a decoded
ACF_ABB/ACF_GBB request/response/acknowledge). Replaces `Command`,
`Response`, `Status`, `Zone`, `Priority`, `CommandType`, `ResponseStatus`.
`Loan` is unchanged in shape.

**New `ToMessage()`/`FromMessage()` mapping** (§15.7.5): `Message.ID` is a
decimal `ByteBusID` string (was the Zone's PascalCase name). New `Meta`
keys `rcp.op` (`"read"`/`"write"`) and `rcp.error` (`"true"`/`"false"`)
replace `rcp.priority`/`rcp.cmd_type`/`rcp.healthy`/`rcp.status`. RCP has no
server-initiated push in the new protocol — `Subscribe()` returns a
permanently-empty stream (this was already true in spirit, just now
explicit). An implementation adapting something that can receive from more
than one stream concurrently (e.g. an RC Server rather than a single-stream
client `Controller`) MUST extend `ID`'s encoding to disambiguate the stream
— no single multi-stream format is mandated, since the common case doesn't
need one.

**Also updated:** the `Adapt`-wrapped `Controller`/`LoaningController`
interfaces (§8.5), the RCP error-sentinel table (§5) collapses to a single
`ErrNotFound` (the old Zone-specific sentinels have no TC18 equivalent),
the CLI `send` flags (§11.2: `--byte-bus-id`/`--op`/`--payload`, was
`--zone`/`--type`/`--payload`), the C++ (§18.2) and Rust (§18.3) binding
type definitions, and the embedded golden vector (`rcp-status.json` →
`rcp-message.json`, and its JSON Schema). REQ-RELAY-040/041 rewritten,
REQ-RELAY-094 added.

## v1.14 — 2026-07-28 (stable)

§13.7.2 module-name registry expansion, prompted by observed naming drift
across the four independent go/cpp/rust/c-RCP implementations now replacing
their protocol core with the real OPEN Alliance TC18 Remote Control Protocol.

Comparing the four repos directly found the same underlying concepts named
differently across languages — e.g. the AVTPDU/ACF wire-framing layer as
`avtp` (go), `wire`/`legacy_wire` (cpp), `avtpdu`+`acf` (rust), `avtp`+`acf`
(c); the E2E/safe-points mechanism as `crcsafe` (go, incorrectly — `e2e` was
already the mandated name), `e2e` (cpp/rust, correct), `safept` (c,
incorrectly). Root cause: each implementation's replacement plan was
designed independently against the abstract spec text, with no reference
implementation to structurally anchor against and no registry entries for
these new concerns (§13.7.2 previously only covered the RCP *control-plane*
concerns of the old, now-superseded protocol generation). A comparison
sweep across go/cpp/rust-DDS's already-converged RTPS internals
(`cdr`/`spdp`/`sedp`/`reliable`/`persist`/`transport`/`guid`/`locator`/
`wildcard`) found no equivalent drift there — those names were never
mandated by the registry either, but stayed consistent because that
build-out was framed as porting go-DDS's actual file structure rather than
independently interpreting spec prose. Added registry entries for both:
the new RCP protocol-core concerns (`avtp`, `acf`, `lifecycle`, `regmap`,
`discovery`, `request`, `fragment`) and the DDS RTPS internals, formalizing
what DDS already had by convention and giving RCP's four implementations an
explicit target to converge on. REQ-RELAY-093.

## v1.13 — 2026-07-27 (stable)

Deep-audit fix pass: two live CLI conformance-gate bugs, several tooling
correctness fixes, and C++/Rust language-binding gaps.

**Bugs fixed:**
- **`relay interop`** with zero binary arguments silently reported PASS/exit 0
  instead of the spec-mandated exit 2 ("no candidates").
- **`relay conform`**: `--strict` placed after the binary path (e.g.
  `relay conform mybinary --strict`) was silently dropped instead of applied,
  because Go's flag parser stops at the first positional argument — this
  could downgrade a should-FAIL CI gate to a passing WARN. `relay conform`
  and `relay trace` now reject unexpected extra positional arguments instead
  of silently ignoring them.
- **`cmd/relay`'s `toolVersion`** had drifted behind `SpecVersion` again
  (the exact class of bug v1.11.1 fixed once already) — bumped, and a test
  now gates on the two staying in sync.
- **Golden error vectors** (`spec/vectors/errors/*.json`) were unreachable
  through RELAY's own `go:embed`-based `Vector`/`VectorNames` API (a bare
  `*.json` glob doesn't recurse into subdirectories) — meaning `relay interop`
  could never exercise any implementation's reject-path behavior, only the
  happy path. Fixed the embed and taught `relay interop`/`relay convert`'s
  golden-vector test to drive the reject-path comparison correctly.
- **`relay crossbar`**: a route fanning out to multiple destinations of
  different protocols with no explicit converter picked one converter (tuned
  to the first destination) and applied it to every destination, silently
  mislabeling the rest. Split into one route per destination internally.
- **`relay crossbar`**'s send sink spawned a fresh subprocess per message;
  spec §11.2's streaming JSON sink is explicitly "the egress dual of
  `subscribe --format json`" — a single persistent process reading a stream
  until EOF. Fixed to match, and `cliNode.Subscribe` now honors the caller's
  configured back-pressure policy instead of always blocking.
- **`router.Router.AddSpoke`/`AddRoute`** mutated shared state without
  holding the package's own mutex — a real data race for concurrent use.
- **SOME/IP**: malformed-ID errors from `FromMessage` now also wrap the
  spec-mandated `ErrMalformedMessage` sentinel (which wraps
  `relay.ErrPayloadTooLarge` per §5.4), in addition to the existing
  `ErrInvalidID` (kept for backwards compatibility).
- `relay convert`'s stdin read is now bounded (512 MiB) instead of unbounded.
- `relay trace`'s live-mode child process is now killed on Ctrl+C instead of
  orphaned.

**Spec content (additive, non-breaking):**
- §17: corrected the `relay conform` coverage claim — it is a black-box CLI
  tool and cannot verify source-level requirements (error sentinels, mock
  presence, frame constraints, etc.); the text now says precisely what is and
  isn't checked, instead of overclaiming.
- §18.2/§18.3: added the missing C++/Rust `HealthProvider`/`MetricsProvider`/
  `Drainer` bindings and `SubscriberOptions::event_id`/`topic_name` fields —
  present in Go (§14.1) but absent from both other language sections.
- §15.2: removed a duplicate, differently-cased `relay::dds::BackPressurePolicy`
  — the type is canonical for all protocols (§14) and lives once in `relay::`.
- `SpecVersion = "1.13"`.

## v1.12 — 2026-07-27 (stable)

Add `"c"` as a valid CLI `language` value.

- **§12.1**: `language` MUST now be one of `"go"`, `"cpp"`, `"rust"`, `"c"` (was
  missing plain C, forcing the one existing C99 implementation, c-RCP, to
  misreport itself as `"cpp"` just to pass `relay conform --strict`'s schema
  validation).
- `spec/schemas/cli-version.json`'s `language` enum, the §13.5 Docker
  `io.relay.language` label pattern, and `spec/version.json`'s `languages`
  array all updated to match.
- `SpecVersion = "1.12"`. Additive, non-breaking — no existing go/cpp/rust
  implementation is affected.

## v1.11.1 — 2026-07-27 (bug fix + doc correction; no normative change)

- **`cmd/relay/conform.go`**: fixed `runConform` so `--format json` no longer
  returns before the `sevFail` exit-code check; both `text` and `json` output
  now fall through to the same unconditional check, so `relay conform --format
  json` on a FAILing binary correctly exits 1 (previously exited 0, silently
  hiding FAIL results from CI/scripts consuming JSON output). Added a
  regression test covering `--format json` against a FAIL binary.
- **README.md**: corrected the stale "v0.1 (draft)" status line to the actual
  current spec version, added the missing `go install
  .../cmd/relay@latest` CLI install instructions, and expanded the CLI section
  from 2 to all 16 subcommands with one-line descriptions; removed the stale
  "(available from v0.5)" annotation on `relay conform`.
- **ROADMAP.md**: corrected the stale "✦ in progress" tags on the v0.1–v0.4
  milestones (all four shipped) to "✦ done", matching every later milestone.
- **spec/relay-spec.md Appendix A**: retitled from "Current project alignment"
  to "Project alignment as of v0.2 (2026-06-16); not maintained thereafter" and
  added a pointer to `relay conform`/`relay report --scan` for current status —
  the table was frozen at v0.2-era data and read as misleadingly current.
- `SpecVersion` unchanged (`1.11`); this is a tooling/doc patch release, not a
  spec content change.

## v1.11 — 2026-06-19 (stable)

Removed the C++ CLI conformance waiver.

- **§17.7 #7**: the waiver that let a C++ library with no CLI target have its CLI
  requirements assessed as "not applicable" is **removed**. Every conformant
  implementation MUST now provide the `version`/`capabilities`/`status` CLI — a
  C++ library exposes it via the `-DRELAY_BUILD_CLI=ON` build target.
- **Rationale:** every real C++ implementation already ships a CLI via a one-line
  build option (cpp-RCP, cpp-CAN, cpp-DDS), so the accommodation was obsolete; and
  it conflicted with the §20 continuous-conformance gates, which require
  `relay conform --strict` *against the built CLI*. A CLI-less implementation can't
  be machine-verified, so it can no longer claim conformance.
- **Impact:** no currently-conformant implementation relied on the waiver, so
  this affects only CLI-less scaffolds (e.g. cpp-MQTT). Tightening conformance —
  shipped as a MINOR because nothing currently passing is invalidated.
- `SpecVersion = "1.11"`.

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
