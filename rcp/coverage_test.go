// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package rcp

import "testing"

//fusa:test REQ-RELAY-040
func TestControlFlagsHas(t *testing.T) {
	f := FlagResponse | FlagWrite | FlagError
	if !f.Has(FlagResponse) || !f.Has(FlagWrite) || !f.Has(FlagError) {
		t.Errorf("Has() missed a set bit: %08b", f)
	}
	if f.Has(FlagRead) || f.Has(FlagAck) || f.Has(FlagMoreSegments) {
		t.Errorf("Has() reported an unset bit: %08b", f)
	}
	if !f.Has(FlagWrite | FlagError) {
		t.Error("Has() must accept a multi-bit want mask")
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
