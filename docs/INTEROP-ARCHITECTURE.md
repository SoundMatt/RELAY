# External-Reference Interop Architecture — THEME-K

## Status

Design document. Precedes implementation of [THEME-K](https://github.com/SoundMatt/RELAY/issues/125)
and its instances INTEROP-01 through INTEROP-07 (issues #147–152). Mirrors
`RCP-ARCHITECTURE.md`'s role: a coordinating document for a change that spans
multiple repositories, written and verified against real CI configuration
before any repo's workflow is touched — not derived from the filing audit's
own text alone.

Every claim below was independently re-verified by reading the actual
`.github/workflows/*.yml` in each repo as it stands, not inferred from the
audit register. Several of the audit's own claims turned out to be wrong or
incomplete; those are called out explicitly rather than silently corrected,
per this project's standing practice of not trusting an audit claim that
hasn't been re-checked against current HEAD.

## The problem, precisely

`relay interop` (RELAY's own CLI, §20.2) is a **self-consistency check**: it
diffs a port's `convert` output against RELAY's own embedded golden vectors.
That is valuable — it catches "this port disagrees with RELAY's own reference
conversion" — but it structurally cannot catch "RELAY and every port sharing
the same misreading of the underlying wire protocol," because every input to
the check traces back to RELAY's own `spec/vectors/*.json`. Genuine
interoperability requires a peer that was **not** built against RELAY's
vectors: a real broker, a real kernel network device plus a third-party CLI,
or a vector set transcribed independently from the protocol's own external
standard.

## Current-state inventory (re-verified, not assumed)

| Family | External reference peer? | Probe-then-skip? | Exit-code discipline | Notes |
|---|---|---|---|---|
| RCP (go/c/cpp/rust) | No — RELAY-internal only; **c-RCP and cpp-RCP run no `relay interop` at all** | N/A | rust-RCP swallows the real exit code (see below) | Audit's "zero external interop" claim confirmed |
| LIN (go/cpp/rust) | No — RELAY-internal only | N/A | rust-LIN swallows the real exit code (see below) | Audit's "zero external interop" claim confirmed |
| MQTT (go/rust) | **Yes** — real `eclipse-mosquitto:2` service container + `mosquitto_pub`/`mosquitto_sub` | **Yes** — audit's claim of "no probe-then-skip" is **wrong**; both repos gate the live-broker job behind a probe step | Clean (real exit codes) | Strongest *functional* pattern (real, always-available broker), but not probe-free as claimed |
| DDS (go/cpp/rust) | Yes in principle, against CycloneDDS | Yes — **and for go-DDS/rust-DDS the probed Docker tag (`eclipse-cyclonedds/cyclonedds:latest`) was never published**, so the job plausibly never runs at all, not merely "sometimes skips" | Clean | cpp-DDS already fixed this by building the peer image locally from upstream source instead of pulling a nonexistent tag |
| CAN (go/cpp/rust) | Yes — real `vcan` kernel device + `can-utils` | Yes, but **honestly documented**: repos state in-line that GitHub-hosted runners lack the `vcan` kernel module and the job "routinely skips" | Clean | Most transparent skip messaging of any family; rust-CAN also correctly separates its loopback test (`can_two_process_interop`) from its real external test (`can_thirdparty_interop`) — a naming pattern worth generalizing |
| SOME/IP (go-SOMEIP) | No — no vsomeip or any external SOME/IP stack referenced anywhere in the repo | N/A | Clean | Audit confirmed |

**Mislabeling / exit-code violations found, by repo:**

- `rust-RCP` (`relay-interop` job) and `rust-LIN` (`relay-interop` job) both
  run `output=$(relay interop ... 2>&1) || true` and then `grep` the captured
  stdout for `EQUIVALENT`/`SKIP`/`ERROR` instead of checking the command's
  real exit code. This silently tolerates both a genuinely broken run and any
  future change to `relay interop`'s output formatting (§20.1 item 3 already
  requires gating on the real result).
- No repo currently mislabels a same-process loopback test as "interop"
  except by omission of a clearer name; rust-DDS's `rtps-interop`/
  `shmem-interop` jobs are explicitly commented as distinct from external
  interop already and don't need renaming. rust-CAN's
  `can_two_process_interop` vs. `can_thirdparty_interop` split is the
  ecosystem's best existing example of the naming discipline INTEROP-07
  asks for — use it as the template, not an example needing a fix.
- No repo has an interop/conformance vector set independently derived from
  the TC18 PDF or the LIN (ISO 17987) standard text. Every "golden vector"
  check in every RCP/LIN repo traces back to `RELAY/spec/vectors/*.json`.

## Canonical conventions (apply ecosystem-wide, mechanical fixes first)

These four conventions resolve INTEROP-04, INTEROP-05, INTEROP-06, and
INTEROP-07 without requiring any new vector set. Each is independently
shippable as a small, low-risk PR per affected repo.

### 1. Naming discipline (INTEROP-07 + the mislabeling half of INTEROP-04)

- **`*-interop`** is reserved for a job/step that talks to a peer this
  repository did not build: a real third-party binary, daemon, or CLI
  (`mosquitto_pub`, `can-utils`'s `cangen`/`candump`, a real CycloneDDS
  build, a future TC18/LIN reference peer).
- **`*-conformance`** (not "interop") is the correct label for any step
  that only runs RELAY's own `relay interop`/`relay conform` CLI — that is
  self-consistency against RELAY's embedded vectors, not third-party
  interoperability. Rename existing `relay-interop` job/step names to
  `relay-conformance` across every repo that currently calls them
  "interop" while only invoking RELAY's own CLI.
- **`*-loopback`** or **`*-multiprocess`** is the correct label for a test
  that runs two instances of this repo's own port against each other.
  rust-CAN's `can_two_process_interop` naming is the reference example —
  cite it directly when this convention lands in other repos.

### 2. Exit-code discipline (the `|| true` half of INTEROP-04)

`relay interop`/`relay conform` already return a real, meaningful exit code
(§11.3). No CI step may wrap that invocation in `|| true` and substitute a
`grep` of the captured stdout. Fix, concretely:

- **rust-RCP** `relay-interop` job: replace
  `output=$(relay interop ... 2>&1) || true` + `grep` with a direct
  invocation whose exit code gates the step (matching go-RCP's existing
  clean pattern in the same family, which needs no change).
- **rust-LIN** `relay-interop` job: identical fix, matching go-LIN's and
  cpp-LIN's already-clean pattern.

This is a pure regression-risk-free mechanical fix — the underlying check is
unchanged, only whether CI actually observes its result.

### 3. Probe-then-skip hardening (INTEROP-05) — with a prerequisite

A probe that can *genuinely* never succeed on the ecosystem's runners (the
`vcan` kernel module is documented as absent on GitHub-hosted runners; no
public `eclipse-cyclonedds/cyclonedds` image exists) is a different problem
from a probe that is simply *unreached* due to a bug. **Fix the bug before
hardening the gate**, or hardening will just turn a silent no-op into a
permanently red required check:

- **go-DDS, rust-DDS, cpp-DDS**: the `docker pull
  eclipse-cyclonedds/cyclonedds:latest` step targets a tag that was never
  published — this is a real defect, not environmental unavailability. Fix
  by building the peer image locally from upstream CycloneDDS source
  (`docker compose ... build cyclone-peer`) instead of pulling a nonexistent
  tag. Only once the probe can genuinely succeed does hardening it make
  sense.
  <br>*(Correction, 2026-08-21: an earlier draft of this document framed
  cpp-DDS as having "already fixed" this bug, implying a pre-existing,
  proven template the other two repos could copy. A follow-up
  CI-history investigation found that's not accurate — cpp-DDS carried the
  identical nonexistent-tag bug through at least 7 of its own preceding CI
  runs and was fixed in its own most recent commit at the time, essentially
  concurrently with go-DDS and rust-DDS, not earlier. All three DDS repos
  reached "bug fixed" maturity at the same time, not two-fixed-one-pending.
  The underlying `Dockerfile.cyclonedds` pattern itself is still the right
  fix and was ported correctly; only the historical framing was wrong.)*
- **Once a probe's target is real** (CAN's `vcan`+`can-utils`, DDS's
  locally-built peer, MQTT's service container), add a hard-failing check on
  the canonical Linux runner specifically: if that probe still reports
  unavailable there, fail the job instead of emitting `::notice::`/
  `::warning::` and silently skipping. Other runners (e.g. a hosted runner
  genuinely missing `vcan`) may still legitimately skip — the goal is that
  *the* runner the ecosystem treats as canonical for gating a merge can no
  longer report green without having executed the check.
- CAN's existing skip messaging (loud `::warning::` plus a
  `$GITHUB_STEP_SUMMARY` banner, explicit in-line comment admitting the job
  "routinely skips") is the right transparency floor even before hardening
  — repos still using a quiet `::notice::` (go-mqtt, rust-MQTT, go-DDS,
  rust-DDS) should upgrade to CAN's messaging as an interim step.

