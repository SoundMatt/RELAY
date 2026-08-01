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
| go-RCP | **missing** — task tracked, no classifier exists at all |
| rust-RCP | **missing** — task tracked, no classifier exists at all |

### 2. Table 30 / evt[2:0] write semantics — one centralized module

**Reference shape: go-RCP's `acf/evt.go`, rust-RCP's `evtgroup.rs`, cpp-RCP's `endpoint::WriteSemantics`.**

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
| go-RCP | conformant (`acf/evt.go`) |
| rust-RCP | conformant (`evtgroup.rs`) |
| cpp-RCP | conformant (`endpoint::WriteSemantics` in `include/rcp/endpoint.hpp`, correctly called into by e.g. `gpio.hpp`'s `apply_gpio_write`) |
| c-RCP | **not centralized** — `ep_can.h`/`ep_lin.h`/`ep_pwm.h`/`ep_spi.h`/`ep_gpio.h` each reference Table 30 independently; task tracked |

### 3. Conditional-request layer — one unified module

**Reference shape: cpp-RCP's `request.hpp`, rust-RCP's `request.rs`.**

Both unify compound/compound-wait/triggered/chained/timed/cancellation
around one request-kind enum plus shared sequencer-bank, priority-tier,
and ledger state. c-RCP's five-way file split (`request_compound.c`,
`request_chained.c`, `request_triggered.c`, `request_timed.c`,
`request_cancel.c`) duplicates that shared state five times; go-RCP's
`request/` package is a partial, inconsistent split (`chained.go`
standalone, a generic `kind.go`, separate `dispatcher.go`). Unifying
go-RCP and c-RCP onto the single-module shape is the largest structural
change in this effort — sequenced last, after the smaller items above are
stable, since it touches the most code per repo.

### 4. Requirement-tag placement — per-function

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

### 5. `.fusa-reqs.json` schema — c-RCP's field set, plus a stable master-catalog id

Canonical fields, all repos:

```
id, title, text, standard, level, asil, scope, status, tc18, tc18_master_id
```

- `id`/`title`/`text`/`standard`/`level`/`asil` — already common to all four.
- `scope`, `status` — c-RCP's convention (`status`: `implemented` /
  `partial` / `not-implemented`, the mechanism that lets a genuine TC18 gap
  be recorded honestly without breaking a 100%-traced CI gate).
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
those two repos. Whether `status`'s not-implemented exemption path
already works (or needs adding) for `gofusa`/`cpfusa` specifically is
also unconfirmed — go-RCP's CI gate today hard-fails on anything short of
100% traced+tested with no exemption mechanism found so far.

### 6. Conditional-request requirement-id grouping — c-RCP's split

Canonical id-prefix convention, independent of whether the underlying
code is one unified module (per §3, above) or several files:

```
REQ-CMP-*      compound / compound-wait
REQ-TRIG-*     triggered
REQ-CHAIN-*    chained
REQ-TIMED-*    timed
REQ-CANCEL-*   clear-all / clear-non-safestate / clear-single
```

go-RCP currently uses one bucket (`REQ-REQ-*`) for all of the above; that
will need re-splitting when go-RCP's conditional-request module is
unified (§3).

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
