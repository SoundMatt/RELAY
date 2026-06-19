// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package rcp

import (
	"testing"

	relay "github.com/SoundMatt/RELAY"
)

//fusa:test REQ-RELAY-040
func TestZoneString(t *testing.T) {
	cases := []struct {
		z    Zone
		want string
	}{
		{ZoneFrontLeft, "FrontLeft"},
		{ZoneFrontRight, "FrontRight"},
		{ZoneRearLeft, "RearLeft"},
		{ZoneRearRight, "RearRight"},
		{ZoneCentral, "Central"},
		{ZoneUnknown, "Unknown"},
	}
	for _, tc := range cases {
		if got := tc.z.String(); got != tc.want {
			t.Errorf("Zone(%d).String() = %q, want %q", tc.z, got, tc.want)
		}
	}
}

//fusa:test REQ-RELAY-040
func TestZoneFromString(t *testing.T) {
	for _, z := range []Zone{ZoneFrontLeft, ZoneFrontRight, ZoneRearLeft, ZoneRearRight, ZoneCentral} {
		if got := ZoneFromString(z.String()); got != z {
			t.Errorf("ZoneFromString(%q) = %v, want %v", z.String(), got, z)
		}
	}
	if ZoneFromString("nonsense") != ZoneUnknown {
		t.Error("unknown zone name must return ZoneUnknown")
	}
}

//fusa:test REQ-RELAY-041
func TestStatusRoundTrip(t *testing.T) {
	orig := Status{Zone: ZoneFrontLeft, Seq: 7, Healthy: true, Payload: []byte{1}}
	msg := orig.ToMessage()
	if msg.Protocol != relay.RCP {
		t.Errorf("Protocol = %v, want RCP", msg.Protocol)
	}
	if msg.ID != "FrontLeft" {
		t.Errorf("ID = %q, want FrontLeft", msg.ID)
	}
	if msg.Meta["rcp.healthy"] != "true" {
		t.Errorf("rcp.healthy = %q", msg.Meta["rcp.healthy"])
	}
	if msg.Seq != 7 {
		t.Errorf("Seq = %d, want 7", msg.Seq)
	}

	got, err := StatusFromMessage(msg)
	if err != nil {
		t.Fatalf("StatusFromMessage: %v", err)
	}
	if got.Zone != ZoneFrontLeft || !got.Healthy || got.Seq != 7 {
		t.Errorf("round-trip mismatch: %+v", got)
	}
}

//fusa:test REQ-RELAY-041
func TestResponseToMessage(t *testing.T) {
	r := Response{CommandID: 1, Zone: ZoneRearRight, Status: StatusOK, Payload: []byte{9}}
	msg := r.ToMessage()
	if msg.Protocol != relay.RCP {
		t.Errorf("Protocol = %v, want RCP", msg.Protocol)
	}
	if msg.ID != "RearRight" {
		t.Errorf("ID = %q, want RearRight", msg.ID)
	}
	if msg.Meta["rcp.status"] != "0" {
		t.Errorf("rcp.status = %q, want 0", msg.Meta["rcp.status"])
	}
}

//fusa:test REQ-RELAY-041
func TestCommandFromMessage(t *testing.T) {
	msg := relay.Message{
		Protocol: relay.RCP,
		ID:       "Central",
		Payload:  []byte{5},
		Meta: map[string]string{
			"rcp.priority": "high",
			"rcp.cmd_type": "set",
		},
	}
	cmd, err := CommandFromMessage(msg)
	if err != nil {
		t.Fatalf("CommandFromMessage: %v", err)
	}
	if cmd.Zone != ZoneCentral {
		t.Errorf("Zone = %v, want Central", cmd.Zone)
	}
	if cmd.Priority != PriorityHigh {
		t.Errorf("Priority = %v, want High", cmd.Priority)
	}
	if cmd.Type != CmdSet {
		t.Errorf("Type = %v, want Set", cmd.Type)
	}
}

//fusa:test REQ-RELAY-041
func TestCommandFromMessageZones(t *testing.T) {
	// Unknown zone name must error with ErrInvalidZone.
	_, err := CommandFromMessage(relay.Message{ID: "Nowhere"})
	if err == nil {
		t.Fatal("expected error for unknown zone")
	}
	// The literal "Unknown" zone is valid (maps to ZoneUnknown, not an error).
	cmd, err := CommandFromMessage(relay.Message{ID: "Unknown"})
	if err != nil || cmd.Zone != ZoneUnknown {
		t.Errorf("CommandFromMessage(Unknown) = %+v, %v; want ZoneUnknown, nil", cmd, err)
	}
}
