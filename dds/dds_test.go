// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package dds

import (
	"encoding/hex"
	"testing"
	"time"

	relay "github.com/SoundMatt/RELAY"
)

//fusa:test REQ-RELAY-033
func TestDefaultQoS(t *testing.T) {
	q := DefaultQoS()
	if q.HistoryDepth != 1 {
		t.Errorf("DefaultQoS.HistoryDepth = %d, want 1", q.HistoryDepth)
	}
	if q.Reliability != BestEffort {
		t.Errorf("DefaultQoS.Reliability = %v, want BestEffort", q.Reliability)
	}
}

//fusa:test REQ-RELAY-033
func TestValidateDomain(t *testing.T) {
	if err := ValidateDomain(0); err != nil {
		t.Errorf("domain 0: %v", err)
	}
	if err := ValidateDomain(232); err != nil {
		t.Errorf("domain 232: %v", err)
	}
	if err := ValidateDomain(233); err == nil {
		t.Error("expected error for domain 233")
	}
	if err := ValidateDomain(-1); err == nil {
		t.Error("expected error for domain -1")
	}
}

//fusa:test REQ-RELAY-034
func TestSampleRoundTrip(t *testing.T) {
	var guid GUID
	copy(guid[:], []byte("0123456789abcdef"))
	orig := Sample{
		Topic:          "vehicle/speed",
		Payload:        []byte{0xDE, 0xAD},
		Timestamp:      time.Unix(1234567890, 0).UTC(),
		SequenceNumber: 42,
		WriterGUID:     guid,
	}

	msg := orig.ToMessage()
	if msg.Protocol != relay.DDS {
		t.Errorf("Protocol = %v, want DDS", msg.Protocol)
	}
	if msg.ID != "vehicle/speed" {
		t.Errorf("ID = %q", msg.ID)
	}
	if msg.Seq != 42 {
		t.Errorf("Seq = %d, want 42", msg.Seq)
	}
	wantGUID := hex.EncodeToString(guid[:])
	if msg.Meta["dds.writer_guid"] != wantGUID {
		t.Errorf("writer_guid = %q, want %q", msg.Meta["dds.writer_guid"], wantGUID)
	}

	got, err := FromMessage(msg)
	if err != nil {
		t.Fatalf("FromMessage: %v", err)
	}
	if got.Topic != orig.Topic || got.SequenceNumber != orig.SequenceNumber {
		t.Errorf("round-trip mismatch: topic=%q seq=%d", got.Topic, got.SequenceNumber)
	}
	if got.WriterGUID != orig.WriterGUID {
		t.Errorf("WriterGUID mismatch")
	}
}
