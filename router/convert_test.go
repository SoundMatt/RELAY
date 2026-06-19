// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package router

import (
	"testing"

	relay "github.com/SoundMatt/RELAY"
)

//fusa:test REQ-RELAY-085
func TestConverters(t *testing.T) {
	m := relay.Message{Protocol: relay.CAN, ID: "1", Payload: []byte{1}, Meta: map[string]string{"k": "v"}}
	// Identity preserves everything.
	if got, _ := Identity(m); got.Protocol != relay.CAN || got.ID != "1" {
		t.Error("Identity must preserve the message")
	}
	// Retag changes only the protocol.
	got, err := Retag(relay.MQTT)(m)
	if err != nil || got.Protocol != relay.MQTT || got.ID != "1" || got.Meta["k"] != "v" {
		t.Errorf("Retag wrong: %+v err=%v", got, err)
	}
	// DefaultConverter: identity for same protocol, retag otherwise.
	if c := DefaultConverter(relay.CAN, relay.CAN); func() bool { o, _ := c(m); return o.Protocol != relay.CAN }() {
		t.Error("same-protocol default must be identity")
	}
	if c := DefaultConverter(relay.CAN, relay.DDS); func() bool { o, _ := c(m); return o.Protocol != relay.DDS }() {
		t.Error("cross-protocol default must retag")
	}
}

//fusa:test REQ-RELAY-085
func TestLookup(t *testing.T) {
	for _, name := range []string{"identity", "to-can", "to-mqtt", "to-someip"} {
		if _, err := Lookup(name); err != nil {
			t.Errorf("Lookup(%q): %v", name, err)
		}
	}
	if _, err := Lookup("nope"); err == nil {
		t.Error("unknown converter must error")
	}
}
