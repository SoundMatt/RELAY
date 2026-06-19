// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	relay "github.com/SoundMatt/RELAY"
)

//fusa:test REQ-RELAY-082
func TestConvertGoldenVectors(t *testing.T) {
	names, err := relay.VectorNames()
	if err != nil {
		t.Fatalf("VectorNames: %v", err)
	}
	tested := 0
	for _, name := range names {
		raw, err := relay.Vector(name)
		if err != nil {
			t.Fatal(err)
		}
		var v struct {
			Type  string          `json:"type"`
			Value json.RawMessage `json:"value"`
		}
		if json.Unmarshal(raw, &v) != nil {
			continue
		}
		proto, ok := typeProtocol[v.Type]
		if !ok {
			continue
		}
		tested++
		var out, errb bytes.Buffer
		err = runConvert(bytes.NewReader(v.Value), &out, &errb, []string{"--protocol", proto})
		if err != nil {
			t.Errorf("convert %s (%s): %v (%s)", name, proto, err, errb.String())
			continue
		}
		var msg relay.Message
		if err := json.Unmarshal(out.Bytes(), &msg); err != nil {
			t.Errorf("convert %s: output not a relay.Message: %v", name, err)
		}
		if msg.Protocol == 0 {
			t.Errorf("convert %s: zero protocol", name)
		}
	}
	if tested == 0 {
		t.Fatal("no golden vectors exercised convert")
	}
}

//fusa:test REQ-RELAY-082
func TestConvertErrors(t *testing.T) {
	cases := []struct {
		name string
		in   string
		args []string
		want int // expected exitCode
	}{
		{"missing protocol", `{}`, nil, 2},
		{"bad format", `{}`, []string{"--protocol", "CAN", "--format", "yaml"}, 2},
		{"unknown protocol", `{}`, []string{"--protocol", "FOO"}, 1},
		{"invalid json", `not-json`, []string{"--protocol", "CAN"}, 1},
		{"validation failure", `{"id":4096,"xl":true,"data":"AQ=="}`, []string{"--protocol", "CAN"}, 1},
	}
	for _, tc := range cases {
		var out, errb bytes.Buffer
		err := runConvert(strings.NewReader(tc.in), &out, &errb, tc.args)
		var code exitCode
		if !errors.As(err, &code) || int(code) != tc.want {
			t.Errorf("%s: err=%v, want exitCode(%d)", tc.name, err, tc.want)
		}
	}
}

//fusa:test REQ-RELAY-082
func TestReferenceConvertProtocols(t *testing.T) {
	// One minimal valid value per protocol must convert without error.
	cases := map[string]string{
		"CAN":    `{"id":1,"data":"AQ=="}`,
		"DDS":    `{"topic":"t","payload":"AQ==","seq":1,"writer_guid":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0]}`,
		"LIN":    `{"id":1,"data":"AQ==","checksum_type":0}`,
		"MQTT":   `{"topic":"t","payload":"AQ==","qos":0}`,
		"RCP":    `{"zone":1,"seq":1,"healthy":true}`,
		"SOMEIP": `{"service_id":1,"method_id":2,"protocol_version":1}`,
	}
	for proto, val := range cases {
		msg, err := referenceConvert(proto, []byte(val))
		if err != nil {
			t.Errorf("referenceConvert(%s): %v", proto, err)
		}
		if msg.Timestamp.IsZero() != true {
			t.Errorf("referenceConvert(%s): timestamp must be normalised to zero", proto)
		}
	}
	// SOME/IP with a bad protocol version must be rejected by Validate.
	if _, err := referenceConvert("SOMEIP", []byte(`{"service_id":1,"method_id":2,"protocol_version":9}`)); err == nil {
		t.Error("SOME/IP wrong protocol version must be rejected")
	}
}
