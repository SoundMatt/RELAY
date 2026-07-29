// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package rcp

import (
	"errors"
	"testing"

	relay "github.com/SoundMatt/RELAY"
)

//fusa:test REQ-RELAY-041
func TestEndpointIDRoundTrip(t *testing.T) {
	for _, addr := range []ByteBusID{0, 9, 255} {
		s := EndpointIDString(addr)
		got, err := ParseEndpointID(s)
		if err != nil {
			t.Fatalf("ParseEndpointID(%q): %v", s, err)
		}
		if got != addr {
			t.Errorf("round-trip mismatch: got %d, want %d", got, addr)
		}
	}
}

//fusa:test REQ-RELAY-041
func TestParseEndpointIDRejectsInvalid(t *testing.T) {
	for _, id := range []string{"", "nonsense", "-1", "256", "9.5"} {
		if _, err := ParseEndpointID(id); !errors.Is(err, ErrNotFound) {
			t.Errorf("ParseEndpointID(%q) error = %v, want ErrNotFound", id, err)
		}
	}
}

//fusa:test REQ-RELAY-041
func TestMessageToMessageRoundTrip(t *testing.T) {
	orig := Message{ByteBusID: 9, Control: FlagWrite, Body: []byte{0xAA}}
	msg := orig.ToMessage()
	if msg.Protocol != relay.RCP {
		t.Errorf("Protocol = %v, want RCP", msg.Protocol)
	}
	if msg.ID != "9" {
		t.Errorf("ID = %q, want 9", msg.ID)
	}
	if msg.Meta["rcp.op"] != "write" {
		t.Errorf("rcp.op = %q, want write", msg.Meta["rcp.op"])
	}
	if msg.Meta["rcp.error"] != "false" {
		t.Errorf("rcp.error = %q, want false", msg.Meta["rcp.error"])
	}

	got, err := FromMessage(msg)
	if err != nil {
		t.Fatalf("FromMessage: %v", err)
	}
	if got.ByteBusID != 9 || !got.Control.Has(FlagWrite) || got.Control.Has(FlagError) {
		t.Errorf("round-trip mismatch: %+v", got)
	}
}

//fusa:test REQ-RELAY-041
func TestFromMessageDefaultsOpFromPayload(t *testing.T) {
	write, err := FromMessage(relay.Message{ID: "1", Payload: []byte{1}})
	if err != nil || !write.Control.Has(FlagWrite) {
		t.Errorf("FromMessage(non-empty payload) = %+v, %v; want FlagWrite", write, err)
	}
	read, err := FromMessage(relay.Message{ID: "1"})
	if err != nil || !read.Control.Has(FlagRead) {
		t.Errorf("FromMessage(empty payload) = %+v, %v; want FlagRead", read, err)
	}
}

//fusa:test REQ-RELAY-041
func TestFromMessageError(t *testing.T) {
	if _, err := FromMessage(relay.Message{ID: "Nowhere"}); !errors.Is(err, ErrNotFound) {
		t.Errorf("FromMessage(bad id) error = %v, want ErrNotFound", err)
	}
}

//fusa:test REQ-RELAY-041
func TestMessageToMessageError(t *testing.T) {
	msg := Message{ByteBusID: 9, Control: FlagResponse | FlagError, Body: []byte{1}}.ToMessage()
	if msg.Meta["rcp.error"] != "true" {
		t.Errorf("rcp.error = %q, want true", msg.Meta["rcp.error"])
	}
	got, err := FromMessage(msg)
	if err != nil {
		t.Fatalf("FromMessage: %v", err)
	}
	if !got.Control.Has(FlagError) {
		t.Errorf("FlagError not preserved through round-trip: %+v", got)
	}
}
