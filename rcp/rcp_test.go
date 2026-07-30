// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package rcp

import (
	"errors"
	"reflect"
	"testing"

	relay "github.com/SoundMatt/RELAY/v2"
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
	orig := Message{ByteBusID: 9, TransactionNum: 42, Control: FlagWrite, ReadSizeOrSegment: 4, Body: []byte{0xAA}}
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
	if msg.Meta["rcp.transaction_num"] != "42" {
		t.Errorf("rcp.transaction_num = %q, want 42", msg.Meta["rcp.transaction_num"])
	}
	if msg.Meta["rcp.read_size_or_segment"] != "4" {
		t.Errorf("rcp.read_size_or_segment = %q, want 4", msg.Meta["rcp.read_size_or_segment"])
	}

	got, err := FromMessage(msg)
	if err != nil {
		t.Fatalf("FromMessage: %v", err)
	}
	if !reflect.DeepEqual(got, orig) {
		t.Errorf("round-trip mismatch: got %+v, want %+v", got, orig)
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
	// FlagResponse is one of the three Control bits §15.7.5 does not map
	// (alongside FlagAck/FlagMoreSegments) — it must NOT survive the round
	// trip. This asserts the documented gap stays visible instead of being
	// hidden by only checking the bit the test happens to care about.
	if got.Control.Has(FlagResponse) {
		t.Errorf("FlagResponse unexpectedly preserved through round-trip (§15.7.5 does not map it): %+v", got)
	}
	if want := FlagRead | FlagError; got.Control != want {
		t.Errorf("Control = %v, want %v (FlagResponse is dropped, not preserved as FlagRead/FlagWrite; see the ToMessage doc comment)", got.Control, want)
	}
}