**Hardening readiness, per family (investigated 2026-08-21 against real CI
history, not assumed from this document):**

| Family | Probe target now real? | Evidence | Verdict |
|---|---|---|---|
| CAN (go/cpp/rust) | No — genuinely unavailable | 24/24 checked runs across all three repos fail identically: `modprobe: FATAL: Module vcan not found in directory /lib/modules/<kernel>`. GitHub-hosted `ubuntu-latest`/`ubuntu-22.04` runners do not ship this kernel module. | **Do not harden.** Hardening today would make CI permanently red on every PR, not occasionally red on a genuine regression. This is the one family where the audit's original caution is fully justified. |
| DDS (go/cpp/rust) | Yes, as of the fixes above | Each repo: 0/7 pre-fix runs succeeded (confirms the bug was real and universal); exactly 1/1 post-fix run succeeded so far, each landing 2026-07-31 to 2026-08-21. | **Not yet — wait for more green runs.** A source build (git clone + CMake) is more failure-prone than a pulled tag ever was; one success isn't enough to certify it reliable on the canonical runner. Revisit after several more runs accumulate. |
| MQTT (go/rust) | Yes — always was (real `eclipse-mosquitto:2` service container) | 6/6 available runs succeed for both repos; real interop tests genuinely execute and pass every time (`mosquitto_pub`/`mosquitto_sub` and a third-party-CLI-backed round trip). | **Safe to harden now.** Strongest evidence of any family — proceed with the canonical-runner hard-fail. |

