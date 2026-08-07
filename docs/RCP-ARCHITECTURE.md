# RCP Common Architecture — go-RCP / c-RCP / cpp-RCP / rust-RCP

## Status

This document defines the **canonical architecture, module lexicon, and
requirement-tagging schema** shared by all four sibling implementations of
the OPEN Alliance TC18 "Remote Control Specification for Ethernet" Remote
Control Protocol (RCP):
[go-RCP](https://github.com/SoundMatt/go-RCP),
[c-RCP](https://github.com/SoundMatt/c-RCP),
[cpp-RCP](https://github.com/SoundMatt/cpp-RCP),
[rust-RCP](https://github.com/SoundMatt/rust-RCP).

It is a **design target**, not a claim that all four repos already conform
to it. Each repo's own `ARCHITECTURE.md` stub records where that repo
currently stands against this document and links the tracking issues for
what's left. RELAY hosts the canonical copy because it is already the
umbrella spec/conventions repo each RCP implementation's CI checks
conformance against (`RELAY conform` / `relay conform` jobs).

Every choice below was derived by comparing the four repos' actual code,
not invented from scratch — each is attributed to the repo(s) whose
existing pattern it's drawn from.

## Module lexicon

One name per protocol concern, used consistently in this document and
(going forward) in each repo's own module/file naming where practical:

| Lexicon term | What it covers | TC18 reference |
|---|---|---|
| **wire / ACF layer** | `byte_message_info` encode/decode: `acf_msg_type`, `acf_msg_length`, `pad`, `mtv`, `byte_bus_id`, `evt`, `hs`, `cs`, `transaction_num`, `op`/`rsp`/`err`/`ms`, `read_size`/`segment_num` | §11.2.1 Table 4, §11.2.2.1 Table 6 |
| **framing / AVTP layer** | NTSCF/TSCF AVTPDU header encode/decode | §11.1 Figures 5-6 |
| **response classification** | turning a decoded response header into Acknowledge / Write-response / Read-response / Error-response | §11.3 Table 15, §11.3.1-§11.3.4 |
| **conditional-request layer** | compound, compound-wait, triggered, chained, timed, and the three cancellation forms | §11.2.2-§11.2.3 |
| **Table 30 / evt[2:0] write semantics** | request-side endpoint write-arithmetic selector, shared across every endpoint type | §13.5 Table 30 |
| **compound-wait comparison** | the evt[2:0]-selected byte_msg_payload-vs-current-status comparison a compound-wait request uses, identical across every endpoint type | §13.5.1 |
| **endpoint-type modules** | GPIO, SPI, PWM (in/out), ADC, I2C, LIN, CAN, UART, ISELED, MDIO, Wakeup | §12 (per-endpoint chapters) |
| **dispatch/routing** | incoming-request → endpoint-handler routing | (implementation concern, not directly a TC18 chapter) |

## Canonical choices

### 1. Response classification — one function, evt-first

**Reference implementation: cpp-RCP's `response_kind_of()`**
(`include/rcp/acf.hpp`).

```
classify_response(header) -> Acknowledge | Write | Read | Error
```

Per TC18 §11.3.1, `evt[3:0] == 0xF` identifies an Acknowledge
**unconditionally** — including when `err == 1` (a rejected Acknowledge is
still an Acknowledge, not an Error Response; Error Response is the
distinct `evt[3:0] < 0x9, err == 1` case per §11.3.4). The evt check must
therefore run **before** any err/op-based branching. cpp-RCP's
implementation does this correctly. c-RCP's `rcp_acf_classify_response()`
did not, until it was fixed (c-RCP#151) — the bug was found by re-deriving
this exact requirement against the spec text.

Current status per repo:

| Repo | Status |
|---|---|
| cpp-RCP | conformant (reference implementation) |
| c-RCP | conformant (fixed, c-RCP#151) |
| go-RCP | conformant (`Message.ResponseKind`, go-RCP#156) |
| rust-RCP | conformant (`ByteMessageInfo::response_kind`, rust-RCP#138, merged) |

### 2. Table 30 / evt[2:0] write semantics — one centralized module

**Reference shape: c-RCP's `rcp_acf_evt_row2_is_plain()` (`acf.h`/`acf.c`), go-RCP's `acf/evt.go`, rust-RCP's `evtgroup.rs`, cpp-RCP's `endpoint::WriteSemantics`.**

TC18 §13.5 Table 30 states its endpoint-type-grouped write-combining rules
once; every endpoint-type module should call into one shared
classifier/combiner rather than re-deriving Table 30 per endpoint. go-RCP
learned this the hard way (through v7.0.0, `evt[2:0]` was carried
correctly on the wire but no endpoint package interpreted it, and
`gpio`/`spi` had each invented their own in-band selector byte instead —
fixed in go-RCP's v8.0.0).

Current status per repo:

| Repo | Status |
|---|---|
| c-RCP | **conformant (reference implementation, fixed this pass)** |
| go-RCP | conformant (`acf/evt.go`) |
| rust-RCP | conformant (`evtgroup.rs`) |
| cpp-RCP | conformant (`endpoint::WriteSemantics` in `include/rcp/endpoint.hpp`, correctly called into by e.g. `gpio.hpp`'s `apply_gpio_write`) |

**c-RCP finding, resolved.** Table 30's endpoint-type row
`{ADC, PWM_IN, I2C, LIN, CAN, UART, ISELED, MDIO}` allows `evt[2:0] =
000b` for a plain request; every other value is either reserved
(`UNSUPPORTED_CMD`) or `111b`'s config-write shape (§12.7.1). c-RCP now
centralizes this in one shared predicate,
`rcp_acf_evt_row2_is_plain(evt)`, called from every endpoint type in the
row (c-RCP#153, c-RCP#154). Two modules had invented their own
non-conformant `evt[2:0]` schemes instead of using it, both now fixed:

- **CAN** (c-RCP#155, v0.109.0): `ep_can.h` had packed `FrameFormat` into
  `evt[2:0]` as a 6-value selector — an invented design modeled
  incorrectly on SPI's genuinely-different Table 30 row (SPI is its own
  dedicated row, `000b-101b` = real channel select; CAN belongs to the
  shared Row-2 rule, where those values must be *rejected*, not
  interpreted). TC18 Figure 39 places `FrameFormat` in the payload's
  leading quadlet instead, pixel-verified against the rendered
  specification page. Every CAN frame this module ever produced before
  the fix was wire-incompatible with a real TC18 peer.
- **LIN** (c-RCP#158, v0.112.0): `ep_lin.h` had invented an 8-value
  `evt[2:0]` "comparison-mode enumeration"
  (EXACT/PREFIX/ANY/NEVER+4-reserved), self-admittedly ("this module's
  own original design... rather than on any spec-derived enumeration").
  LIN sits in the same Row-2 group as every other endpoint here, with no
  exception in Table 30 — its own §13.7.10.1 prose ("a match under the
  conditions given by `evt[2:0]`") describes the *same* universal
  §13.5.1 vocabulary canonical choice #3 (below) covers, constrained to
  mode `000b` (exact match) by Table 30, not a LIN-private scheme.

### 3. Compound-wait comparison (§13.5.1) — one shared, endpoint-agnostic primitive

**Reference implementation: c-RCP's `rcp_acf_compound_wait_evt_valid()` /
`rcp_acf_compound_wait_match()` (`acf.h`/`acf.c`), c-RCP#156.**

TC18 §13.5.1 gives a compound-wait request's `evt[2:0]` an entirely
different meaning than Table 30 gives it for a Standard request: it
selects one of eight ways to compare that request's own
`byte_msg_payload` against the addressed endpoint's current status —
exact match, AND-with-1s-mask, AND-with-0s-mask, reserved, and four
leading-quadlet high/low-word `>=`/`<=` comparisons — **identically
across every endpoint type**, including the length-capping rule ("only
the first four out of 20 received bytes will be checked when
byte_msg_payload has only four bytes"). This is a completely separate
mechanism from canonical choice #2 above (which governs `evt[2:0]` for
*Standard* requests); confusing the two, or reimplementing this
comparison per endpoint type, is the exact anti-pattern #2 already
documents for Table 30.

Before c-RCP#156/#157, nothing in any of the four repos implemented this
rule at all. c-RCP had two *partial*, endpoint-specific, never-wired
helpers instead — `rcp_ep_spi_compound_wait_status_equal()` (only the
000b exact-match mode, plus a hardcoded, non-conformant fixed 4-byte
comparison length that isn't a real SPI-specific rule) and
`rcp_ep_pwm_in_compound_wait_compare()` (only the four `>=`/`<=` modes,
as a typed period/duty specialization) — neither reachable from real
request decode, since `rcp_compound_decode_request()` silently discarded
the ACF header's own `evt` field entirely.

**What c-RCP now does, as the reference shape:**

1. One generic primitive in the wire/ACF layer (`acf.h`/`acf.c`), not in
   any endpoint module — the comparison is a property of the mechanism,
   not of any one endpoint type.
2. `evt` threaded end-to-end: `rcp_compound_encode_request()` takes it as
   a real parameter (previously every compound-wait request silently
   encoded `evt = 0`); `rcp_compound_decode_request()` surfaces it.
3. Real per-request dispatch: each pending compound-wait request stores
   its own decoded `evt` and an owned copy of its `byte_msg_payload`
   (`rcp_server_pending_t.compound_wait_evt`/`compound_wait_target`),
   evaluated independently against a caller-supplied, endpoint-scoped
   `current_status` at every tick. A single flat "the wait condition"
   bool (c-RCP's own pre-fix `wait_condition_met` design) cannot
   represent two simultaneously-pending compound-wait requests on the
   same endpoint with different targets — this was a real, second defect
   found while wiring the primitive up, not just an API smell.
4. Reserved `evt[2:0] = 011b` rejected at admission time
   (`UNSUPPORTED_CMD`), per TC18's own rule, rather than stored as a
   request that could simply never match.
5. Any endpoint-specific comparison need (e.g. LIN's own exact-match
   requirement, canonical choice #2's LIN finding above) delegates to
   this same primitive rather than reimplementing comparison logic —
   `rcp_ep_lin_response_matches()` is a thin wrapper over
   `rcp_acf_compound_wait_match(0, ...)`.

Current status per repo:

| Repo | Status |
|---|---|
| c-RCP | **conformant (reference implementation, c-RCP#156/#157)** |
| go-RCP | not yet implemented |
| cpp-RCP | not yet implemented |
| rust-RCP | not yet implemented |

Porting this to the other three repos is the next item of work: each
needs (a) the generic 8-mode primitive in its own wire/ACF layer, (b)
`evt` threaded through that repo's own compound-wait encode/decode
surface, and (c) real per-request dispatch wiring in whatever module
owns that repo's request store/scheduler — adapted to each repo's own
existing conditional-request-layer shape (canonical choice #4 below),
not a mechanical file-for-file port.

### 4. Conditional-request layer — one unified module

**Reference shape: cpp-RCP's `request.hpp`, rust-RCP's `request.rs`.**

Both unify compound/compound-wait/triggered/chained/timed/cancellation
around one request-kind enum plus shared sequencer-bank, priority-tier,
and ledger state. c-RCP's five-way file split (`request_compound.c`,
request_chained.c`, `request_triggered.c`, `request_timed.c`,
`request_cancel.c`) duplicates that shared state five times; go-RCP's
`request/` package is a partial, inconsistent split (`chained.go`
standalone, a generic `kind.go`, separate `dispatcher.go`). Unifying
go-RCP and c-RCP onto the single-module shape is the largest structural
change in this effort — sequenced last, after the smaller items above are
stable, since it touches the most code per repo.

**go-RCP wire-format finding (2026-08-02, not yet fixed):** beyond the
module-shape gap above, go-RCP's `request/envelope.go` Compound,
CompoundWait, Triggered, and Timed envelopes independently invent their
own byte layout rather than TC18's real one — a correctness defect, not
just an organizational one, separate from (and larger than) canonical
choice #3's compound-wait-comparison gap above. Compound/CompoundWait use
a generic value-comparison `Conditional`
(`SequencerID`+`CompareOp`+`Operand`+`AdvanceOnMatch`) where TC18
§11.2.2.1/§11.2.2.2 define a sequencer *state machine* instead
(`cmp_start_state`/`cmp_next_state`/`cmp_sequencer`/`cmp_exec_delay`/
`cmp_repetitions` — no value comparison at all); Triggered
(`EncodeTriggered`/`DecodeTriggered`) encodes only a 1-byte source,
missing 4 of TC18 §11.2.2.3 Table 8's 5 real fields
(`trigger_signal_nr`/`trigger_threshold`/`trigger_exec_delay`/
`trigger_repetitions`); Timed (`EncodeTimed`/`DecodeTimed`) uses an
8-byte caller-defined microsecond clock where TC18 §11.2.2.5 defines a
48-bit gPTP-domain `presentation_time` (6 bytes, 3.25-day rollover).
rust-RCP's `src/request.rs` (real `start_state`-equality sequencer
gating) and c-RCP's `request_compound.c`/`request_timed.c` are the
correct reference shapes for the redesign. This is a multi-file semantic
rewrite (sequencer-evaluation logic, not just wire bytes), not a quick
patch. Tracked as go-RCP's largest open conformance item.

**rust-RCP related finding — CONFIRMED (2026-08-02 flagged, 2026-08-07
confirmed), not yet fixed:** rust-RCP's Timed-request readiness check
(`REQ-TIME-002`/`REQ-TIME-003`) uses `AvtpTimestamp` — a 32-bit type
rolling over every ~4.3 seconds — for a Timed request's presentation-time
gate. This is no longer just flagged: the code's own doc comment
("Provenance note: `TimedExecutionTime`'s wire placement, width...") is
explicit that this was an unconfirmed judgment call, not a verified wire
fact — "nothing in this crate's roadmap states that a Timed request's
execution-time field is actually 32 bits wide." TC18's real
`presentation_time` (§11.2.2.5, TC18.txt L1596-1649) is 48-bit,
gPTP-domain, ~3.25-day rollover — a materially different value from
`AvtpTimestamp`'s domain. Unlike the go-RCP finding above, this is a
defect on an *existing, traced, tested* requirement, not an absence, so
it is not staged in a `.fusa-reqs-pending.json` entry — it needs a direct
fix (new type or width correction) in a future session.

**cpp-RCP and rust-RCP field-level gaps beneath their correct module
shape (found 2026-08-07):** both repos are correctly cited above as the
*reference shape* for module unification — that assessment stands. But a
closer, field-by-field read against TC18 §11.2.2 found both have real
gaps underneath that good architecture, now staged in each repo's
`.fusa-reqs-pending.json` (see the schema-6 pending-reqs mechanism
above):

- **cpp-RCP**: the opcode/routing/state-machine layer
  (`RequestTypeOpcode`'s 8 real TC18 opcodes including 0x06
  ClearNonSafestate, `mtv=false` repurposing, `expected_start_state`) is
  correctly designed — not the same class of defect as go-RCP's
  conditional-envelope layer. But three field-level behaviors are
  genuinely absent (the real TC18 field names appear only in comments,
  never as real fields/variables/consumed values anywhere in `include/`
  or `src/`): Compound's `cmp_exec_delay`/`cmp_repetitions`, Triggered's
  `trigger_source_ep`/`trigger_signal_nr`/`trigger_threshold`, and
  Timed's `presentation_time` (no scheduling/dispatch logic exists
  anywhere — a Timed request today has no distinguishing runtime
  behavior from a standard one).
- **rust-RCP**: notably more complete than cpp-RCP — `cmp_exec_delay`/
  `cmpw_exec_delay`, `trigger_exec_delay`, and `trigger_repetitions`
  (`TriggerRepeatCount`/`is_trigger_repeat_exhausted`) are all real,
  well-tested implementations already. Two gaps remain: Compound/
  CompoundWait's `cmp_repetitions`/`cmpw_repetitions` (repeat count) is
  absent entirely, and Triggered's `trigger_source_ep`/
  `trigger_signal_nr`/`trigger_threshold` are absent
  (`should_count_trigger_occurrence(endpoint_busy)` always returns true
  regardless of which signal or endpoint actually fired).

### 5. Requirement-tag placement — per-function

**Reference convention: c-RCP's per-function `//cfusa:req REQ-ID`**,
placed directly above the function/type it covers.

go-RCP's, cpp-RCP's, and rust-RCP's file-level convention (every tag for
a file collected at the top, e.g. `doc.go`) is how go-RCP's Table 27
mistake happened: a test file's tags pointed at an unrelated requirement,
and nothing about reading either file in isolation made that visible — a
per-function tag directly above the thing it covers is self-checking in a
way a file-level block isn't.

**Comment syntax is not a repo choice.** Each language's x-FuSa tool
parses a fixed comment prefix — go-RCP/rust-RCP/c-RCP-equivalent all
require no space (`//fusa:req`, `//cfusa:req`); cpp-RCP's `cpfusa` (as of
v0.18.0) is confirmed to parse `// fusa:req` **with** a space. Whether
cpfusa would also accept the no-space form is not yet confirmed. This
document does not mandate a syntax change — that would mean filing an
issue against the tool repo (`cpp-FuSa`), not editing it directly, per
standing policy against editing other repos.

### 6. `.fusa-reqs.json` schema — c-RCP's field set, plus a stable master-catalog id

Canonical fields, all repos:

```
id, title, text, standard, level, asil, scope, status, tc18, tc18_master_id
```

- `id`/`title`/`text`/`standard`/`level`/`asil` — already common to all four.
- `scope`, `status` — c-RCP's convention (`status`: `implemented` /
  `partial` / `not-implemented`). **Correction (2026-08-07): this is NOT a
  working CI-gate exemption mechanism, contrary to what this document
  previously claimed.** Directly verified: c-RCP has 148 requirements
  labeled `partial`/`not-implemented` in `.fusa-reqs.json`, yet a freshly
  built, CI-pinned `cfusa` (v0.5.50) reports c-RCP genuinely at 1023/1023
  (100%) traced and tested — `cfusa`'s `--req-coverage 100`/`--sec-tested
  100` gates do not read the `status` field at all. Those 148 labels are
  stale/decorative: the requirements were actually finished at some point
  after the label was written, and nothing ever updated it. The same
  empirical test (a fake untraced entry added to a scratch copy of
  go-RCP's `.fusa-reqs.json`) confirms `gofusa`'s CI check hard-fails on
  any untraced/untested requirement with no exemption path at all. **No
  tool in this ecosystem has a real status-based exemption mechanism.**
  The `status` field may still be worth keeping as human-readable
  documentation, but must not be relied on or assumed to affect tool
  behavior.
- `tc18` — c-RCP's existing citation-string convention:
  `"§X.Y Table N, TC18.txt L1234-1256"`.
- `tc18_master_id` — **new**: the id of the corresponding entry in the
  640-entry master TC18 requirement catalog built for this propagation
  effort (`TC18-11.3-006` etc.), so cross-repo reconciliation becomes an
  exact-match lookup instead of the line-range-overlap heuristic
  `cross_reference.py` currently has to use.

All four repos' x-FuSa tools (`gofusa`, `cfusa`, `cpfusa`, `rsfusa`)
already tolerate unrecognized JSON fields without error — confirmed for
`gofusa`/`cfusa` by direct testing; assumed but not yet spot-checked for
`cpfusa`/`rsfusa`, to be confirmed before this schema is relied upon in
those two repos.

**The real mechanism for a genuinely-missing (no requirement AND no
implementation) TC18 clause, established 2026-08-07:** a `.fusa-reqs-
pending.json` file at each repo's root — same field schema as that
repo's own `.fusa-reqs.json`, plus a `gap` field explaining what's
missing and the grep/read evidence behind that conclusion, and a
`_readme` field explaining the convention inline. Confirmed, per repo,
via a freshly built pinned tool, to be completely invisible to
`gofusa`/`cfusa`/`cpfusa`/`rsfusa` (never read, never affects
`check`/`trace` output), so it carries zero CI risk while still being a
real, structured, migratable record. An entry moves into the real
`.fusa-reqs.json` only once it has a genuine implementation and test.
Already in place: cpp-RCP (conditional-request field gaps), rust-RCP
(conditional-request field gaps), go-RCP (PWM_OUT Table 42 trigger
signals) — see each repo's own `.fusa-reqs-pending.json` and canonical
choice #4 below for the conditional-request findings specifically.

### 7. Conditional-request requirement-id grouping — c-RCP's split

Canonical id-prefix convention, independent of whether the underlying
code is one unified module (per §4, above) or several files:

```
REQ-CMP-*      compound / compound-wait
REQ-TRIG-*     triggered
REQ-CHAIN-*    chained
REQ-TIMED-*    timed
REQ-CANCEL-*   clear-all / clear-non-safestate / clear-single
```

go-RCP currently uses one bucket (`REQ-REQ-*`) for all of the above; that
will need re-splitting when go-RCP's conditional-request module is
unified (§4).

## Verification standard for any change made against this document

- Full native build + full test suite for the repo being changed.
- The real, CI-pinned x-FuSa binary for that repo (not a different
  version) run locally: `check` (0 errors) and `trace` (100%
  traced+tested).
- Any behavior-changing fix is mutation-tested (revert it, confirm the
  now-missing test coverage actually fails) before being trusted.
- CI green on the real PR before merge.

## Provenance

Built from a hands-on, read-only architecture inventory of all four
repos' actual source (not from documentation, which may be stale) during
the TC18 requirement-catalog propagation effort. See each repo's
`ARCHITECTURE.md` stub for repo-specific file-path mappings and
conformance-tracking status.
