// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

// Package someip defines the canonical SOME/IP message types and relay.Message
// conversion per RELAY spec §15.6.
package someip

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	relay "github.com/SoundMatt/RELAY"
)

// SOMEIPProtocolVersion is the only valid SOME/IP protocol version (§15.6).
//
//fusa:req REQ-RELAY-042
const SOMEIPProtocolVersion uint8 = 0x01

// Type aliases for SOME/IP identifiers (§15.6).
type (
	ServiceID  = uint16
	MethodID   = uint16
	ClientID   = uint16
	SessionID  = uint16
	InstanceID = uint16
	EventID    = uint16
)

// MessageType is the SOME/IP message type byte.
//
//fusa:req REQ-RELAY-042
type MessageType uint8

const (
	MsgTypeRequest           MessageType = 0x00
	MsgTypeRequestNoReturn   MessageType = 0x01
	MsgTypeNotification      MessageType = 0x02
	MsgTypeResponse          MessageType = 0x80
	MsgTypeError             MessageType = 0x81
	MsgTypeTPRequest         MessageType = 0x20
	MsgTypeTPRequestNoReturn MessageType = 0x21
	MsgTypeTPNotification    MessageType = 0x22
	MsgTypeTPResponse        MessageType = 0xA0
	MsgTypeTPError           MessageType = 0xA1
)

func (m MessageType) String() string {
	switch m {
	case MsgTypeRequest:
		return "request"
	case MsgTypeRequestNoReturn:
		return "request_no_return"
	case MsgTypeNotification:
		return "notification"
	case MsgTypeResponse:
		return "response"
	case MsgTypeError:
		return "error"
	case MsgTypeTPRequest:
		return "tp_request"
	case MsgTypeTPRequestNoReturn:
		return "tp_request_no_return"
	case MsgTypeTPNotification:
		return "tp_notification"
	case MsgTypeTPResponse:
		return "tp_response"
	case MsgTypeTPError:
		return "tp_error"
	default:
		return strconv.FormatUint(uint64(m), 10)
	}
}

// ReturnCode is the SOME/IP return code byte.
//
//fusa:req REQ-RELAY-042
type ReturnCode uint8

const (
	RetOK                    ReturnCode = 0x00
	RetNotOK                 ReturnCode = 0x01
	RetUnknownService        ReturnCode = 0x02
	RetUnknownMethod         ReturnCode = 0x03
	RetNotReady              ReturnCode = 0x04
	RetNotReachable          ReturnCode = 0x05
	RetTimeout               ReturnCode = 0x06
	RetWrongProtocolVersion  ReturnCode = 0x07
	RetWrongInterfaceVersion ReturnCode = 0x08
	RetMalformedMessage      ReturnCode = 0x09
	RetWrongMessageType      ReturnCode = 0x0A
)

// MethodHandler is the signature for a SOME/IP server method handler.
// Returning a non-nil error causes the server to respond with MsgTypeError.
type MethodHandler func(ctx interface{}, req Message) ([]byte, error)

// Message is the canonical SOME/IP message.
// ProtocolVersion MUST equal SOMEIPProtocolVersion on both send and receive.
//
//fusa:req REQ-RELAY-042
type Message struct {
	ServiceID        uint16      `json:"service_id"`
	MethodID         uint16      `json:"method_id"`
	ClientID         uint16      `json:"client_id"`
	SessionID        uint16      `json:"session_id"`
	ProtocolVersion  uint8       `json:"protocol_version"`
	InterfaceVersion uint8       `json:"interface_version"`
	MessageType      MessageType `json:"message_type"`
	ReturnCode       ReturnCode  `json:"return_code"`
	Payload          []byte      `json:"payload,omitempty"`
}

// ErrInvalidID is returned by FromMessage when msg.ID cannot be parsed.
var ErrInvalidID = fmt.Errorf("someip: invalid service/method ID format")

// ErrWrongProtocolVersion is returned when ProtocolVersion ≠ SOMEIPProtocolVersion.
var ErrWrongProtocolVersion = fmt.Errorf("someip: wrong protocol version")

// Validate checks that ProtocolVersion is 0x01.
func (m Message) Validate() error {
	if m.ProtocolVersion != SOMEIPProtocolVersion {
		return fmt.Errorf("someip: ProtocolVersion 0x%02X != 0x01: %w", m.ProtocolVersion, ErrWrongProtocolVersion)
	}
	return nil
}

// ToMessage converts m to a relay.Message per §15.7.6.
// The conversion is lossless: every SOME/IP header field is carried either in
// the ID ("serviceID/methodID") or in Meta. someip.msg_type carries the numeric
// message type for round-trip fidelity; someip.msg_type_name carries the
// human-readable label for diagnostics.
//
//fusa:req REQ-RELAY-043
//fusa:req REQ-RELAY-057
func (m Message) ToMessage() relay.Message {
	return relay.Message{
		Protocol:  relay.SOMEIP,
		ID:        fmt.Sprintf("%d/%d", m.ServiceID, m.MethodID),
		Payload:   m.Payload,
		Timestamp: time.Now(),
		Meta: map[string]string{
			"someip.client_id":         strconv.FormatUint(uint64(m.ClientID), 10),
			"someip.session_id":        strconv.FormatUint(uint64(m.SessionID), 10),
			"someip.msg_type":          strconv.FormatUint(uint64(m.MessageType), 10),
			"someip.msg_type_name":     m.MessageType.String(),
			"someip.return_code":       strconv.FormatUint(uint64(m.ReturnCode), 10),
			"someip.interface_version": strconv.FormatUint(uint64(m.InterfaceVersion), 10),
		},
	}
}

// FromMessage converts a relay.Message to a Message per §15.7.6.
// Returns ErrInvalidID if msg.ID is not in "serviceID/methodID" decimal form.
//
//fusa:req REQ-RELAY-043
func FromMessage(msg relay.Message) (Message, error) {
	parts := strings.SplitN(msg.ID, "/", 2)
	if len(parts) != 2 {
		return Message{}, fmt.Errorf("someip: ID %q must be \"svcID/methodID\": %w", msg.ID, ErrInvalidID)
	}
	svc, err1 := strconv.ParseUint(parts[0], 10, 16)
	meth, err2 := strconv.ParseUint(parts[1], 10, 16)
	if err1 != nil || err2 != nil {
		return Message{}, fmt.Errorf("someip: ID %q: %w", msg.ID, ErrInvalidID)
	}
	m := Message{
		ServiceID:       uint16(svc),
		MethodID:        uint16(meth),
		Payload:         msg.Payload,
		ProtocolVersion: SOMEIPProtocolVersion,
	}
	if cid := msg.Meta["someip.client_id"]; cid != "" {
		v, _ := strconv.ParseUint(cid, 10, 16)
		m.ClientID = uint16(v)
	}
	if sid := msg.Meta["someip.session_id"]; sid != "" {
		v, _ := strconv.ParseUint(sid, 10, 16)
		m.SessionID = uint16(v)
	}
	if mt := msg.Meta["someip.msg_type"]; mt != "" {
		v, _ := strconv.ParseUint(mt, 10, 8)
		m.MessageType = MessageType(v)
	}
	if rc := msg.Meta["someip.return_code"]; rc != "" {
		v, _ := strconv.ParseUint(rc, 10, 8)
		m.ReturnCode = ReturnCode(v)
	}
	if iv := msg.Meta["someip.interface_version"]; iv != "" {
		v, _ := strconv.ParseUint(iv, 10, 8)
		m.InterfaceVersion = uint8(v)
	}
	return m, nil
}