### 4. Service-container-first, where a real image exists (INTEROP-06)

MQTT's pattern —

```yaml
services:
  mosquitto:
    image: eclipse-mosquitto:2
    ports:
      - 1883:1883
```

— is the standard **wherever a suitable public image genuinely exists**,
because it guarantees the peer is present on every run rather than depending
on a runtime probe at all. This is not a drop-in replacement for every
family, though: CAN's peer is a kernel device (`vcan`), not a containerized
daemon, so a `services:` block doesn't apply; DDS has no official published
CycloneDDS image, so cpp-DDS's local-build-then-probe is the correct
adaptation of the same underlying principle (guarantee-first, not
runtime-luck-first) rather than a literal copy of MQTT's YAML. Apply the
*principle* (an unconditionally-present peer beats a probed one) family by
family, not the literal `services:` syntax where it doesn't fit.

## The RCP and LIN vector-harness gap (INTEROP-01, INTEROP-02) — scoped, not built here

This is deliberately **not** designed to completion in this document. Two
reasons:

1. Authoring genuinely independent TC18 or LIN-standard (ISO 17987) golden
   vectors means reading the external specification text directly and
   hand-deriving wire-level byte sequences, cross-checked against nothing
   RELAY or any port already produces. That is a careful, slow, error-prone
   exercise on its own — the opposite of something to rush inside a
   coordinating architecture doc.
2. Unlike the mechanical conventions above, a wrong vector here doesn't just
   miss a bug — it actively teaches every port the same wrong behavior a
   second time, this time under the banner of "independently verified."
   Getting the derivation process right matters more than getting it done
   fast.

