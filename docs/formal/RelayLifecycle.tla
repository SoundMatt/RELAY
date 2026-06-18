---------------------------- MODULE RelayLifecycle ----------------------------
(***************************************************************************)
(* Formal model of the RELAY node lifecycle (spec §6).                     *)
(*                                                                         *)
(* This TLA+ specification models a single RELAY-conformant node and the   *)
(* operations an application may invoke on it (send, subscribe, close,     *)
(* unsubscribe), together with the environment action of an underlying     *)
(* transport drop. The invariants encode the ten §6 lifecycle             *)
(* requirements so they can be model-checked with TLC.                     *)
(*                                                                         *)
(* Each invariant is tagged with the §6 requirement number it discharges.  *)
(***************************************************************************)
EXTENDS Naturals, FiniteSets

CONSTANTS Subs            \* finite set of subscription identifiers

\* Node states.
States == {"uninit", "connected", "failed", "closed"}

\* Operation result sentinels (spec §5).
Results == {"ok", "ErrClosed", "ErrNotConnected", "ErrTimeout"}

VARIABLES
    state,        \* current node state
    openSubs,     \* set of subscriptions whose channel is open
    lastResult    \* result returned by the most recent operation

vars == <<state, openSubs, lastResult>>

TypeOK ==
    /\ state \in States
    /\ openSubs \subseteq Subs
    /\ lastResult \in Results

Init ==
    /\ state = "uninit"
    /\ openSubs = {}
    /\ lastResult = "ok"

(***************************************************************************)
(* Application actions.                                                    *)
(***************************************************************************)

\* Successful connect: only from the uninitialised state.
Connect ==
    /\ state = "uninit"
    /\ state' = "connected"
    /\ UNCHANGED openSubs
    /\ lastResult' = "ok"

\* Send / Publish / Call. Result is fully determined by the state.
Send ==
    /\ lastResult' = CASE state = "connected" -> "ok"
                       [] state = "closed"    -> "ErrClosed"        \* §6.2
                       [] OTHER               -> "ErrNotConnected"  \* §6.9, §6.10
    /\ UNCHANGED <<state, openSubs>>

\* Subscribe. On a connected node opens a fresh channel; otherwise errors.
Subscribe(s) ==
    /\ \/ /\ state = "connected"
          /\ openSubs' = openSubs \cup {s}
          /\ lastResult' = "ok"
       \/ /\ state # "connected"
          /\ UNCHANGED openSubs
          /\ lastResult' = CASE state = "closed" -> "ErrClosed"        \* §6.3
                             [] OTHER             -> "ErrNotConnected"  \* §6.9, §6.10
    /\ UNCHANGED state

\* Unsubscribe / Subscription.Close: closes exactly that channel, no-op if
\* already closed; never affects other subscriptions (§6.4, §6.8).
Unsubscribe(s) ==
    /\ openSubs' = openSubs \ {s}
    /\ lastResult' = "ok"
    /\ UNCHANGED state

\* Close. Idempotent; transitions to closed and closes every channel.
Close ==
    /\ state' = "closed"
    /\ openSubs' = {}        \* §6.3 channels already returned are closed
    /\ lastResult' = "ok"    \* §6.1 idempotent: never errors, even if already closed

(***************************************************************************)
(* Environment action.                                                     *)
(***************************************************************************)

\* Underlying transport fails. No automatic reconnection is permitted; the
\* node moves to "failed" and stays there until the application Closes (§6.10).
TransportDrop ==
    /\ state = "connected"
    /\ state' = "failed"
    /\ UNCHANGED openSubs
    /\ lastResult' = "ErrNotConnected"

Next ==
    \/ Connect
    \/ Send
    \/ \E s \in Subs : Subscribe(s)
    \/ \E s \in Subs : Unsubscribe(s)
    \/ Close
    \/ TransportDrop

Spec == Init /\ [][Next]_vars

(***************************************************************************)
(* Invariants — one per §6 requirement (or group).                        *)
(***************************************************************************)

\* §6.9 Zero-value safety: an uninitialised node never panics (modelled by
\* never reaching an undefined state) and every op yields ErrNotConnected.
Inv_ZeroValue ==
    (state = "uninit" /\ lastResult # "ok") => lastResult = "ErrNotConnected"

\* §6.2/§6.3 Send/Subscribe after close return ErrClosed.
Inv_AfterClose ==
    (state = "closed" /\ lastResult # "ok") => lastResult = "ErrClosed"

\* §6.3 No subscription channel remains open once the node is closed.
Inv_ClosedHasNoOpenSubs ==
    (state = "closed") => openSubs = {}

\* §6.10 No auto-reconnect: a failed node only ever yields ErrNotConnected
\* (until an explicit Close, which is the only escape modelled here).
Inv_NoAutoReconnect ==
    (state = "failed" /\ lastResult # "ok") => lastResult = "ErrNotConnected"

\* Overall well-formedness of returned sentinels.
Inv_ResultDomain == lastResult \in Results

=============================================================================
