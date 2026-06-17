// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

// Package relay defines the shared specification types for the SoundMatt
// embedded network protocol ecosystem. All six protocol implementations
// (CAN, DDS, LIN, MQTT, RCP, SOME/IP) build against these types.
package relay

import (
	"fmt"
	"time"
)

// Protocol identifies a network protocol implementation. Zero is reserved.
//
//fusa:req REQ-RELAY-001
type Protocol int

// Protocol constants, one per supported network protocol (§3).
//
//fusa:req REQ-RELAY-002
const (
	CAN    Protocol = 1
	DDS    Protocol = 2
	LIN    Protocol = 3
	MQTT   Protocol = 4
	RCP    Protocol = 5
	SOMEIP Protocol = 6
)

// String returns the canonical upper-case name of the protocol.
//
//fusa:req REQ-RELAY-003
func (p Protocol) String() string {
	switch p {
	case CAN:
		return "CAN"
	case DDS:
		return "DDS"
	case LIN:
		return "LIN"
	case MQTT:
		return "MQTT"
	case RCP:
		return "RCP"
	case SOMEIP:
		return "SOMEIP"
	default:
		return "unknown"
	}
}

// Version is a semantic version triple.
//
//fusa:req REQ-RELAY-004
type Version struct {
	Major int `json:"major"`
	Minor int `json:"minor"`
	Patch int `json:"patch"`
}

// String returns the version in "MAJOR.MINOR.PATCH" form.
//
//fusa:req REQ-RELAY-005
func (v Version) String() string {
	return fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
}

// Message is the universal cross-protocol envelope used by relay.Node,
// relay.Caller, and observability tooling. It is not a wire format.
//
// ID carries the protocol-specific routing key (see spec §4.2).
// Meta carries optional protocol-specific fields (see spec §4.3).
// Seq and Meta are omitted from JSON when zero/nil (REQ-RELAY-007).
//
//fusa:req REQ-RELAY-006
//fusa:req REQ-RELAY-007
type Message struct {
	Protocol  Protocol          `json:"protocol"`
	Version   Version           `json:"version"`
	ID        string            `json:"id"`
	Payload   []byte            `json:"payload"`
	Timestamp time.Time         `json:"timestamp"`
	Seq       uint64            `json:"seq,omitempty"`
	Meta      map[string]string `json:"meta,omitempty"`
}