What this document does fix now: the **shape** of the harness, so that when
vector authoring happens (as its own, separately-scoped follow-on effort,
matching the phased "batch per protocol" pattern already used for this
session's `tc18_master_id` propagation work), every repo wires it in the
same way.

- **Vector location**: a new `spec/vectors/external/tc18/` (and, for LIN, a
  parallel `spec/vectors/external/lin-iso17987/`) directory in RELAY,
  distinct from `spec/vectors/*.json` (which remains RELAY's own reference
  vectors, governed by §15.8/Requirement 16 and unaffected by this work).
  Each external vector file records, at minimum, the TC18/ISO-17987
  section and page/table reference it was transcribed from — the citation
  *is* the independence proof; a vector with no citation back to the
  external spec text is indistinguishable from one derived off an
  implementation and provides none of the value this initiative exists to
  add.
- **Harness shape**: a new `relay interop --external <vectors-dir>` mode (or
  a sibling command if `interop`'s existing hub-spoke design — §20.1 item 3
  — doesn't fit an externally-sourced fixture set cleanly; needs its own
  design pass once vector authoring starts) that feeds a port's `convert`
  output through the same equivalence check `relay interop` already uses,
  but against `spec/vectors/external/` instead of `spec/vectors/`. This
  reuses §20.1's existing machinery rather than inventing a second
  conformance pipeline.
- **CI wiring**: once the vector set and harness command exist, each RCP/LIN
  repo adds one job in the same shape as CAN's `can-interop` — a real,
  named `tc18-interop`/`lin-iso17987-interop` job, required (no `|| true`,
  no probe — the vector set is a checked-in file, not an environmental
  dependency, so there is nothing to probe).
- **Sequencing**: RCP before LIN — RCP is the ecosystem's flagship protocol
  per THEME-K's own framing and already has the deeper cross-repo tooling
  investment (`docs/RCP-ARCHITECTURE.md`, the `tc18_master_id` propagation
  work) to build on.

## What to ship first (recommended order)

Ordered by risk and independence — each is a standalone PR, no step depends
on a later one:

1. **INTEROP-04 exit-code fix** (rust-RCP, rust-LIN) — two small, safe PRs.
2. **INTEROP-07 / naming pass** — rename `relay-interop` jobs that only call
   RELAY's own CLI to `relay-conformance`, ecosystem-wide.
3. **DDS Docker-image bug fix** (go-DDS, rust-DDS) — adopt cpp-DDS's
   local-build pattern; this is a correctness fix independent of THEME-K's
   broader framing and worth shipping even before the hardening step.
4. **INTEROP-05 hardening** — once (3) lands, add the canonical-runner
   hard-fail for CAN, DDS, and MQTT's probes.
5. **INTEROP-06** — audit any family still on quiet `::notice::` skip
   messaging and upgrade to CAN's transparent skip banner as an interim
   step even before (4) lands everywhere.
6. **INTEROP-01/02 vector harness** — the large, separately-scoped,
   multi-session effort described above. Not started by this document;
   this document exists so that when it starts, it starts with an agreed
   shape instead of four/three independent, incompatible designs.

## Verification standard for any change made against this document

Same standard as `RCP-ARCHITECTURE.md`: every claim re-checked against the
repo's actual current CI configuration before a PR is opened, not assumed
from this document or the originating audit. Any CI change lands with the
job actually observed to run (not just parse) in a real Actions run before
merge — a probe-hardening change in particular must be confirmed to fail
loudly when the peer is genuinely absent and pass cleanly when it's present,
not merely reviewed by reading the YAML.

## Provenance

Written 2026-08-21, following the THEME-K umbrella issue (#125) and its six
instances (INTEROP-01 through INTEROP-07, issues #147–152) filed from the
2026-07-29 ecosystem audit register. Every finding in the "Current-state
inventory" section above was independently re-verified by reading the real
`.github/workflows/*.yml` in each named repo, not taken from the audit
register's own text — this surfaced one incorrect audit claim (MQTT's
probe-then-skip) and one previously-undocumented real bug (go-DDS/rust-DDS's
CycloneDDS image tag was never published) that the audit itself missed.
