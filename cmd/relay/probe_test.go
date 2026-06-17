// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"
	"testing"
)

//fusa:test REQ-RELAY-060
func TestRunProbeSelfJSON(t *testing.T) {
	bin := buildTestBinary(t)
	var out, errb bytes.Buffer
	if err := runProbe(&out, &errb, []string{"--format", "json", bin}); err != nil {
		t.Fatalf("runProbe: %v", err)
	}
	var results []probeResult
	if err := json.Unmarshal(out.Bytes(), &results); err != nil {
		t.Fatalf("probe json: %v\n%s", err, out.String())
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	r := results[0]
	if !r.Conformant {
		t.Errorf("relay binary should be conformant: %+v", r)
	}
	if r.Tool != "relay" {
		t.Errorf("tool = %q, want relay", r.Tool)
	}
	if r.SpecVersion == "" {
		t.Error("spec_version must be reported")
	}
}

//fusa:test REQ-RELAY-060
func TestRunProbeSelfText(t *testing.T) {
	bin := buildTestBinary(t)
	var out, errb bytes.Buffer
	if err := runProbe(&out, &errb, []string{bin}); err != nil {
		t.Fatalf("runProbe: %v", err)
	}
	if !strings.Contains(out.String(), "relay") {
		t.Errorf("probe text should mention the relay tool, got:\n%s", out.String())
	}
	if !strings.Contains(out.String(), "BINARY") {
		t.Error("probe text should have a header row")
	}
}

//fusa:test REQ-RELAY-060
func TestRunProbeNonConformant(t *testing.T) {
	// A binary that exists but is not RELAY-conformant (e.g. /bin/echo) must be
	// reported as not conformant in explicit mode, not crash.
	echo, err := filepath.Abs("/bin/echo")
	if err != nil {
		t.Skip("no /bin/echo")
	}
	var out, errb bytes.Buffer
	if err := runProbe(&out, &errb, []string{"--format", "json", echo}); err != nil {
		t.Fatalf("runProbe: %v", err)
	}
	var results []probeResult
	if err := json.Unmarshal(out.Bytes(), &results); err != nil {
		t.Fatalf("probe json: %v", err)
	}
	if len(results) != 1 || results[0].Conformant {
		t.Errorf("/bin/echo must be reported as non-conformant, got %+v", results)
	}
}

//fusa:test REQ-RELAY-060
func TestRunProbeNoArgs(t *testing.T) {
	var out, errb bytes.Buffer
	err := runProbe(&out, &errb, nil)
	var code exitCode
	if !errors.As(err, &code) || int(code) != 2 {
		t.Errorf("probe with no args should return exitCode(2), got %v", err)
	}
}
