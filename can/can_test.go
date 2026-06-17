// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package can

import (
	"errors"
	"testing"

	relay "github.com/SoundMatt/RELAY"
)

//fusa:test REQ-RELAY-030
func TestFrameConstants(t *testing.T) {
	if CANMaxDataLen != 8 || CANFDMaxDataLen != 64 {
		t.Errorf("unexpected data len constants: %d %d", CANMaxDataLen, CANFDMaxDataLen)
	}
	if MaxDataLen(false) != 8 || MaxDataLen(true) != 64 {
		t.Errorf("MaxDataLen wrong: %d %d", MaxDataLen(false), MaxDataLen(true))
	}
}

//fusa:test REQ-RELAY-030
func TestFilterMatches(t *testing.T) {
	f := Filter{ID: 0x100, Mask: 0x7FF}
	if !f.Matches(Frame{ID: 0x100}) {
		t.Error("exact match should succeed")
	}
	if f.Matches(Frame{ID: 0x101}) {
		t.Error("different ID should not match")
	}
}

//fusa:test REQ-RELAY-031
func TestValidateFrame(t *testing.T) {
	cases := []struct {
		f  Frame
		ok bool
	}{
		{Frame{ID: 0x7FF, Data: make([]byte, 8)}, true},
		{Frame{ID: 0x800, Data: make([]byte, 1)}, false}, // std ID too large
		{Frame{ID: 0x1FFFFFFF, Ext: true, Data: make([]byte, 8)}, true},
		{Frame{ID: 0x20000000, Ext: true, Data: make([]byte, 1)}, false},   // ext ID too large
		{Frame{ID: 1, BRS: true, FD: false, Data: make([]byte, 1)}, false}, // BRS without FD
		{Frame{ID: 1, RTR: true, FD: true, Data: make([]byte, 1)}, false},  // RTR with FD
		{Frame{ID: 1, FD: true, Data: make([]byte, 64)}, true},
		{Frame{ID: 1, FD: true, Data: make([]byte, 65)}, false}, // FD data too long
		{Frame{ID: 1, Data: make([]byte, 9)}, false},            // classic data too long
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
			t.Errorf("case %d: error must wrap ErrInvalidFrame, got %v", i, err)
		}
	}
}

//fusa:test REQ-RELAY-031
func TestValidateFrameNotPayloadTooLarge(t *testing.T) {
	// ValidateFrame must return ErrInvalidFrame, never relay.ErrPayloadTooLarge.
	f := Frame{ID: 1, Data: make([]byte, 100)}
	err := ValidateFrame(f)
	if errors.Is(err, relay.ErrPayloadTooLarge) {
		t.Error("ValidateFrame must not return relay.ErrPayloadTooLarge")
	}
}

//fusa:test REQ-RELAY-032
func TestToFromMessage(t *testing.T) {
	orig := Frame{ID: 0x123, Ext: true, FD: true, BRS: true, Data: []byte{1, 2, 3}}
	msg := orig.ToMessage()

	if msg.Protocol != relay.CAN {
		t.Errorf("Protocol = %v, want CAN", msg.Protocol)
	}
	if msg.ID != "291" { // 0x123 = 291
		t.Errorf("ID = %q, want %q", msg.ID, "291")
	}

	got, err := FromMessage(msg)
	if err != nil {
		t.Fatalf("FromMessage: %v", err)
	}
	if got.ID != orig.ID || got.Ext != orig.Ext || got.FD != orig.FD || got.BRS != orig.BRS {
		t.Errorf("round-trip mismatch: %+v != %+v", got, orig)
	}
}

//fusa:test REQ-RELAY-032
func TestFromMessageInvalidID(t *testing.T) {
	_, err := FromMessage(relay.Message{ID: "not-a-number"})
	if err == nil {
		t.Fatal("expected error for non-numeric ID")
	}
	if !errors.Is(err, ErrInvalidFrame) {
		t.Errorf("error must wrap ErrInvalidFrame, got %v", err)
	}
}
