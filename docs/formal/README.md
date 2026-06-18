# Formal Verification — RELAY Node Lifecycle (§6)

`RelayLifecycle.tla` is a TLA+ model of the RELAY node lifecycle. It exists so
the §6 lifecycle requirements are not only prose but a machine-checkable state
machine: TLC explores every reachable state and confirms the invariants below
hold on all of them.

## Running the model checker

With the [TLA+ tools](https://github.com/tlaplus/tlaplus) installed:

```sh
tlc RelayLifecycle.tla -config RelayLifecycle.cfg
```

TLC reports "Model checking completed. No error has been found." when all
invariants hold over the full reachable state space.

## Model

A single node moves through four states — `uninit`, `connected`, `failed`,
`closed` — under application actions (`Connect`, `Send`, `Subscribe`,
`Unsubscribe`, `Close`) and one environment action (`TransportDrop`). Each
operation records the sentinel it returns (`ok`, `ErrClosed`,
`ErrNotConnected`, `ErrTimeout`) so the invariants can constrain it.

## Requirement → invariant mapping

Each §6 lifecycle requirement is discharged by the model as follows:

| §6 requirement | How the model discharges it |
|---|---|
| 6.1 Idempotent close | `Close` always sets `lastResult = "ok"` and is enabled in every state, including `closed`. |
| 6.2 Send after close | `Inv_AfterClose`: in `closed`, `Send` yields `ErrClosed`. |
| 6.3 Receive after close | `Inv_AfterClose` (Subscribe → `ErrClosed`) + `Inv_ClosedHasNoOpenSubs` (channels closed). |
| 6.4 Unsubscribe semantics | `Unsubscribe(s)` removes only `s` and is a no-op when `s` is already closed. |
| 6.5 Context cancellation | Deadline expiry is modelled by the `ErrTimeout` sentinel in the result domain (`Inv_ResultDomain`). |
| 6.6 Concurrent close | `Close` is enabled concurrently with any in-flight action and always succeeds. |
| 6.7 Concurrent sends | `Send` is state-determined and side-effect-free on `state`/`openSubs`, so any interleaving is safe. |
| 6.8 Multiple subscriptions | `Subscribe(s)`/`Unsubscribe(s)` operate per-id; closing one never affects another. |
| 6.9 Zero-value safety | `Inv_ZeroValue`: in `uninit`, every non-ok result is `ErrNotConnected`. |
| 6.10 Reconnection policy | `Inv_NoAutoReconnect`: no action moves `failed → connected`; a failed node only yields `ErrNotConnected`. |

The model is embedded in the `relay` binary as evidence (`relay.Evidence("formal-model")`)
and bundled by `relay audit-pack`. The Go test `TestFormalModelCoversLifecycle`
asserts the model references all ten §6 requirements so the mapping cannot drift.
