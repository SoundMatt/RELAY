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

// Frame is the canonical CAN / CAN-FD / CAN XL frame.
//
// The XL, SDT, VCID, AF and SEC fields carry CAN XL (ISO 11898-1:2024)
// information and are zero/false for classic and CAN-FD frames. ESI is the
// Error State Indicator, valid for CAN-FD and CAN XL frames.
//
//fusa:req REQ-RELAY-030
//fusa:req REQ-RELAY-070
//fusa:req REQ-RELAY-071
type Frame struct {
	ID   uint32 `json:"id"`
	Ext  bool   `json:"ext,omitempty"`
	RTR  bool   `json:"rtr,omitempty"`
	FD   bool   `json:"fd,omitempty"`
	BRS  bool   `json:"brs,omitempty"`
	ESI  bool   `json:"esi,omitempty"`  // Error State Indicator (CAN-FD / CAN XL)
	XL   bool   `json:"xl,omitempty"`   // CAN XL format
	SDT  uint8  `json:"sdt,omitempty"`  // SDU Type (CAN XL)
	VCID uint8  `json:"vcid,omitempty"` // Virtual CAN network ID (CAN XL)
	AF   uint32 `json:"af,omitempty"`   // Acceptance Field (CAN XL)
	SEC  bool   `json:"sec,omitempty"`  // Simple Extended Content flag (CAN XL)
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
//fusa:req REQ-RELAY-070
const (
	CANMaxDataLen   = 8
	CANFDMaxDataLen = 64
	CANXLMinDataLen = 1    // CAN XL frames carry at least one data byte
	CANXLMaxDataLen = 2048 // CAN XL payload limit
	CANMaxStdID     = 0x7FF
	CANMaxExtID     = 0x1FFFFFFF
	CANXLMaxPrioID  = 0x7FF // CAN XL Priority ID is 11-bit
)

// MaxDataLen returns the maximum data length for the given FD mode.
// It predates CAN XL; for an XL-aware limit use Frame.MaxDataLen.
//
//fusa:req REQ-RELAY-030
func MaxDataLen(fd bool) int {
	if fd {
		return CANFDMaxDataLen
	}
	return CANMaxDataLen
}

// MaxDataLen returns the maximum payload length for this frame's format:
// 2048 for CAN XL, 64 for CAN-FD, 8 for classic CAN.
//
//fusa:req REQ-RELAY-070
func (f Frame) MaxDataLen() int {
	switch {
	case f.XL:
		return CANXLMaxDataLen
	case f.FD:
		return CANFDMaxDataLen
	default:
		return CANMaxDataLen
	}
}

// ValidateFrame checks all structural constraints from §15.1.
// Returns ErrInvalidFrame on any violation; never returns ErrPayloadTooLarge.
//
//fusa:req REQ-RELAY-031
//fusa:req REQ-RELAY-072
func ValidateFrame(f Frame) error {
	// FD and XL are mutually exclusive frame formats.
	if f.FD && f.XL {
		return fmt.Errorf("can: FD and XL are mutually exclusive: %w", ErrInvalidFrame)
	}
	// ESI is only meaningful for CAN-FD or CAN XL frames.
	if f.ESI && !f.FD && !f.XL {
		return fmt.Errorf("can: ESI requires FD or XL: %w", ErrInvalidFrame)
	}

	if f.XL {
		// CAN XL uses an 11-bit Priority ID with no standard/extended distinction.
		if f.Ext {
			return fmt.Errorf("can: XL frames must not set Ext: %w", ErrInvalidFrame)
		}
		if f.ID > CANXLMaxPrioID {
			return fmt.Errorf("can: XL priority ID 0x%X exceeds max 0x%X: %w", f.ID, CANXLMaxPrioID, ErrInvalidFrame)
		}
		// CAN XL has no remote frames and always runs at the data bit rate.
		if f.RTR {
			return fmt.Errorf("can: RTR must be false for XL frames: %w", ErrInvalidFrame)
		}
		if f.BRS {
			return fmt.Errorf("can: BRS must be false for XL frames: %w", ErrInvalidFrame)
		}
		if len(f.Data) < CANXLMinDataLen {
			return fmt.Errorf("can: XL data length %d below min %d: %w", len(f.Data), CANXLMinDataLen, ErrInvalidFrame)
		}
	} else {
		// ID range (classic / FD)
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
	}

	// Data length
	maxLen := f.MaxDataLen()
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
//fusa:req REQ-RELAY-073
func (f Frame) ToMessage() relay.Message {
	meta := map[string]string{
		"can.ext": strconv.FormatBool(f.Ext),
		"can.fd":  strconv.FormatBool(f.FD),
		"can.rtr": strconv.FormatBool(f.RTR),
		"can.brs": strconv.FormatBool(f.BRS),
	}
	// CAN-FD / CAN XL fields are emitted only when set, so classic and FD
	// frames keep a stable, minimal Meta.
	if f.ESI {
		meta["can.esi"] = "true"
	}
	if f.XL {
		meta["can.xl"] = "true"
	}
	if f.SDT != 0 {
		meta["can.sdt"] = strconv.FormatUint(uint64(f.SDT), 10)
	}
	if f.VCID != 0 {
		meta["can.vcid"] = strconv.FormatUint(uint64(f.VCID), 10)
	}
	if f.AF != 0 {
		meta["can.af"] = strconv.FormatUint(uint64(f.AF), 10)
	}
	if f.SEC {
		meta["can.sec"] = "true"
	}
	return relay.Message{
		Protocol:  relay.CAN,
		ID:        strconv.FormatUint(uint64(f.ID), 10),
		Payload:   f.Data,
		Timestamp: time.Now(),
		Meta:      meta,
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
	if v := msg.Meta["can.esi"]; v != "" {
		f.ESI, _ = strconv.ParseBool(v)
	}
	if v := msg.Meta["can.xl"]; v != "" {
		f.XL, _ = strconv.ParseBool(v)
	}
	if v := msg.Meta["can.sdt"]; v != "" {
		if n, err := strconv.ParseUint(v, 10, 8); err == nil {
			f.SDT = uint8(n)
		}
	}
	if v := msg.Meta["can.vcid"]; v != "" {
		if n, err := strconv.ParseUint(v, 10, 8); err == nil {
			f.VCID = uint8(n)
		}
	}
	if v := msg.Meta["can.af"]; v != "" {
		if n, err := strconv.ParseUint(v, 10, 32); err == nil {
			f.AF = uint32(n)
		}
	}
	if v := msg.Meta["can.sec"]; v != "" {
		f.SEC, _ = strconv.ParseBool(v)
	}
	return f, nil
}
