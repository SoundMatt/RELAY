// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package can

import (
	"errors"
	"reflect"
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

//fusa:test REQ-RELAY-070
func TestXLMaxDataLen(t *testing.T) {
	if CANXLMaxDataLen != 2048 || CANXLMinDataLen != 1 {
		t.Errorf("unexpected XL constants: %d %d", CANXLMaxDataLen, CANXLMinDataLen)
	}
	if (Frame{}).MaxDataLen() != 8 {
		t.Error("classic MaxDataLen should be 8")
	}
	if (Frame{FD: true}).MaxDataLen() != 64 {
		t.Error("FD MaxDataLen should be 64")
	}
	if (Frame{XL: true}).MaxDataLen() != 2048 {
		t.Error("XL MaxDataLen should be 2048")
	}
}

//fusa:test REQ-RELAY-072
//fusa:test REQ-RELAY-071
func TestValidateXLFrame(t *testing.T) {
	cases := []struct {
		name string
		f    Frame
		ok   bool
	}{
		{"valid XL", Frame{ID: 0x7FF, XL: true, SDT: 3, VCID: 1, AF: 0xCAFE, Data: make([]byte, 2048)}, true},
		{"XL data too long", Frame{ID: 1, XL: true, Data: make([]byte, 2049)}, false},
		{"XL empty data", Frame{ID: 1, XL: true, Data: nil}, false},
		{"XL priority overflow", Frame{ID: 0x800, XL: true, Data: make([]byte, 1)}, false},
		{"XL with Ext", Frame{ID: 1, XL: true, Ext: true, Data: make([]byte, 1)}, false},
		{"XL with RTR", Frame{ID: 1, XL: true, RTR: true, Data: make([]byte, 1)}, false},
		{"XL with BRS", Frame{ID: 1, XL: true, BRS: true, Data: make([]byte, 1)}, false},
		{"FD and XL", Frame{ID: 1, FD: true, XL: true, Data: make([]byte, 1)}, false},
		{"ESI without FD/XL", Frame{ID: 1, ESI: true, Data: make([]byte, 1)}, false},
		{"ESI with FD", Frame{ID: 1, FD: true, ESI: true, Data: make([]byte, 1)}, true},
		{"ESI with XL", Frame{ID: 1, XL: true, ESI: true, Data: make([]byte, 1)}, true},
	}
	for _, tc := range cases {
		err := ValidateFrame(tc.f)
		if tc.ok && err != nil {
			t.Errorf("%s: unexpected error: %v", tc.name, err)
		}
		if !tc.ok {
			if err == nil {
				t.Errorf("%s: expected error", tc.name)
			} else if !errors.Is(err, ErrInvalidFrame) {
				t.Errorf("%s: error must wrap ErrInvalidFrame, got %v", tc.name, err)
			}
		}
	}
}

//fusa:test REQ-RELAY-073
func TestXLToFromMessageRoundTrip(t *testing.T) {
	orig := Frame{ID: 0x123, XL: true, ESI: true, SDT: 5, VCID: 2, AF: 51966, SEC: true, Data: []byte{0xDE, 0xAD, 0xBE, 0xEF}}
	got, err := FromMessage(orig.ToMessage())
	if err != nil {
		t.Fatalf("FromMessage: %v", err)
	}
	if !reflect.DeepEqual(got, orig) {
		t.Errorf("XL round-trip mismatch:\n got: %+v\nwant: %+v", got, orig)
	}
}

//fusa:test REQ-RELAY-073
func TestClassicMetaUnchanged(t *testing.T) {
	// A classic frame must not emit any XL/ESI Meta keys, keeping output stable.
	msg := Frame{ID: 1, Data: []byte{1}}.ToMessage()
	for _, k := range []string{"can.esi", "can.xl", "can.sdt", "can.vcid", "can.af", "can.sec"} {
		if _, ok := msg.Meta[k]; ok {
			t.Errorf("classic frame must not emit %s", k)
		}
	}
}
