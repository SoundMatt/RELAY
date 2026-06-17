// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

// Package can defines the canonical CAN/CAN-FD frame types, validation, and
// relay.Message conversion per RELAY spec §15.1.
package can

import (
	"fmt"
	"strconv"
	"time"

	relay "github.com/SoundMatt/RELAY"
)

// Frame is the canonical CAN / CAN-FD frame.
//
//fusa:req REQ-RELAY-030
type Frame struct {
	ID   uint32 `json:"id"`
	Ext  bool   `json:"ext,omitempty"`
	RTR  bool   `json:"rtr,omitempty"`
	FD   bool   `json:"fd,omitempty"`
	BRS  bool   `json:"brs,omitempty"`
	Data []byte `json:"data"`
}

// Filter matches CAN frames by ID and mask.
//
//fusa:req REQ-RELAY-030
type Filter struct {
	ID   uint32 `json:"id"`
	Mask uint32 `json:"mask"`
}

// Matches returns true if fr.ID matches the filter (fr.ID & Mask == ID & Mask).
func (f Filter) Matches(fr Frame) bool { return fr.ID&f.Mask == f.ID&f.Mask }

// LoanedFrame is a zero-copy frame from LoaningBus.Loan().
// Callers MUST call Return() when done; the release function is unexported to
// prevent bypassing Return().
//
//fusa:req REQ-RELAY-030
type LoanedFrame struct {
	Frame
	release func()
}

// Return releases the loaned buffer back to the implementation.
func (f *LoanedFrame) Return() {
	if f.release != nil {
		f.release()
	}
}

// CAN frame size and ID limits (§15.1).
//
//fusa:req REQ-RELAY-030
const (
	CANMaxDataLen   = 8
	CANFDMaxDataLen = 64
	CANMaxStdID     = 0x7FF
	CANMaxExtID     = 0x1FFFFFFF
)

// MaxDataLen returns the maximum data length for the given FD mode.
//
//fusa:req REQ-RELAY-030
func MaxDataLen(fd bool) int {
	if fd {
		return CANFDMaxDataLen
	}
	return CANMaxDataLen
}

// ValidateFrame checks all structural constraints from §15.1.
// Returns ErrInvalidFrame on any violation; never returns ErrPayloadTooLarge.
//
//fusa:req REQ-RELAY-031
func ValidateFrame(f Frame) error {
	// ID range
	if f.Ext {
		if f.ID > CANMaxExtID {
			return fmt.Errorf("can: extended ID 0x%X exceeds max 0x%X: %w", f.ID, CANMaxExtID, ErrInvalidFrame)
		}
	} else {
		if f.ID > CANMaxStdID {
			return fmt.Errorf("can: standard ID 0x%X exceeds max 0x%X: %w", f.ID, CANMaxStdID, ErrInvalidFrame)
		}
	}
	// BRS only valid for FD
	if f.BRS && !f.FD {
		return fmt.Errorf("can: BRS requires FD: %w", ErrInvalidFrame)
	}
	// RTR is invalid when FD is set
	if f.RTR && f.FD {
		return fmt.Errorf("can: RTR must be false when FD is true: %w", ErrInvalidFrame)
	}
	// Data length
	maxLen := MaxDataLen(f.FD)
	if len(f.Data) > maxLen {
		return fmt.Errorf("can: data length %d exceeds max %d: %w", len(f.Data), maxLen, ErrInvalidFrame)
	}
	return nil
}

// ErrInvalidFrame is returned by ValidateFrame for structural violations.
// It is distinct from relay.ErrPayloadTooLarge (§5.4).
var ErrInvalidFrame = fmt.Errorf("can: invalid frame")

// ToMessage converts f to a relay.Message per §15.7.1.
//
//fusa:req REQ-RELAY-032
func (f Frame) ToMessage() relay.Message {
	return relay.Message{
		Protocol:  relay.CAN,
		ID:        strconv.FormatUint(uint64(f.ID), 10),
		Payload:   f.Data,
		Timestamp: time.Now(),
		Meta: map[string]string{
			"can.ext": strconv.FormatBool(f.Ext),
			"can.fd":  strconv.FormatBool(f.FD),
			"can.rtr": strconv.FormatBool(f.RTR),
			"can.brs": strconv.FormatBool(f.BRS),
		},
	}
}

// FromMessage converts a relay.Message to a Frame per §15.7.1.
// Returns ErrInvalidFrame if msg.ID is not a valid uint32.
//
//fusa:req REQ-RELAY-032
func FromMessage(msg relay.Message) (Frame, error) {
	id, err := strconv.ParseUint(msg.ID, 10, 32)
	if err != nil {
		return Frame{}, fmt.Errorf("can: invalid frame ID %q: %w", msg.ID, ErrInvalidFrame)
	}
	f := Frame{
		ID:   uint32(id),
		Data: msg.Payload,
	}
	if v := msg.Meta["can.ext"]; v != "" {
		f.Ext, _ = strconv.ParseBool(v)
	}
	if v := msg.Meta["can.fd"]; v != "" {
		f.FD, _ = strconv.ParseBool(v)
	}
	if v := msg.Meta["can.rtr"]; v != "" {
		f.RTR, _ = strconv.ParseBool(v)
	}
	if v := msg.Meta["can.brs"]; v != "" {
		f.BRS, _ = strconv.ParseBool(v)
	}
	return f, nil
}
