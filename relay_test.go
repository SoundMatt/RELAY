// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package relay

import (
	"encoding/json"
	"testing"
	"time"
)

//fusa:test REQ-RELAY-001
//fusa:test REQ-RELAY-002
func TestProtocolValues(t *testing.T) {
	if Protocol(0) == CAN || Protocol(0) == DDS || Protocol(0) == LIN ||
		Protocol(0) == MQTT || Protocol(0) == RCP || Protocol(0) == SOMEIP {
		t.Error("protocol 0 (reserved) must not equal any named protocol")
	}
	cases := []struct {
		p    Protocol
		want int
	}{
		{CAN, 1}, {DDS, 2}, {LIN, 3}, {MQTT, 4}, {RCP, 5}, {SOMEIP, 6},
	}
	for _, c := range cases {
		if int(c.p) != c.want {
			t.Errorf("%s = %d, want %d", c.p, int(c.p), c.want)
		}
	}
}

//fusa:test REQ-RELAY-003
func TestProtocolString(t *testing.T) {
	cases := []struct {
		p    Protocol
		want string
	}{
		{CAN, "CAN"}, {DDS, "DDS"}, {LIN, "LIN"},
		{MQTT, "MQTT"}, {RCP, "RCP"}, {SOMEIP, "SOMEIP"},
		{Protocol(0), "unknown"}, {Protocol(99), "unknown"},
	}
	for _, c := range cases {
		if got := c.p.String(); got != c.want {
			t.Errorf("Protocol(%d).String() = %q, want %q", int(c.p), got, c.want)
		}
	}
}

//fusa:test REQ-RELAY-059
func TestParseProtocol(t *testing.T) {
	cases := []struct {
		in   string
		want Protocol
		ok   bool
	}{
		{"CAN", CAN, true}, {"can", CAN, true}, {" Dds ", DDS, true},
		{"LIN", LIN, true}, {"mqtt", MQTT, true}, {"RCP", RCP, true},
		{"SOMEIP", SOMEIP, true}, {"some/ip", SOMEIP, true},
		{"", 0, false}, {"bogus", 0, false},
	}
	for _, c := range cases {
		got, ok := ParseProtocol(c.in)
		if ok != c.ok || got != c.want {
			t.Errorf("ParseProtocol(%q) = (%v, %v), want (%v, %v)", c.in, got, ok, c.want, c.ok)
		}
	}
	// Round-trips with String for the named protocols.
	for _, p := range []Protocol{CAN, DDS, LIN, MQTT, RCP, SOMEIP} {
		if got, ok := ParseProtocol(p.String()); !ok || got != p {
			t.Errorf("round-trip failed for %s", p)
		}
	}
}

//fusa:test REQ-RELAY-004
//fusa:test REQ-RELAY-005
func TestVersionString(t *testing.T) {
	cases := []struct {
		v    Version
		want string
	}{
		{Version{1, 2, 3}, "1.2.3"},
		{Version{}, "0.0.0"},
		{Version{0, 1, 0}, "0.1.0"},
	}
	for _, c := range cases {
		if got := c.v.String(); got != c.want {
			t.Errorf("Version%v.String() = %q, want %q", c.v, got, c.want)
		}
	}
}

//fusa:test REQ-RELAY-006
func TestMessageJSONRoundTrip(t *testing.T) {
	ts := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	original := Message{
		Protocol:  CAN,
		Version:   Version{Major: 1},
		ID:        "256",
		Payload:   []byte{0x01, 0x02, 0x03},
		Timestamp: ts,
		Seq:       42,
		Meta:      map[string]string{"can.ext": "false", "can.fd": "true"},
	}
	b, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var got Message
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if got.Protocol != original.Protocol {
		t.Errorf("Protocol = %v, want %v", got.Protocol, original.Protocol)
	}
	if got.ID != original.ID {
		t.Errorf("ID = %q, want %q", got.ID, original.ID)
	}
	if got.Seq != original.Seq {
		t.Errorf("Seq = %d, want %d", got.Seq, original.Seq)
	}
	if string(got.Payload) != string(original.Payload) {
		t.Errorf("Payload = %v, want %v", got.Payload, original.Payload)
	}
	if got.Meta["can.fd"] != "true" {
		t.Errorf("Meta[can.fd] = %q, want %q", got.Meta["can.fd"], "true")
	}
}

//fusa:test REQ-RELAY-007
func TestMessageSeqOmitempty(t *testing.T) {
	m := Message{Protocol: CAN, ID: "1"}
	b, _ := json.Marshal(m)
	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if _, ok := raw["seq"]; ok {
		t.Error("seq must be omitted from JSON when zero")
	}
}

//fusa:test REQ-RELAY-007
func TestMessageMetaOmitempty(t *testing.T) {
	m := Message{Protocol: DDS, ID: "topic"}
	b, _ := json.Marshal(m)
	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if _, ok := raw["meta"]; ok {
		t.Error("meta must be omitted from JSON when nil")
	}
}
