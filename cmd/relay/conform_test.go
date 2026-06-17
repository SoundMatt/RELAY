// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// --- unit tests for schema validators ---

//fusa:test REQ-RELAY-053
func TestValidateVersionDocValid(t *testing.T) {
	data := []byte(`{
		"tool":"go-can","version":"1.0.0","spec_version":"0.2",
		"language":"go","runtime":"go1.25.0","protocol":"CAN","protocol_int":1
	}`)
	fs := validateVersionDoc(data)
	for _, f := range fs {
		if f.Severity == sevFail {
			t.Errorf("unexpected FAIL: %s %s", f.Req, f.Message)
		}
	}
}

//fusa:test REQ-RELAY-053
func TestValidateVersionDocMissingFields(t *testing.T) {
	data := []byte(`{"tool":"x"}`) // missing version, spec_version, language, runtime
	fs := validateVersionDoc(data)
	fails := countBySeverity(fs, sevFail)
	if fails == 0 {
		t.Error("expected FAIL findings for missing required fields, got none")
	}
}

//fusa:test REQ-RELAY-053
func TestValidateVersionDocNullProtocol(t *testing.T) {
	data := []byte(`{
		"tool":"relay","version":"0.1.0","spec_version":"0.2",
		"language":"go","runtime":"go1.25.0"
	}`)
	fs := validateVersionDoc(data)
	hasWarn := false
	for _, f := range fs {
		if f.Severity == sevWarn && strings.Contains(f.Message, "protocol") {
			hasWarn = true
		}
		if f.Severity == sevFail && strings.Contains(f.Message, "protocol") {
			t.Errorf("null protocol should WARN not FAIL: %s", f.Message)
		}
	}
	if !hasWarn {
		t.Error("expected WARN for null protocol, got none")
	}
}

//fusa:test REQ-RELAY-053
func TestValidateVersionDocUnknownLanguage(t *testing.T) {
	data := []byte(`{
		"tool":"t","version":"1.0","spec_version":"0.2",
		"language":"java","runtime":"jvm","protocol":"X","protocol_int":99
	}`)
	fs := validateVersionDoc(data)
	hasFail := false
	for _, f := range fs {
		if f.Severity == sevFail && strings.Contains(f.Message, "language") {
			hasFail = true
		}
	}
	if !hasFail {
		t.Error("expected FAIL for unknown language, got none")
	}
}

//fusa:test REQ-RELAY-054
func TestValidateCapabilitiesDocValid(t *testing.T) {
	data := []byte(`{
		"kind":"capabilities","tool":"go-can","version":"1.0.0","spec_version":"0.2",
		"commands":["version","capabilities","status"],
		"transports":[],"features":[],"interfaces":[],"optional_interfaces":[],
		"adapt":true
	}`)
	fs := validateCapabilitiesDoc(data)
	for _, f := range fs {
		if f.Severity == sevFail {
			t.Errorf("unexpected FAIL: %s %s", f.Req, f.Message)
		}
	}
}

//fusa:test REQ-RELAY-054
func TestValidateCapabilitiesDocWithProtocol(t *testing.T) {
	// A single-protocol implementation includes protocol/protocol_int in its
	// capabilities document (spec §12.2). These MUST be accepted.
	data := []byte(`{
		"kind":"capabilities","tool":"go-can","protocol":"CAN","protocol_int":1,
		"version":"1.0.0","spec_version":"0.3",
		"commands":["version","capabilities","status"],
		"transports":["socketcan"],"features":["fd"],"interfaces":["Bus"],
		"optional_interfaces":["HealthProvider"],"adapt":true
	}`)
	fs := validateCapabilitiesDoc(data)
	for _, f := range fs {
		if f.Severity == sevFail {
			t.Errorf("single-protocol capabilities doc must not FAIL: %s %s", f.Req, f.Message)
		}
	}
}

//fusa:test REQ-RELAY-054
func TestValidateCapabilitiesDocWrongKind(t *testing.T) {
	data := []byte(`{
		"kind":"version","tool":"t","version":"1.0","spec_version":"0.2",
		"commands":["version","capabilities","status"],
		"transports":[],"features":[],"interfaces":[],"optional_interfaces":[],
		"adapt":false
	}`)
	fs := validateCapabilitiesDoc(data)
	hasFail := false
	for _, f := range fs {
		if f.Severity == sevFail && strings.Contains(f.Message, "kind") {
			hasFail = true
		}
	}
	if !hasFail {
		t.Error("expected FAIL for wrong kind, got none")
	}
}

//fusa:test REQ-RELAY-054
func TestValidateCapabilitiesDocMissingCommand(t *testing.T) {
	data := []byte(`{
		"kind":"capabilities","tool":"t","version":"1.0","spec_version":"0.2",
		"commands":["version","capabilities"],
		"transports":[],"features":[],"interfaces":[],"optional_interfaces":[],
		"adapt":false
	}`)
	fs := validateCapabilitiesDoc(data)
	hasFail := false
	for _, f := range fs {
		if f.Severity == sevFail && strings.Contains(f.Message, "status") {
			hasFail = true
		}
	}
	if !hasFail {
		t.Error("expected FAIL for missing status command, got none")
	}
}

