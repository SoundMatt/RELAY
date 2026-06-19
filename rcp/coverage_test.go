// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package rcp

import "testing"

//fusa:test REQ-RELAY-040
func TestPriorityStringRoundTrip(t *testing.T) {
	cases := []struct {
		p    Priority
		want string
	}{
		{PriorityNormal, "normal"},
		{PriorityHigh, "high"},
		{PriorityCritical, "critical"},
	}
	for _, tc := range cases {
		if got := tc.p.String(); got != tc.want {
			t.Errorf("Priority(%d).String() = %q, want %q", tc.p, got, tc.want)
		}
		if got := priorityFromString(tc.want); got != tc.p {
			t.Errorf("priorityFromString(%q) = %v, want %v", tc.want, got, tc.p)
		}
	}
	if priorityFromString("nonsense") != PriorityNormal {
		t.Error("unknown priority must default to PriorityNormal")
	}
}

//fusa:test REQ-RELAY-040
func TestCommandTypeStringRoundTrip(t *testing.T) {
	cases := []struct {
		c    CommandType
		want string
	}{
		{CmdNoop, "noop"},
		{CmdSet, "set"},
		{CmdGet, "get"},
		{CmdReset, "reset"},
		{CmdWatchdog, "watchdog"},
		{CmdSleep, "sleep"},
		{CmdWake, "wake"},
	}
	for _, tc := range cases {
		if got := tc.c.String(); got != tc.want {
			t.Errorf("CommandType(%d).String() = %q, want %q", tc.c, got, tc.want)
		}
		if got := cmdTypeFromString(tc.want); got != tc.c {
			t.Errorf("cmdTypeFromString(%q) = %v, want %v", tc.want, got, tc.c)
		}
	}
	if cmdTypeFromString("nonsense") != CmdNoop {
		t.Error("unknown command type must default to CmdNoop")
	}
}

//fusa:test REQ-RELAY-040
func TestLoanReturn(t *testing.T) {
	// Return must invoke the release function exactly once when set.
	released := 0
	l := Loan{Payload: []byte{1, 2}, release: func() { released++ }}
	l.Return()
	if released != 1 {
		t.Errorf("release called %d times, want 1", released)
	}
	// Return must be a safe no-op when release is nil (zero-value Loan).
	(&Loan{}).Return()
}
