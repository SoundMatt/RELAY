// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

// Package rcp defines the canonical RCP types and relay.Message conversion
// per RELAY spec §15.5.
package rcp

import (
	"fmt"
	"strconv"
	"time"

	relay "github.com/SoundMatt/RELAY"
)

// Zone identifies a physical zone in the vehicle.
// String() returns PascalCase names as required by §15.7.5 routing.
//
//fusa:req REQ-RELAY-040
type Zone uint8

const (
	ZoneUnknown    Zone = 0
	ZoneFrontLeft  Zone = 1
	ZoneFrontRight Zone = 2
	ZoneRearLeft   Zone = 3
	ZoneRearRight  Zone = 4
	ZoneCentral    Zone = 5
)

// String returns the PascalCase zone name required for relay.Message.ID routing.
func (z Zone) String() string {
	switch z {
	case ZoneFrontLeft:
		return "FrontLeft"
	case ZoneFrontRight:
		return "FrontRight"
	case ZoneRearLeft:
		return "RearLeft"
	case ZoneRearRight:
		return "RearRight"
	case ZoneCentral:
		return "Central"
	default:
		return "Unknown"
	}
}

// ZoneFromString parses a PascalCase zone name. Returns ZoneUnknown if unrecognised.
func ZoneFromString(s string) Zone {
	switch s {
	case "FrontLeft":
		return ZoneFrontLeft
	case "FrontRight":
		return ZoneFrontRight
	case "RearLeft":
		return ZoneRearLeft
	case "RearRight":
		return ZoneRearRight
	case "Central":
		return ZoneCentral
	default:
		return ZoneUnknown
	}
}

// Priority is the RCP command priority level.
//
//fusa:req REQ-RELAY-040
type Priority uint8

const (
	PriorityNormal   Priority = 0
	PriorityHigh     Priority = 1
	PriorityCritical Priority = 2
)

func (p Priority) String() string {
	switch p {
	case PriorityHigh:
		return "high"
	case PriorityCritical:
		return "critical"
	default:
		return "normal"
	}
}

func priorityFromString(s string) Priority {
	switch s {
	case "high":
		return PriorityHigh
	case "critical":
		return PriorityCritical
	default:
		return PriorityNormal
	}
}

// CommandType identifies the RCP command variant.
//
//fusa:req REQ-RELAY-040
type CommandType uint16

const (
	CmdNoop     CommandType = 0
	CmdSet      CommandType = 1
	CmdGet      CommandType = 2
	CmdReset    CommandType = 3
	CmdWatchdog CommandType = 4
	CmdSleep    CommandType = 5
	CmdWake     CommandType = 6
)

func (c CommandType) String() string {
	switch c {
	case CmdSet:
		return "set"
	case CmdGet:
		return "get"
	case CmdReset:
		return "reset"
	case CmdWatchdog:
		return "watchdog"
	case CmdSleep:
		return "sleep"
	case CmdWake:
		return "wake"
	default:
		return "noop"
	}
}

func cmdTypeFromString(s string) CommandType {
	switch s {
	case "set":
		return CmdSet
	case "get":
		return CmdGet
	case "reset":
		return CmdReset
	case "watchdog":
		return CmdWatchdog
	case "sleep":
		return CmdSleep
	case "wake":
		return CmdWake
	default:
		return CmdNoop
	}
}

// ResponseStatus is the RCP response status code.
//
//fusa:req REQ-RELAY-040
type ResponseStatus uint8

const (
	StatusOK      ResponseStatus = 0
	StatusError   ResponseStatus = 1
	StatusTimeout ResponseStatus = 2
	StatusBusy    ResponseStatus = 3
	StatusUnknown ResponseStatus = 4
)

// Command is a RCP control command sent to a zone controller.
//
//fusa:req REQ-RELAY-040
type Command struct {
	ID       uint32      `json:"id"`
	Zone     Zone        `json:"zone"`
	Type     CommandType `json:"type"`
	Priority Priority    `json:"priority"`
	Payload  []byte      `json:"payload,omitempty"`
}

// Response is the reply from a zone controller to a Command.
//
//fusa:req REQ-RELAY-040
type Response struct {
	CommandID uint32         `json:"command_id"`
	Zone      Zone           `json:"zone"`
	Status    ResponseStatus `json:"status"`
	Payload   []byte         `json:"payload,omitempty"`
}

// Status is a periodic status broadcast from a zone controller.
//
//fusa:req REQ-RELAY-040
type Status struct {
	Zone    Zone   `json:"zone"`
	Seq     uint32 `json:"seq"`
	Healthy bool   `json:"healthy"`
	Payload []byte `json:"payload,omitempty"`
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

// ErrInvalidZone is returned when a zone name cannot be parsed.
var ErrInvalidZone = fmt.Errorf("rcp: invalid zone name")

// ToMessage converts s to a relay.Message per §15.7.5 (Subscribe direction).
//
//fusa:req REQ-RELAY-041
func (s Status) ToMessage() relay.Message {
	return relay.Message{
		Protocol:  relay.RCP,
		ID:        s.Zone.String(),
		Payload:   s.Payload,
		Timestamp: time.Now(),
		Seq:       uint64(s.Seq),
		Meta: map[string]string{
			"rcp.healthy": strconv.FormatBool(s.Healthy),
		},
	}
}

// StatusFromMessage converts a relay.Message to a Status.
//
//fusa:req REQ-RELAY-041
func StatusFromMessage(msg relay.Message) (Status, error) {
	z := ZoneFromString(msg.ID)
	return Status{
		Zone:    z,
		Seq:     uint32(msg.Seq),
		Healthy: msg.Meta["rcp.healthy"] == "true",
		Payload: msg.Payload,
	}, nil
}

// CommandFromMessage converts a relay.Message to a Command per §15.7.5 (Call direction).
//
//fusa:req REQ-RELAY-041
func CommandFromMessage(msg relay.Message) (Command, error) {
	z := ZoneFromString(msg.ID)
	if z == ZoneUnknown && msg.ID != "Unknown" {
		return Command{}, fmt.Errorf("rcp: unknown zone %q: %w", msg.ID, ErrInvalidZone)
	}
	return Command{
		Zone:     z,
		Priority: priorityFromString(msg.Meta["rcp.priority"]),
		Type:     cmdTypeFromString(msg.Meta["rcp.cmd_type"]),
		Payload:  msg.Payload,
	}, nil
}

// ResponseToMessage converts a Response to a relay.Message per §15.7.5.
//
//fusa:req REQ-RELAY-041
func (r Response) ToMessage() relay.Message {
	return relay.Message{
		Protocol:  relay.RCP,
		ID:        r.Zone.String(),
		Payload:   r.Payload,
		Timestamp: time.Now(),
		Meta: map[string]string{
			"rcp.status": strconv.FormatUint(uint64(r.Status), 10),
		},
	}
}
