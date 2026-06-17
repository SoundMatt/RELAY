// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package someip

import (
	"errors"
	"testing"

	relay "github.com/SoundMatt/RELAY"
)

//fusa:test REQ-RELAY-042
func TestProtocolVersion(t *testing.T) {
	if SOMEIPProtocolVersion != 0x01 {
		t.Errorf("SOMEIPProtocolVersion = 0x%02X, want 0x01", SOMEIPProtocolVersion)
	}
}

//fusa:test REQ-RELAY-042
func TestMessageValidate(t *testing.T) {
	good := Message{ServiceID: 1, MethodID: 2, ProtocolVersion: SOMEIPProtocolVersion}
	if err := good.Validate(); err != nil {
		t.Errorf("valid message: %v", err)
	}
	bad := Message{ServiceID: 1, MethodID: 2, ProtocolVersion: 0x02}
	if err := bad.Validate(); err == nil {
		t.Error("expected error for wrong protocol version")
	} else if !errors.Is(err, ErrWrongProtocolVersion) {
		t.Errorf("error must wrap ErrWrongProtocolVersion, got %v", err)
	}
}

//fusa:test REQ-RELAY-042
func TestMessageTypeString(t *testing.T) {
	if MsgTypeRequest.String() != "request" {
		t.Errorf("MsgTypeRequest.String() = %q", MsgTypeRequest.String())
	}
	if MsgTypeError.String() != "error" {
		t.Errorf("MsgTypeError.String() = %q", MsgTypeError.String())
	}
}

//fusa:test REQ-RELAY-043
func TestMessageRoundTrip(t *testing.T) {
	orig := Message{
		ServiceID:        0x1234,
		MethodID:         0x5678,
		ProtocolVersion:  SOMEIPProtocolVersion,
		InterfaceVersion: 2,
		MessageType:      MsgTypeRequest,
		ReturnCode:       RetOK,
		Payload:          []byte{0xCA, 0xFE},
	}
	msg := orig.ToMessage()
	if msg.Protocol != relay.SOMEIP {
		t.Errorf("Protocol = %v, want SOMEIP", msg.Protocol)
	}
	if msg.ID != "4660/22136" { // 0x1234=4660, 0x5678=22136
		t.Errorf("ID = %q, want %q", msg.ID, "4660/22136")
	}
	if msg.Meta["someip.msg_type"] != "request" {
		t.Errorf("msg_type = %q", msg.Meta["someip.msg_type"])
	}

	got, err := FromMessage(msg)
	if err != nil {
		t.Fatalf("FromMessage: %v", err)
	}
	if got.ServiceID != orig.ServiceID || got.MethodID != orig.MethodID {
		t.Errorf("round-trip IDs mismatch: svc=%d meth=%d", got.ServiceID, got.MethodID)
	}
	if got.InterfaceVersion != orig.InterfaceVersion {
		t.Errorf("InterfaceVersion = %d, want %d", got.InterfaceVersion, orig.InterfaceVersion)
	}
}

//fusa:test REQ-RELAY-043
func TestFromMessageInvalidID(t *testing.T) {
	_, err := FromMessage(relay.Message{ID: "notvalid"})
	if err == nil {
		t.Error("expected error for non-slash ID")
	}
	if !errors.Is(err, ErrInvalidID) {
		t.Errorf("must wrap ErrInvalidID, got %v", err)
	}
}
