// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	relay "github.com/SoundMatt/RELAY/v2"
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

//fusa:test REQ-RELAY-101
func TestInteropMatrixFields(t *testing.T) {
	// The JSON report is a versioned relay-interop-matrix/1 artifact with an
	// explicit participants list, not just an ad hoc pass/fail blob.
	bin := buildTestBinary(t)
	var out, errb bytes.Buffer
	if err := runInterop(&out, &errb, []string{"--protocol", "CAN", "--format", "json", bin}); err != nil {
		t.Fatalf("interop: %v (%s)", err, errb.String())
	}
	var doc interopDoc
	if err := json.Unmarshal(out.Bytes(), &doc); err != nil {
		t.Fatalf("unmarshal interop json: %v", err)
	}
	if doc.Kind != "relay-interop-matrix" {
		t.Errorf("kind = %q, want relay-interop-matrix", doc.Kind)
	}
	if doc.MatrixVersion != "relay-interop-matrix/1" {
		t.Errorf("matrix_version = %q, want relay-interop-matrix/1", doc.MatrixVersion)
	}
	if len(doc.Participants) != 1 || doc.Participants[0] != filepath.Base(bin) {
		t.Errorf("participants = %v, want [%s]", doc.Participants, filepath.Base(bin))
	}
	if len(doc.Regressions) != 0 {
		t.Errorf("no --baseline given: regressions should be empty, got %v", doc.Regressions)
	}
}

//fusa:test REQ-RELAY-101
func TestInteropBaselineRegression(t *testing.T) {
	// A cell that was EQUIVALENT in a committed baseline but MISMATCHes now
	// is a regression: --baseline must fail even though the mismatching
	// binary was never checked against this specific baseline before.
	//
	// Restricted to a single accept-path (canonical) vector via --vectors:
	// a broken-convert binary's reject-path cells look "correctly rejected"
	// (Equivalent=true) by design (see the v.Kind=="error" branch), which
	// would otherwise mask a genuine accept-path regression in this test.
	bin := buildTestBinary(t)
	dir := t.TempDir()
	vecDir := t.TempDir()
	raw, err := relay.Vector("can-standard-frame")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(vecDir, "v.json"), raw, 0o644); err != nil {
		t.Fatal(err)
	}

	var baseOut, errb bytes.Buffer
	if err := runInterop(&baseOut, &errb, []string{"--vectors", vecDir, "--format", "json", bin}); err != nil {
		t.Fatalf("baseline run: %v (%s)", err, errb.String())
	}
	basePath := filepath.Join(dir, "baseline.json")
	if err := os.WriteFile(basePath, baseOut.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}

	// Now regress: same participant name (relay's own binary name), but a
	// broken convert. --baseline compares by participant name, so drive the
	// regressed run through a script literally named after the reference
	// binary's basename to land on the same matrix key.
	regressedDir := t.TempDir()
	regressedBin := filepath.Join(regressedDir, filepath.Base(bin))
	script := "#!/bin/sh\ncase \"$1\" in\ncapabilities) echo '{\"kind\":\"capabilities\",\"tool\":\"fake\",\"protocol\":\"CAN\",\"commands\":[\"version\",\"capabilities\",\"status\",\"convert\"],\"adapt\":true}' ;;\nconvert) exit 1 ;;\nesac\n"
	if err := os.WriteFile(regressedBin, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}

	var out, errb2 bytes.Buffer
	err = runInterop(&out, &errb2, []string{"--vectors", vecDir, "--format", "json", "--baseline", basePath, regressedBin})
	var code exitCode
	if !errors.As(err, &code) || int(code) != 1 {
		t.Fatalf("regressed run must exit 1, got %v (%s)", err, out.String())
	}
	var doc interopDoc
	if err := json.Unmarshal(out.Bytes(), &doc); err != nil {
		t.Fatalf("unmarshal interop json: %v", err)
	}
	// Assert on the structured Regressions field directly, not just exit
	// code/text: the underlying mismatch already forces exit 1 on its own,
	// so this is the only assertion that actually exercises the regression
	// detector rather than the ordinary any-mismatch-fails path. Must
	// require an EXACT participant match (": <participant> regressed"),
	// not a substring: the always-equivalent "relay (reference)" row that
	// both the baseline and current run also carry contains the built
	// test binary's own name ("relay") as a substring, which would
	// otherwise mask a broken regression detector by matching on the
	// reference row's own spurious "regression" instead of the real one.
	participant := filepath.Base(bin)
	wantPrefix := "can-standard-frame: " + participant + " regressed"
	var found string
	for _, r := range doc.Regressions {
		if strings.HasPrefix(r, wantPrefix) {
			found = r
			break
		}
	}
	if found == "" {
		t.Fatalf("expected a regression naming participant %q, got %v", participant, doc.Regressions)
	}
	if !strings.Contains(found, "regressed from EQUIVALENT") {
		t.Errorf("regression entry = %q, want it to mention regressed from EQUIVALENT", found)
	}
}

