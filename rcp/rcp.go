// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

// Package rcp defines the canonical RCP types and relay.Message conversion
// per RELAY spec §15.5. RCP means the OPEN Alliance TC18 Remote Control
// Protocol Specification v0.5.1_RC as of the RELAY v2.0 MAJOR release — a
// breaking change (per REQ-RELAY-094) from the placeholder
// Zone/Command/Response/Status protocol these type names described before
// v2.0, with no compatibility shim.
package rcp

import (
	"fmt"
	"strconv"
	"time"

	relay "github.com/SoundMatt/RELAY/v2"
)

// ByteBusID addresses a single Endpoint on an RC Server. Unique only within
// the StreamID of the AVTPDU that carries it.
//
//fusa:req REQ-RELAY-040
type ByteBusID uint8

// TransactionNum correlates a request with its eventual response, scoped to
// the enclosing stream.
//
//fusa:req REQ-RELAY-040
type TransactionNum uint16

// ControlFlags are an ACF message's request-descriptor control bits.
//
//fusa:req REQ-RELAY-040
type ControlFlags uint8

// ControlFlags values.
const (
	FlagAck          ControlFlags = 1 << 7
	FlagRead         ControlFlags = 1 << 6
	FlagWrite        ControlFlags = 1 << 5
	FlagResponse     ControlFlags = 1 << 4
	FlagError        ControlFlags = 1 << 3
	FlagMoreSegments ControlFlags = 1 << 2
)

// Has reports whether all bits of want are set in f.
func (f ControlFlags) Has(want ControlFlags) bool { return f&want == want }

// Message is a decoded ACF_ABB/ACF_GBB request, response, or acknowledge.
//
//fusa:req REQ-RELAY-040
type Message struct {
	ByteBusID         ByteBusID      `json:"byte_bus_id"`
	TransactionNum    TransactionNum `json:"transaction_num,omitempty"`
	Control           ControlFlags   `json:"control"`
	ReadSizeOrSegment uint16         `json:"read_size_or_segment,omitempty"`
	Timestamp         uint64         `json:"timestamp,omitempty"`
	Body              []byte         `json:"body,omitempty"`
}

// Loan is a zero-copy payload buffer from LoaningController.Loan().
// Callers MUST call Return() when done.
//
//fusa:req REQ-RELAY-040
type Loan struct {
	Payload []byte
	release func()
}

// Return releases the loaned buffer.
func (l *Loan) Return() {
	if l.release != nil {
		l.release()
	}
}

// ErrNotFound is returned when a relay.Message.ID does not parse to a
// valid ByteBusID (0-255).
var ErrNotFound = fmt.Errorf("rcp: endpoint id not found: %w", relay.ErrNotConnected)

// EndpointIDString renders addr as the relay.Message.ID string
// FromMessage/ToMessage use to address one Endpoint.
//
//fusa:req REQ-RELAY-041
func EndpointIDString(addr ByteBusID) string {
	return strconv.Itoa(int(addr))
}

// ParseEndpointID parses a relay.Message.ID string produced by
// EndpointIDString back into a ByteBusID. Returns ErrNotFound for a
// malformed string or a value outside 0-255.
//
//fusa:req REQ-RELAY-041
func ParseEndpointID(id string) (ByteBusID, error) {
	n, err := strconv.Atoi(id)
	if err != nil || n < 0 || n > 255 {
		return 0, fmt.Errorf("rcp: endpoint id %q: %w", id, ErrNotFound)
	}
	return ByteBusID(n), nil
}

// FromMessage converts a relay.Message into an addressed request Message
// per §15.7.5 (Caller.Call()/Send() direction). msg.ID supplies ByteBusID
// (see ParseEndpointID); Meta["rcp.op"] ("read" or "write") supplies
// Control, defaulting to write when msg.Payload is non-empty, else read.
// Meta["rcp.transaction_num"] and Meta["rcp.read_size_or_segment"], each a
// decimal uint16 string, supply TransactionNum and ReadSizeOrSegment;
// absent or malformed values default to 0.
//
//fusa:req REQ-RELAY-041
func FromMessage(msg relay.Message) (Message, error) {
	addr, err := ParseEndpointID(msg.ID)
	if err != nil {
		return Message{}, err
	}
	var control ControlFlags
	switch msg.Meta["rcp.op"] {
	case "read":
		control = FlagRead
	case "write":
		control = FlagWrite
	default:
		if len(msg.Payload) > 0 {
			control = FlagWrite
		} else {
			control = FlagRead
		}
	}
	if msg.Meta["rcp.error"] == "true" {
		control |= FlagError
	}
	var txn TransactionNum
	if n, err := strconv.ParseUint(msg.Meta["rcp.transaction_num"], 10, 16); err == nil {
		txn = TransactionNum(n)
	}
	var readSize uint16
	if n, err := strconv.ParseUint(msg.Meta["rcp.read_size_or_segment"], 10, 16); err == nil {
		readSize = uint16(n)
	}
	return Message{
		ByteBusID:         addr,
		TransactionNum:    txn,
		Control:           control,
		ReadSizeOrSegment: readSize,
		Body:              msg.Payload,
	}, nil
}

// ToMessage converts a Message to a relay.Message per §15.7.5.
// Meta["rcp.op"] mirrors m.Control's FlagRead/FlagWrite bit,
// Meta["rcp.error"] mirrors its FlagError bit, and
// Meta["rcp.transaction_num"] / Meta["rcp.read_size_or_segment"] carry
// TransactionNum and ReadSizeOrSegment as decimal strings — so
// ToMessage/FromMessage round-trip exactly the fields §15.7.5 maps:
// ByteBusID, Body, TransactionNum, ReadSizeOrSegment, and the
// read/write/error bits of Control. m.Control's other defined bits —
// FlagAck, FlagResponse, FlagMoreSegments — are NOT carried and do not
// survive a round trip (a Message built with, say, FlagResponse set
// comes back with only FlagRead/FlagWrite and FlagError as sent by
// ToMessage/FromMessage); neither is m.Timestamp (the native AVTP
// presentation timestamp) — relay.Message.Timestamp always reflects
// local receipt time instead, as for every other protocol in §15.7.
// ToMessage is a single, direction-agnostic converter: it always sets
// both rcp.op and rcp.error regardless of whether m represents an
// outbound request or an inbound response; §15.7.5's request/response
// tables describe which of the two keys is semantically meaningful in
// each direction, not that the other is ever absent from Meta.
//
//fusa:req REQ-RELAY-041
func (m Message) ToMessage() relay.Message {
	op := "read"
	if m.Control.Has(FlagWrite) {
		op = "write"
	}
	return relay.Message{
		Protocol:  relay.RCP,
		ID:        EndpointIDString(m.ByteBusID),
		Payload:   m.Body,
		Timestamp: time.Now(),
		Meta: map[string]string{
			"rcp.op":                   op,
			"rcp.error":                strconv.FormatBool(m.Control.Has(FlagError)),
			"rcp.transaction_num":      strconv.FormatUint(uint64(m.TransactionNum), 10),
			"rcp.read_size_or_segment": strconv.FormatUint(uint64(m.ReadSizeOrSegment), 10),
		},
	}
}
