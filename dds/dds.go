// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

// Package dds defines the canonical DDS types and relay.Message conversion
// per RELAY spec §15.2.
package dds

import (
	"encoding/hex"
	"fmt"
	"time"

	relay "github.com/SoundMatt/RELAY"
)

// GUID is a 16-byte DDS writer GUID.
//
//fusa:req REQ-RELAY-033
type GUID [16]byte

// ReliabilityKind controls DDS reliability QoS.
type ReliabilityKind int

const (
	BestEffort ReliabilityKind = 0
	Reliable   ReliabilityKind = 1
)

// DurabilityKind controls DDS durability QoS.
type DurabilityKind int

const (
	Volatile       DurabilityKind = 0
	TransientLocal DurabilityKind = 1
)

// QoS holds all DDS endpoint quality-of-service parameters.
// Defaults: Reliability=BestEffort, Durability=Volatile, HistoryDepth=1.
//
//fusa:req REQ-RELAY-033
type QoS struct {
	Reliability       ReliabilityKind `json:"reliability"`
	Durability        DurabilityKind  `json:"durability"`
	HistoryDepth      int             `json:"history_depth"`
	Deadline          time.Duration   `json:"deadline"`
	MaxSampleSize     int             `json:"max_sample_size"`
	TransportPriority int             `json:"transport_priority"`
	LatencyBudget     time.Duration   `json:"latency_budget"`
	Lifespan          time.Duration   `json:"lifespan"`
	PublishPeriod     time.Duration   `json:"publish_period"`
}

// DefaultQoS returns a QoS with RELAY-specified defaults.
func DefaultQoS() QoS { return QoS{HistoryDepth: 1} }

// Sample is a DDS data sample with metadata.
//
//fusa:req REQ-RELAY-033
type Sample struct {
	Topic          string    `json:"topic"`
	Payload        []byte    `json:"payload"`
	Timestamp      time.Time `json:"timestamp"`
	SequenceNumber uint64    `json:"seq"`
	WriterGUID     GUID      `json:"writer_guid"`
}

// Domain is a DDS domain ID. MUST be in range 0–232 inclusive.
type Domain int

// ToMessage converts s to a relay.Message per §15.7.2.
//
//fusa:req REQ-RELAY-034
func (s Sample) ToMessage() relay.Message {
	return relay.Message{
		Protocol:  relay.DDS,
		ID:        s.Topic,
		Payload:   s.Payload,
		Timestamp: s.Timestamp,
		Seq:       s.SequenceNumber,
		Meta: map[string]string{
			"dds.writer_guid": hex.EncodeToString(s.WriterGUID[:]),
		},
	}
}

// FromMessage converts a relay.Message to a Sample per §15.7.2.
//
//fusa:req REQ-RELAY-034
func FromMessage(msg relay.Message) (Sample, error) {
	s := Sample{
		Topic:          msg.ID,
		Payload:        msg.Payload,
		Timestamp:      msg.Timestamp,
		SequenceNumber: msg.Seq,
	}
	if g := msg.Meta["dds.writer_guid"]; g != "" {
		b, err := hex.DecodeString(g)
		if err == nil && len(b) == 16 {
			copy(s.WriterGUID[:], b)
		}
	}
	return s, nil
}

// ErrDomainOutOfRange is returned when a Domain value is outside 0–232.
var ErrDomainOutOfRange = fmt.Errorf("dds: domain out of range 0–232")

// ValidateDomain returns ErrDomainOutOfRange if d is not in 0–232.
func ValidateDomain(d Domain) error {
	if d < 0 || d > 232 {
		return ErrDomainOutOfRange
	}
	return nil
}