//fusa:test REQ-RELAY-101
func TestInteropBaselineNoRegressionOnPreexistingMismatch(t *testing.T) {
	// A participant that was never in the baseline (or never equivalent for
	// a given vector) is not a "regression" — --baseline must not newly fail
	// on pre-existing, already-known non-conformance.
	emptyBaseline := filepath.Join(t.TempDir(), "empty.json")
	if err := os.WriteFile(emptyBaseline, []byte(`{"kind":"relay-interop-matrix","matrix_version":"relay-interop-matrix/1","reference":"relay (reference)","participants":[],"result":"PASS","vectors":[]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	bad := writeScript(t, `echo '{"protocol":1,"id":"999999","payload":"","timestamp":"0001-01-01T00:00:00Z","meta":{}}'`)
	var out, errb bytes.Buffer
	err := runInterop(&out, &errb, []string{"--protocol", "CAN", "--baseline", emptyBaseline, bad})
	// Still fails overall (interop always fails on any current mismatch),
	// but must not be reported as a *regression* since the baseline had no
	// EQUIVALENT cell for this participant to regress from.
	var code exitCode
	if !errors.As(err, &code) || int(code) != 1 {
		t.Fatalf("mismatch must still exit 1 regardless of baseline, got %v", err)
	}
	if strings.Contains(out.String(), "regressed from EQUIVALENT") {
		t.Errorf("a cell absent from the baseline must not be reported as a regression:\n%s", out.String())
	}
}

//fusa:test REQ-RELAY-101
func TestInteropBaselineMissingFile(t *testing.T) {
	bin := buildTestBinary(t)
	var out, errb bytes.Buffer
	err := runInterop(&out, &errb, []string{"--protocol", "CAN", "--baseline", filepath.Join(t.TempDir(), "nope.json"), bin})
	var code exitCode
	if !errors.As(err, &code) || int(code) != 2 {
		t.Errorf("missing --baseline file must exit 2, got %v", err)
	}
}

//fusa:test REQ-RELAY-101
func TestInteropRegressionsHelper(t *testing.T) {
	base := interopDoc{Vectors: []interopVectorResult{
		{Vector: "v1", Cells: []interopCell{{Participant: "a", Equivalent: true}}},
		{Vector: "v2", Cells: []interopCell{{Participant: "a", Equivalent: false}}}, // never equivalent
	}}
	cur := interopDoc{Vectors: []interopVectorResult{
		{Vector: "v1", Cells: []interopCell{{Participant: "a", Equivalent: false, Detail: "payload differs"}}}, // regressed
		{Vector: "v2", Cells: []interopCell{{Participant: "a", Equivalent: false}}},                            // still not equivalent, not a regression
	}}
	got := interopRegressions(base, cur)
	if len(got) != 1 || !strings.Contains(got[0], "v1") || !strings.Contains(got[0], "payload differs") {
		t.Errorf("interopRegressions = %v, want exactly one v1 regression mentioning the detail", got)
	}
}

//fusa:test REQ-RELAY-083
func TestInteropBrokenConvertFails(t *testing.T) {
	// A spoke that ADVERTISES convert but whose convert errors is a conformance
	// failure even in non-strict mode (distinct from a genuinely absent convert).
	broken := writeScript(t, `case "$1" in
capabilities) echo '{"kind":"capabilities","tool":"fake","protocol":"CAN","commands":["version","capabilities","status","convert"],"adapt":true}' ;;
convert) exit 1 ;;
esac`)
	var out, errb bytes.Buffer
	err := runInterop(&out, &errb, []string{"--protocol", "CAN", broken})
	var code exitCode
	if !errors.As(err, &code) || int(code) != 1 {
		t.Errorf("advertised-but-broken convert must FAIL (exit 1), got %v", err)
	}
	if strings.Contains(out.String(), "SKIP") {
		t.Errorf("advertised-but-broken convert must not be reported as SKIP:\n%s", out.String())
	}
}
