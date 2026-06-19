// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	relay "github.com/SoundMatt/RELAY"
)

//fusa:test REQ-RELAY-083
func TestInteropSelfEquivalent(t *testing.T) {
	// The relay binary's own convert must be equivalent to the in-process
	// reference for every vector (it is the reference).
	bin := buildTestBinary(t)
	var out, errb bytes.Buffer
	if err := runInterop(&out, &errb, []string{"--protocol", "CAN", bin}); err != nil {
		t.Fatalf("interop self: %v (%s)", err, errb.String())
	}
	s := out.String()
	if !strings.Contains(s, "RESULT: PASS") || strings.Contains(s, "MISMATCH") {
		t.Errorf("self-interop should be all-equivalent PASS:\n%s", s)
	}
}

//fusa:test REQ-RELAY-083
func TestInteropMissingConvert(t *testing.T) {
	// A binary that has no convert (exits non-zero) is skipped by default...
	noConvert := writeScript(t, "exit 3")
	var out, errb bytes.Buffer
	if err := runInterop(&out, &errb, []string{"--protocol", "CAN", noConvert}); err != nil {
		t.Errorf("non-strict missing convert must not fail: %v", err)
	}
	if !strings.Contains(out.String(), "SKIP") {
		t.Errorf("missing convert should be reported as SKIP:\n%s", out.String())
	}
	// ...but --strict turns the skip into a failure.
	var o2, e2 bytes.Buffer
	err := runInterop(&o2, &e2, []string{"--protocol", "CAN", "--strict", noConvert})
	var code exitCode
	if !errors.As(err, &code) || int(code) != 1 {
		t.Errorf("strict missing convert must exit 1, got %v", err)
	}
}

//fusa:test REQ-RELAY-083
func TestInteropMismatch(t *testing.T) {
	// A binary whose convert emits a divergent relay.Message must MISMATCH.
	bad := writeScript(t, `echo '{"protocol":1,"id":"999999","payload":"","timestamp":"0001-01-01T00:00:00Z","meta":{}}'`)
	var out, errb bytes.Buffer
	err := runInterop(&out, &errb, []string{"--protocol", "CAN", bad})
	var code exitCode
	if !errors.As(err, &code) || int(code) != 1 {
		t.Errorf("mismatch must exit 1, got %v", err)
	}
	if !strings.Contains(out.String(), "MISMATCH") {
		t.Errorf("expected MISMATCH in report:\n%s", out.String())
	}
}

//fusa:test REQ-RELAY-083
func TestInteropNoVectors(t *testing.T) {
	var out, errb bytes.Buffer
	err := runInterop(&out, &errb, []string{"--protocol", "NOSUCH"})
	var code exitCode
	if !errors.As(err, &code) || int(code) != 2 {
		t.Errorf("unknown protocol must exit 2, got %v", err)
	}
}

//fusa:test REQ-RELAY-083
func TestInteropFormats(t *testing.T) {
	bin := buildTestBinary(t)
	for _, f := range []string{"json", "markdown", "text"} {
		var out, errb bytes.Buffer
		if err := runInterop(&out, &errb, []string{"--protocol", "CAN", "--format", f, bin}); err != nil {
			t.Errorf("interop --format %s: %v", f, err)
		}
		if out.Len() == 0 {
			t.Errorf("interop --format %s produced no output", f)
		}
	}
	// Unknown format is an error.
	var o, e bytes.Buffer
	if err := runInterop(&o, &e, []string{"--protocol", "CAN", "--format", "yaml", bin}); err == nil {
		t.Error("unknown interop format must error")
	}
}

//fusa:test REQ-RELAY-083
func TestLoadInteropVectorsFilter(t *testing.T) {
	all, err := loadInteropVectors("", "")
	if err != nil {
		t.Fatal(err)
	}
	can, err := loadInteropVectors("", "CAN")
	if err != nil {
		t.Fatal(err)
	}
	if len(can) == 0 || len(can) >= len(all) {
		t.Errorf("CAN filter should select a strict subset: %d of %d", len(can), len(all))
	}
	for _, v := range can {
		if v.Protocol != "CAN" {
			t.Errorf("filter leaked non-CAN vector %s (%s)", v.Name, v.Protocol)
		}
	}
}

//fusa:test REQ-RELAY-083
func TestDiffMessages(t *testing.T) {
	ref := relayMsg("256", []byte{1, 2}, map[string]string{"can.fd": "true", "can.ext": "false"})
	// id + payload + changed meta + missing meta + extra meta all in one.
	got := relayMsg("999", []byte{9}, map[string]string{"can.fd": "false", "extra": "x"})
	d := diffMessages(ref, got)
	for _, want := range []string{"id", "payload differs", "meta can.fd", "missing meta can.ext", "extra meta extra"} {
		if !strings.Contains(d, want) {
			t.Errorf("diff %q missing %q", d, want)
		}
	}
	// Identical messages fall back to the generic "differs".
	if diffMessages(ref, ref) != "differs" {
		t.Errorf("equal messages should yield the generic fallback, got %q", diffMessages(ref, ref))
	}
}

//fusa:test REQ-RELAY-083
func TestInteropVectorsDir(t *testing.T) {
	// Drive interop from an on-disk vectors directory (the --vectors branch).
	dir := t.TempDir()
	raw, err := relay.Vector("can-standard-frame")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "v.json"), raw, 0o644); err != nil {
		t.Fatal(err)
	}
	bin := buildTestBinary(t)
	var out, errb bytes.Buffer
	if err := runInterop(&out, &errb, []string{"--vectors", dir, bin}); err != nil {
		t.Fatalf("interop --vectors: %v (%s)", err, errb.String())
	}
	if !strings.Contains(out.String(), "RESULT: PASS") {
		t.Errorf("interop from dir should PASS:\n%s", out.String())
	}
}

func relayMsg(id string, payload []byte, meta map[string]string) relay.Message {
	return relay.Message{Protocol: relay.CAN, ID: id, Payload: payload, Meta: meta}
}
