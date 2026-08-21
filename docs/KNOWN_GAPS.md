# Known Implementation Gaps

## Status

This document tracks **specific, named x-Net implementations' current
deviations** from `spec/relay-spec.md`'s normative requirements — the class
of note that used to live as inline asides inside the spec's own prose
(e.g. "this is a breaking change from go-CAN's current signature"). It
exists so that content has somewhere to live that isn't the spec itself:
a per-implementation status note goes stale on its own schedule, entirely
independent of when the spec's own normative text last changed, and mixing
the two meant the spec's own prose either had to be re-reviewed every time
an implementation shipped a fix, or — as happened — silently went stale
while still reading as current.

**This document is explicitly non-normative.** It records what a specific
implementation currently does or doesn't do; it does not itself define or
change any RELAY requirement. `spec/relay-spec.md` remains the sole
normative source. Nothing in this file gates `relay conform` or any other
CI check.

**Staleness caveat.** Each entry below was moved here verbatim from the
spec's own prose (issue #134 / [REL-SPEC-7]) on 2026-08-21, not
independently re-verified against the named repository's current source at
that time. Appendix A's own table (`spec/relay-spec.md`) — the table these
notes originally cross-referenced — is itself explicitly marked "not
maintained thereafter" as of the v0.2 milestone, which is exactly the
staleness risk that motivated moving this content out of normative prose
in the first place. Treat every entry here as a claim to verify against the
named repository's own current source before relying on it, not as a
live-tracked fact.

## Entries

### go-CAN: `Subscribe()` signature

`spec/relay-spec.md` §8.1 specifies `Subscribe(filters []Filter, opts
...SubscriberOption)` — a slice, not variadic, to avoid ambiguity with the
variadic `opts`. As of the spec text this entry was extracted from, this
was a breaking change from go-CAN's own then-current
`Subscribe(filters ...Filter)` signature.

### go-CAN: `ValidateFrame` RTR/FD check

`spec/relay-spec.md` §15.1's `ValidateFrame` constraints include "RTR MUST
be false when FD is true". As of the spec text this entry was extracted
from, go-CAN's own `ValidateFrame` did not enforce this specific check.

### go-LIN: `SetSchedule`

`spec/relay-spec.md` §8.3 documents a `SetSchedule` method on the LIN
`MasterBus` interface. As of the spec text this entry was extracted from,
this method was new relative to go-LIN's own then-current interface.

### go-SOMEIP: `SubscriberConfig`/`SubscriberOption` naming

`spec/relay-spec.md` §14.1 requires the names `SubscriberConfig` and
`SubscriberOption` to be used consistently across all protocols. As of the
spec text this entry was extracted from, go-SOMEIP used
`SubscribeConfig`/`SubscribeOption` instead.
