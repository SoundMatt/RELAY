// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package relay_test

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	relay "github.com/SoundMatt/RELAY"
	"github.com/SoundMatt/RELAY/can"
	"github.com/SoundMatt/RELAY/dds"
	"github.com/SoundMatt/RELAY/lin"
	"github.com/SoundMatt/RELAY/mqtt"
	"github.com/SoundMatt/RELAY/rcp"
	"github.com/SoundMatt/RELAY/someip"
)

// vector is the on-disk golden reference vector format (spec/vectors/*.json).
type vector struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Type        string          `json:"type"`
	Value       json.RawMessage `json:"value"`
	Message     json.RawMessage `json:"message"`
}

// TestGoldenVectorsRoundTrip verifies that every committed golden vector
// (1) marshals from its canonical Value to exactly the stored relay.Message
// (timestamp excluded), and (2) round-trips back through FromMessage to the
// original canonical Value. This keeps spec/vectors/ honest against the code.
//
//fusa:test REQ-RELAY-057
func TestGoldenVectorsRoundTrip(t *testing.T) {
	paths, err := filepath.Glob("spec/vectors/*.json")
	if err != nil {
		t.Fatalf("glob vectors: %v", err)
	}
	if len(paths) == 0 {
		t.Fatal("no golden vectors found in spec/vectors/")
	}

	for _, p := range paths {
		p := p
		t.Run(filepath.Base(p), func(t *testing.T) {
			data, err := os.ReadFile(p)
			if err != nil {
				t.Fatalf("read %s: %v", p, err)
			}
			var v vector
			if err := json.Unmarshal(data, &v); err != nil {
				t.Fatalf("unmarshal vector %s: %v", p, err)
			}

			var want relay.Message
			if err := json.Unmarshal(v.Message, &want); err != nil {
				t.Fatalf("unmarshal expected message: %v", err)
			}

			got, back := convert(t, v)
			// Timestamps are non-deterministic for types that stamp time.Now();
			// normalise before comparing.
			got.Timestamp = want.Timestamp
			if !reflect.DeepEqual(got, want) {
				gj, _ := json.MarshalIndent(got, "", "  ")
				wj, _ := json.MarshalIndent(want, "", "  ")
				t.Errorf("ToMessage mismatch for %s\n got: %s\nwant: %s", v.Name, gj, wj)
			}

			// Round-trip: FromMessage(message) must reproduce the canonical Value.
			var wantValue, gotValue interface{}
			if err := json.Unmarshal(v.Value, &wantValue); err != nil {
				t.Fatalf("unmarshal value: %v", err)
			}
			bj, _ := json.Marshal(back)
			if err := json.Unmarshal(bj, &gotValue); err != nil {
				t.Fatalf("re-unmarshal round-tripped value: %v", err)
			}
			if !reflect.DeepEqual(gotValue, wantValue) {
				t.Errorf("FromMessage round-trip mismatch for %s\n got: %s\nwant: %s", v.Name, bj, v.Value)
			}
		})
	}
}

