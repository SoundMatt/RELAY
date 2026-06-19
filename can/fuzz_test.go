// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package can

import (
	"reflect"
	"testing"
)

// FuzzValidateFrameNoPanic asserts ValidateFrame is total: it must return an
// error or nil for any input, never panic (SG-001 robustness).
//
//fusa:test REQ-RELAY-031
func FuzzValidateFrameNoPanic(f *testing.F) {
	f.Add(uint32(0x123), true, true, true, true, []byte{1, 2, 3})
	f.Add(uint32(0xFFFFFFFF), false, false, false, false, []byte(nil))
	f.Fuzz(func(t *testing.T, id uint32, ext, fd, xl, esi bool, data []byte) {
		_ = ValidateFrame(Frame{ID: id, Ext: ext, FD: fd, XL: xl, ESI: esi, Data: data})
	})
}

// FuzzRoundTripLossless asserts FromMessage(ToMessage(f)) == f for every valid
// classic/FD frame — the §15.7 losslessness property (SG-002).
//
//fusa:test REQ-RELAY-032
func FuzzRoundTripLossless(f *testing.F) {
	f.Add(uint32(0x123), true, true, true, []byte{1, 2, 3})
	f.Fuzz(func(t *testing.T, id uint32, fd, brs, esi bool, data []byte) {
		fr := Frame{
			ID:   id & CANMaxExtID,
			Ext:  true,
			FD:   fd,
			BRS:  brs && fd, // BRS requires FD
			ESI:  esi && fd, // ESI requires FD or XL
			Data: data,
		}
		if ValidateFrame(fr) != nil {
			return // only valid frames are required to round-trip
		}
		got, err := FromMessage(fr.ToMessage())
		if err != nil {
			t.Fatalf("FromMessage on a valid frame failed: %v", err)
		}
		if !reflect.DeepEqual(got, fr) {
			t.Errorf("round-trip not lossless:\n got: %+v\nwant: %+v", got, fr)
		}
	})
}
