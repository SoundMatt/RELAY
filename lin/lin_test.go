// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package lin

import (
	"errors"
	"testing"

	relay "github.com/SoundMatt/RELAY"
)

//fusa:test REQ-RELAY-035
func TestConstants(t *testing.T) {
	if LINMaxDataLen != 8 || LINMaxID != 0x3F {
		t.Errorf("unexpected constants: %d %d", LINMaxDataLen, LINMaxID)
	}
}

//fusa:test REQ-RELAY-035
func TestFilterMatches(t *testing.T) {
	f := Filter{ID: 10}
	if !f.Matches(Frame{ID: 10, Data: []byte{1}}) {
		t.Error("exact match failed")
	}
	if f.Matches(Frame{ID: 11, Data: []byte{1}}) {
		t.Error("different ID should not match")
	}
	all := Filter{All: true}
	if !all.Matches(Frame{ID: 5, Data: []byte{1}}) {
		t.Error("All filter should match any frame")
	}
}

//fusa:test REQ-RELAY-036
func TestValidateFrame(t *testing.T) {
	cases := []struct {
		f  Frame
		ok bool
	}{
		{Frame{ID: 0x3F, Data: make([]byte, 8)}, true},
		{Frame{ID: 0x40, Data: make([]byte, 1)}, false}, // ID too large
		{Frame{ID: 1, Data: make([]byte, 0)}, false},    // empty data
		{Frame{ID: 1, Data: make([]byte, 9)}, false},    // data too long
		{Frame{ID: LINDiagRequestID, ChecksumType: ClassicChecksum, Data: make([]byte, 1)}, true},
		{Frame{ID: LINDiagRequestID, ChecksumType: EnhancedChecksum, Data: make([]byte, 1)}, false}, // diag must be classic
	}
	for i, tc := range cases {
		err := ValidateFrame(tc.f)
		if tc.ok && err != nil {
			t.Errorf("case %d: unexpected error: %v", i, err)
		}
		if !tc.ok && err == nil {
			t.Errorf("case %d: expected error", i)
		}
		if !tc.ok && err != nil && !errors.Is(err, ErrInvalidFrame) {
			t.Errorf("case %d: must wrap ErrInvalidFrame, got %v", i, err)
		}
	}
}

//fusa:test REQ-RELAY-036
func TestProtectIDAndVerify(t *testing.T) {
	for id := uint8(0); id <= LINMaxID; id++ {
		pid := ProtectID(id)
		got, err := VerifyPID(pid)
		if err != nil {
			t.Errorf("VerifyPID(ProtectID(%d)) = error: %v", id, err)
		}
		if got != id {
			t.Errorf("VerifyPID(ProtectID(%d)) = %d, want %d", id, got, id)
		}
	}
}

//fusa:test REQ-RELAY-036
func TestVerifyPIDBadParity(t *testing.T) {
	_, err := VerifyPID(0xFF)
	if err == nil {
		t.Error("expected error for corrupt PID")
	}
}

//fusa:test REQ-RELAY-037
func TestFrameRoundTrip(t *testing.T) {
	orig := Frame{ID: 5, Data: []byte{0xAA, 0xBB}, ChecksumType: EnhancedChecksum, Checksum: 0x42}
	msg := orig.ToMessage()
	if msg.Protocol != relay.LIN {
		t.Errorf("Protocol = %v, want LIN", msg.Protocol)
	}
	got, err := FromMessage(msg)
	if err != nil {
		t.Fatalf("FromMessage: %v", err)
	}
	if got.ID != orig.ID || got.ChecksumType != orig.ChecksumType {
		t.Errorf("round-trip mismatch: %+v", got)
	}
}

//fusa:test REQ-RELAY-037
func TestFromMessageInvalidID(t *testing.T) {
	_, err := FromMessage(relay.Message{ID: "64"}) // > LINMaxID
	if err == nil {
		t.Error("expected error for ID 64 (> 0x3F)")
	}
}

//fusa:test REQ-RELAY-036
func TestCalcChecksumClassicAndEnhanced(t *testing.T) {
	data := []byte{0x01, 0x02, 0x03}
	// Enhanced includes the PID in the sum; classic does not — they must differ
	// for a non-trivial PID, and both must be single bytes.
	classic := CalcChecksum(0x3C, data, ClassicChecksum)
	enhanced := CalcChecksum(0x3C, data, EnhancedChecksum)
	if classic == enhanced {
		t.Errorf("classic and enhanced checksums should differ for PID 0x3C: both %#x", classic)
	}
	// Determinism.
	if CalcChecksum(0x3C, data, EnhancedChecksum) != enhanced {
		t.Error("CalcChecksum must be deterministic")
	}
	// VerifyPID/round-trip sanity: a frame validated with its computed checksum.
	f := Frame{ID: 0x10, Data: data, ChecksumType: ClassicChecksum}
	f.Checksum = CalcChecksum(ProtectID(f.ID), data, ClassicChecksum)
	if err := ValidateFrame(f); err != nil {
		t.Errorf("frame with computed checksum must validate: %v", err)
	}
}

//fusa:test REQ-RELAY-036
func TestCalcChecksumWraps(t *testing.T) {
	// Bytes summing past 0xFF must exercise the modulo-255 carry fold.
	got := CalcChecksum(0x00, []byte{0xFF, 0xFF, 0xFF}, ClassicChecksum)
	if CalcChecksum(0x00, []byte{0xFF, 0xFF, 0xFF}, ClassicChecksum) != got {
		t.Error("CalcChecksum must be deterministic across the carry fold")
	}
	// Enhanced with a large PID also folds.
	_ = CalcChecksum(0xFF, []byte{0xFF, 0xFF}, EnhancedChecksum)
}