// convert decodes the vector's canonical Value, calls ToMessage, and returns
// both the produced relay.Message and the value round-tripped via FromMessage.
func convert(t *testing.T, v vector) (msg relay.Message, back interface{}) {
	t.Helper()
	switch v.Type {
	case "can.Frame":
		var f can.Frame
		mustUnmarshal(t, v.Value, &f)
		b, err := can.FromMessage(f.ToMessage())
		if err != nil {
			t.Fatalf("can.FromMessage: %v", err)
		}
		return f.ToMessage(), b
	case "dds.Sample":
		var s dds.Sample
		mustUnmarshal(t, v.Value, &s)
		b, err := dds.FromMessage(s.ToMessage())
		if err != nil {
			t.Fatalf("dds.FromMessage: %v", err)
		}
		return s.ToMessage(), b
	case "lin.Frame":
		var f lin.Frame
		mustUnmarshal(t, v.Value, &f)
		b, err := lin.FromMessage(f.ToMessage())
		if err != nil {
			t.Fatalf("lin.FromMessage: %v", err)
		}
		return f.ToMessage(), b
	case "mqtt.Message":
		var m mqtt.Message
		mustUnmarshal(t, v.Value, &m)
		b, err := mqtt.FromMessage(m.ToMessage())
		if err != nil {
			t.Fatalf("mqtt.FromMessage: %v", err)
		}
		return m.ToMessage(), b
	case "rcp.Status":
		var s rcp.Status
		mustUnmarshal(t, v.Value, &s)
		b, err := rcp.StatusFromMessage(s.ToMessage())
		if err != nil {
			t.Fatalf("rcp.StatusFromMessage: %v", err)
		}
		return s.ToMessage(), b
	case "someip.Message":
		var m someip.Message
		mustUnmarshal(t, v.Value, &m)
		b, err := someip.FromMessage(m.ToMessage())
		if err != nil {
			t.Fatalf("someip.FromMessage: %v", err)
		}
		return m.ToMessage(), b
	default:
		t.Fatalf("unknown vector type %q", v.Type)
		return relay.Message{}, nil
	}
}

func mustUnmarshal(t *testing.T, data []byte, v interface{}) {
	t.Helper()
	if err := json.Unmarshal(data, v); err != nil {
		t.Fatalf("unmarshal into %T: %v", v, err)
	}
}

// errVector is an error-condition golden vector (spec/vectors/errors/*.json):
// a canonical value that MUST be rejected by its type's validator with a
// specific named error sentinel.
type errVector struct {
	Name  string          `json:"name"`
	Type  string          `json:"type"`
	Kind  string          `json:"kind"`
	Value json.RawMessage `json:"value"`
	Error string          `json:"error"`
}

// TestErrorVectors verifies that every error-condition vector is rejected by
// the relevant validator with the named sentinel error.
//
//fusa:test REQ-RELAY-057
func TestErrorVectors(t *testing.T) {
	paths, err := filepath.Glob("spec/vectors/errors/*.json")
	if err != nil {
		t.Fatalf("glob error vectors: %v", err)
	}
	if len(paths) == 0 {
		t.Fatal("no error vectors found in spec/vectors/errors/")
	}

	for _, p := range paths {
		p := p
		t.Run(filepath.Base(p), func(t *testing.T) {
			data, err := os.ReadFile(p)
			if err != nil {
				t.Fatalf("read %s: %v", p, err)
			}
			var ev errVector
			if err := json.Unmarshal(data, &ev); err != nil {
				t.Fatalf("unmarshal error vector: %v", err)
			}
			gotErr, sentinel := validateErr(t, ev)
			if gotErr == nil {
				t.Fatalf("%s: expected error %s, got nil", ev.Name, ev.Error)
			}
			if !errors.Is(gotErr, sentinel) {
				t.Errorf("%s: error %v does not wrap expected sentinel %s", ev.Name, gotErr, ev.Error)
			}
		})
	}
}

// validateErr decodes an error vector's value and runs the appropriate
// validator, returning the produced error and the sentinel it must wrap.
func validateErr(t *testing.T, ev errVector) (err error, sentinel error) {
	t.Helper()
	switch ev.Type {
	case "can.Frame":
		var f can.Frame
		mustUnmarshal(t, ev.Value, &f)
		return can.ValidateFrame(f), can.ErrInvalidFrame
	case "lin.Frame":
		var f lin.Frame
		mustUnmarshal(t, ev.Value, &f)
		return lin.ValidateFrame(f), lin.ErrInvalidFrame
	case "someip.Message":
		var m someip.Message
		mustUnmarshal(t, ev.Value, &m)
		return m.Validate(), someip.ErrWrongProtocolVersion
	case "dds.Domain":
		var d dds.Domain
		mustUnmarshal(t, ev.Value, &d)
		return dds.ValidateDomain(d), dds.ErrDomainOutOfRange
	default:
		t.Fatalf("unknown error vector type %q", ev.Type)
		return nil, nil
	}
}

// ensure time import is used even if all types stamp deterministically.
var _ = time.Time{}
