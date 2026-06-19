// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package someip

import (
	"errors"
	"reflect"
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
		ClientID:         0x0009,
		SessionID:        0x00AB,
		ProtocolVersion:  SOMEIPProtocolVersion,
		InterfaceVersion: 2,
		MessageType:      MsgTypeNotification,
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
	// someip.msg_type carries the numeric value for lossless round-trip;
	// someip.msg_type_name carries the diagnostic label.
	if msg.Meta["someip.msg_type"] != "2" {
		t.Errorf("msg_type = %q, want \"2\"", msg.Meta["someip.msg_type"])
	}
	if msg.Meta["someip.msg_type_name"] != "notification" {
		t.Errorf("msg_type_name = %q, want \"notification\"", msg.Meta["someip.msg_type_name"])
	}

	got, err := FromMessage(msg)
	if err != nil {
		t.Fatalf("FromMessage: %v", err)
	}
	// The conversion MUST be lossless (§15.7, hazard H-002).
	if !reflect.DeepEqual(got, orig) {
		t.Errorf("round-trip not lossless:\n got: %+v\nwant: %+v", got, orig)
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

//fusa:test REQ-RELAY-042
func TestMessageTypeStringAllValues(t *testing.T) {
	cases := map[MessageType]string{
		MsgTypeRequest:           "request",
		MsgTypeRequestNoReturn:   "request_no_return",
		MsgTypeNotification:      "notification",
		MsgTypeResponse:          "response",
		MsgTypeError:             "error",
		MsgTypeTPRequest:         "tp_request",
		MsgTypeTPRequestNoReturn: "tp_request_no_return",
		MsgTypeTPNotification:    "tp_notification",
		MsgTypeTPResponse:        "tp_response",
		MsgTypeTPError:           "tp_error",
	}
	for mt, want := range cases {
		if got := mt.String(); got != want {
			t.Errorf("MessageType(%#x).String() = %q, want %q", uint8(mt), got, want)
		}
	}
	// Unknown values fall back to the numeric form.
	if got := MessageType(0x42).String(); got != "66" {
		t.Errorf("unknown MessageType.String() = %q, want %q", got, "66")
	}
}
