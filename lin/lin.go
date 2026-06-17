// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

// Package lin defines canonical LIN frame types, validation helpers, and
// relay.Message conversion per RELAY spec §15.3.
package lin

import (
	"fmt"
	"strconv"
	"time"

	relay "github.com/SoundMatt/RELAY"
)

// ChecksumType selects the LIN checksum algorithm.
//
//fusa:req REQ-RELAY-035
type ChecksumType int

// ChecksumType values.
const (
	ClassicChecksum  ChecksumType = 0
	EnhancedChecksum ChecksumType = 1
)

// Frame is the canonical LIN frame.
//
//fusa:req REQ-RELAY-035
type Frame struct {
	ID           uint8        `json:"id"`
	Data         []byte       `json:"data"`
	Checksum     uint8        `json:"checksum"`
	ChecksumType ChecksumType `json:"checksum_type"`
}

// Filter matches LIN frames by ID, or all frames when All is true.
//
//fusa:req REQ-RELAY-035
type Filter struct {
	ID  uint8 `json:"id"`
	All bool  `json:"all"`
}

// Matches returns true if fr matches the filter.
func (f Filter) Matches(fr Frame) bool { return f.All || fr.ID == f.ID }

// ScheduleEntry is one slot in a LIN master schedule table.
//
//fusa:req REQ-RELAY-035
type ScheduleEntry struct {
	ID      uint8  `json:"id"`
	DelayMs uint32 `json:"delay_ms"`
}

// LIN ID and data length limits (§15.3).
//
//fusa:req REQ-RELAY-035
const (
	LINMaxDataLen     = 8
	LINMaxID          = 0x3F
	LINDiagRequestID  = 0x3C
	LINDiagResponseID = 0x3D
)

// ErrInvalidFrame is returned by ValidateFrame for structural violations.
var ErrInvalidFrame = fmt.Errorf("lin: invalid frame")

// ValidateFrame checks all structural constraints from §15.3.
//
//fusa:req REQ-RELAY-036
func ValidateFrame(f Frame) error {
	if f.ID > LINMaxID {
		return fmt.Errorf("lin: ID 0x%02X exceeds max 0x%02X: %w", f.ID, LINMaxID, ErrInvalidFrame)
	}
	if len(f.Data) < 1 || len(f.Data) > LINMaxDataLen {
		return fmt.Errorf("lin: data length %d not in 1–8: %w", len(f.Data), ErrInvalidFrame)
	}
	// Diagnostic frames must use ClassicChecksum.
	if (f.ID == LINDiagRequestID || f.ID == LINDiagResponseID) && f.ChecksumType != ClassicChecksum {
		return fmt.Errorf("lin: diagnostic frame 0x%02X requires ClassicChecksum: %w", f.ID, ErrInvalidFrame)
	}
	return nil
}

// ProtectID returns the LIN protected ID (PID) for a raw frame ID.
// The raw ID must be in range 0x00–0x3F.
//
//fusa:req REQ-RELAY-036
func ProtectID(id uint8) uint8 {
	id &= 0x3F
	p0 := (id>>0 ^ id>>1 ^ id>>2 ^ id>>4) & 1
	p1 := ^((id >> 1) ^ (id >> 3) ^ (id >> 4) ^ (id >> 5)) & 1
	return id | (p0 << 6) | (p1 << 7)
}

// VerifyPID checks parity bits in a LIN PID and returns the raw frame ID.
// Returns ErrInvalidFrame if parity bits are wrong.
//
//fusa:req REQ-RELAY-036
func VerifyPID(pid uint8) (uint8, error) {
	id := pid & 0x3F
	if ProtectID(id) != pid {
		return 0, fmt.Errorf("lin: PID 0x%02X parity error: %w", pid, ErrInvalidFrame)
	}
	return id, nil
}

// CalcChecksum computes the LIN checksum for the given PID and data.
//
//fusa:req REQ-RELAY-036
func CalcChecksum(pid uint8, data []byte, ct ChecksumType) uint8 {
	var sum uint32
	if ct == EnhancedChecksum {
		sum += uint32(pid)
	}
	for _, b := range data {
		sum += uint32(b)
		if sum > 0xFF {
			sum -= 0xFF
		}
	}
	return uint8(^uint8(sum))
}

// ToMessage converts f to a relay.Message per §15.7.3.
//
//fusa:req REQ-RELAY-037
func (f Frame) ToMessage() relay.Message {
	ct := "classic"
	if f.ChecksumType == EnhancedChecksum {
		ct = "enhanced"
	}
	return relay.Message{
		Protocol:  relay.LIN,
		ID:        strconv.FormatUint(uint64(f.ID), 10),
		Payload:   f.Data,
		Timestamp: time.Now(),
		Meta: map[string]string{
			"lin.checksum_type": ct,
			"lin.checksum":      strconv.FormatUint(uint64(f.Checksum), 10),
		},
	}
}

// FromMessage converts a relay.Message to a Frame per §15.7.3.
// Returns ErrInvalidFrame if msg.ID is not a valid uint8 in 0–63.
//
//fusa:req REQ-RELAY-037
func FromMessage(msg relay.Message) (Frame, error) {
	id64, err := strconv.ParseUint(msg.ID, 10, 8)
	if err != nil || id64 > LINMaxID {
		return Frame{}, fmt.Errorf("lin: invalid frame ID %q: %w", msg.ID, ErrInvalidFrame)
	}
	f := Frame{
		ID:   uint8(id64),
		Data: msg.Payload,
	}
	if ct := msg.Meta["lin.checksum_type"]; ct == "enhanced" {
		f.ChecksumType = EnhancedChecksum
	}
	if cs := msg.Meta["lin.checksum"]; cs != "" {
		v, _ := strconv.ParseUint(cs, 10, 8)
		f.Checksum = uint8(v)
	}
	return f, nil
}