//fusa:test REQ-RELAY-054
func TestValidateCapabilitiesDocAdaptWarn(t *testing.T) {
	data := []byte(`{
		"kind":"capabilities","tool":"relay","version":"0.1.0","spec_version":"0.2",
		"commands":["version","capabilities","status"],
		"transports":[],"features":[],"interfaces":[],"optional_interfaces":[],
		"adapt":false
	}`)
	fs := validateCapabilitiesDoc(data)
	hasWarn := false
	hasFail := false
	for _, f := range fs {
		if f.Severity == sevWarn && strings.Contains(f.Message, "adapt") {
			hasWarn = true
		}
		if f.Severity == sevFail {
			hasFail = true
		}
	}
	if !hasWarn {
		t.Error("expected WARN for adapt=false, got none")
	}
	if hasFail {
		t.Errorf("complete adapt=false doc should not FAIL: %+v", fs)
	}
}

//fusa:test REQ-RELAY-055
func TestValidateStatusDocValid(t *testing.T) {
	data := []byte(`{
		"tool":"go-can","version":"1.0.0","healthy":true,"connected":false,
		"endpoint":"","details":{}
	}`)
	fs := validateStatusDoc(data)
	for _, f := range fs {
		if f.Severity == sevFail {
			t.Errorf("unexpected FAIL: %s %s", f.Req, f.Message)
		}
	}
}

//fusa:test REQ-RELAY-055
func TestValidateStatusDocMissingHealthy(t *testing.T) {
	data := []byte(`{"tool":"t","version":"1.0","connected":false}`)
	fs := validateStatusDoc(data)
	hasFail := false
	for _, f := range fs {
		if f.Severity == sevFail && strings.Contains(f.Message, "healthy") {
			hasFail = true
		}
	}
	if !hasFail {
		t.Error("expected FAIL for missing healthy field")
	}
}

//fusa:test REQ-RELAY-055
func TestValidateStatusDocInvalidJSON(t *testing.T) {
	data := []byte(`not json`)
	fs := validateStatusDoc(data)
	if len(fs) != 1 || fs[0].Severity != sevFail {
		t.Error("expected single FAIL for invalid JSON")
	}
}

// --- integration test: relay conform relay ---

//fusa:test REQ-RELAY-052
func TestRunConformSelf(t *testing.T) {
	// Build the relay binary into a temp file and run conform against it.
	// Use runConform with a fake binary that emits valid JSON responses instead,
	// to avoid requiring `go build` in the test environment.
	// We test the overall flow by providing pre-canned JSON via a shell wrapper.
	// Since exec is required, skip if we can't build.
	bin := buildTestBinary(t)
	var out bytes.Buffer
	var errbuf bytes.Buffer
	err := runConform(&out, &errbuf, []string{bin})
	// relay conform relay should produce WARN (protocol null) but not FAIL.
	if err != nil {
		var code exitCode
		if errors.As(err, &code) && int(code) == 1 {
			t.Logf("conform output:\n%s", out.String())
			t.Error("relay conform relay returned FAIL exit code — relay must conform to itself")
		} else {
			t.Logf("conform output:\n%s", out.String())
			t.Errorf("unexpected error: %v", err)
		}
	}
}

//fusa:test REQ-RELAY-052
func TestRunConformJSONFormat(t *testing.T) {
	bin := buildTestBinary(t)
	var out bytes.Buffer
	var errbuf bytes.Buffer
	if err := runConform(&out, &errbuf, []string{"--format", "json", bin}); err != nil {
		var code exitCode
		if errors.As(err, &code) && int(code) == 1 {
			t.Logf("conform output:\n%s", out.String())
			t.Errorf("relay conform relay --format json returned FAIL: %v", err)
			return
		}
	}
	var cr conformResult
	if err := json.Unmarshal(out.Bytes(), &cr); err != nil {
		t.Fatalf("conform --format json output is not valid JSON: %v\noutput: %s", err, out.String())
	}
	if cr.Binary == "" {
		t.Error("conform JSON result: binary field is empty")
	}
	if len(cr.Findings) == 0 {
		t.Error("conform JSON result: findings is empty")
	}
}

//fusa:test REQ-RELAY-052
func TestRunConformNoArgs(t *testing.T) {
	var out bytes.Buffer
	var errbuf bytes.Buffer
	err := runConform(&out, &errbuf, []string{})
	var code exitCode
	if !errors.As(err, &code) || int(code) != 2 {
		t.Errorf("conform with no args should return exitCode(2), got %v", err)
	}
}

func countBySeverity(fs []conformFinding, sev conformSeverity) int {
	n := 0
	for _, f := range fs {
		if f.Severity == sev {
			n++
		}
	}
	return n
}

// buildTestBinary compiles the relay CLI into a temp directory and returns the path.
// Skips the test if the build fails (e.g., no Go toolchain in the test environment).
func buildTestBinary(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	bin := filepath.Join(dir, "relay")
	cmd := exec.Command("go", "build", "-o", bin, ".")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Skipf("could not build relay binary: %v\n%s", err, out)
	}
	return bin
}
