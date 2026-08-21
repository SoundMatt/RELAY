// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	relay "github.com/SoundMatt/RELAY/v2"
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
//fusa:test REQ-RELAY-047
//fusa:test REQ-RELAY-079
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
	// No protocol declared, so this fixture must self-declare multi_protocol:
	// true (spec §12.2, §17 Requirement 1) to avoid the new protocol-null FAIL.
	data := []byte(`{
		"kind":"capabilities","tool":"go-can","version":"1.0.0","spec_version":"0.2",
		"multi_protocol":true,
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
//fusa:test REQ-RELAY-048
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
//fusa:test REQ-RELAY-097
func TestValidateCapabilitiesDocAdaptFalseFails(t *testing.T) {
	// A single-protocol implementation (multi_protocol absent/false) that
	// exports no Adapt() MUST FAIL (spec §17 Requirement 6), not WARN.
	data := []byte(`{
		"kind":"capabilities","tool":"go-can","protocol":"CAN","protocol_int":1,
		"version":"0.1.0","spec_version":"0.2",
		"commands":["version","capabilities","status"],
		"transports":[],"features":[],"interfaces":[],"optional_interfaces":[],
		"adapt":false
	}`)
	fs := validateCapabilitiesDoc(data)
	hasFail := false
	for _, f := range fs {
		if f.Severity == sevFail && strings.Contains(f.Message, "adapt") {
			hasFail = true
		}
	}
	if !hasFail {
		t.Error("expected FAIL for adapt=false on a single-protocol tool, got none")
	}
}

//fusa:test REQ-RELAY-054
//fusa:test REQ-RELAY-097
func TestValidateCapabilitiesDocNullProtocolFails(t *testing.T) {
	// A single-protocol implementation (multi_protocol absent/false) with a
	// null protocol MUST FAIL (spec §17 Requirement 1), not WARN.
	data := []byte(`{
		"kind":"capabilities","tool":"go-can","version":"0.1.0","spec_version":"0.2",
		"commands":["version","capabilities","status"],
		"transports":[],"features":[],"interfaces":[],"optional_interfaces":[],
		"adapt":true
	}`)
	fs := validateCapabilitiesDoc(data)
	hasFail := false
	for _, f := range fs {
		if f.Severity == sevFail && strings.Contains(f.Message, "protocol") {
			hasFail = true
		}
	}
	if !hasFail {
		t.Error("expected FAIL for null protocol on a single-protocol tool, got none")
	}
}

//fusa:test REQ-RELAY-054
//fusa:test REQ-RELAY-097
func TestValidateCapabilitiesDocMultiProtocolExempt(t *testing.T) {
	// multi_protocol:true legitimizes both a null protocol and adapt:false
	// (spec §12.2, §17 Requirements 1 and 6) — RELAY's own reference CLI's
	// exact shape. Neither MUST produce a finding at all, not even a WARN.
	data := []byte(`{
		"kind":"capabilities","tool":"relay","multi_protocol":true,
		"version":"0.1.0","spec_version":"0.2",
		"commands":["version","capabilities","status"],
		"transports":[],"features":[],"interfaces":[],"optional_interfaces":[],
		"adapt":false
	}`)
	fs := validateCapabilitiesDoc(data)
	for _, f := range fs {
		if f.Severity != sevPass {
			t.Errorf("multi_protocol:true doc must produce no FAIL/WARN for protocol/adapt, got %s %s: %s", f.Severity, f.Req, f.Message)
		}
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
//fusa:test REQ-RELAY-049
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
func TestRunConformJSONFormatFail(t *testing.T) {
	// Regression test: --format json MUST still surface a non-zero exit code
	// when the conformance result is FAIL, exactly like --format text does.
	// Previously runConform returned right after enc.Encode(cr) in the "json"
	// case, bypassing the sevFail exit-code check entirely.
	bin := buildFailingBinary(t)
	var out bytes.Buffer
	var errbuf bytes.Buffer
	err := runConform(&out, &errbuf, []string{"--format", "json", bin})
	if err == nil {
		t.Fatalf("expected non-nil error for FAIL result in --format json, got nil\noutput: %s", out.String())
	}
	var code exitCode
	if !errors.As(err, &code) || int(code) != 1 {
		t.Errorf("expected exitCode(1) for FAIL result in --format json, got %v", err)
	}
	var cr conformResult
	if jsonErr := json.Unmarshal(out.Bytes(), &cr); jsonErr != nil {
		t.Fatalf("conform --format json output is not valid JSON: %v\noutput: %s", jsonErr, out.String())
	}
	if cr.Result != sevFail {
		t.Errorf("expected conformResult.Result=FAIL, got %s", cr.Result)
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

// --- tests for --manifest / buildManifest (spec §17.2, Requirement 15) ---

//fusa:test REQ-RELAY-095
func TestBuildManifestSelf(t *testing.T) {
	bin := buildTestBinary(t)
	m := buildManifest(bin)

	if m.Kind != "relay-conform-manifest" {
		t.Errorf("Kind = %q, want %q", m.Kind, "relay-conform-manifest")
	}
	if m.ManifestVersion != "relay-conform/1" {
		t.Errorf("ManifestVersion = %q, want %q", m.ManifestVersion, "relay-conform/1")
	}
	if m.Tool == "" {
		t.Error("Tool is empty")
	}
	if m.BinaryVersion == "" {
		t.Error("BinaryVersion is empty")
	}
	if m.SpecVersion == "" {
		t.Error("SpecVersion is empty")
	}
	if len(m.Requirements) != 16 {
		t.Fatalf("len(Requirements) = %d, want 16", len(m.Requirements))
	}
	for i, r := range m.Requirements {
		if r.ID != i+1 {
			t.Errorf("Requirements[%d].ID = %d, want %d", i, r.ID, i+1)
		}
		if r.Status == "" {
			t.Errorf("Requirements[%d] (id=%d) has empty status", i, r.ID)
		}
		if r.Verifier == "" {
			t.Errorf("Requirements[%d] (id=%d) has empty verifier", i, r.ID)
		}
		// §17.2: a manifest MUST NOT report PASS for a requirement relay
		// conform cannot actually observe. Only 1, 6, 7, 12 may be PASS/FAIL;
		// everything else must never be reported PASS.
		observable := map[int]bool{1: true, 6: true, 7: true, 12: true}
		if !observable[r.ID] && r.Status == statusPass {
			t.Errorf("Requirements[%d] (id=%d) reports PASS but is not in the observable set", i, r.ID)
		}
	}
	// relay must conform to itself, so the overall manifest result must be PASS.
	if m.Overall != statusPass {
		t.Errorf("Overall = %s, want %s (relay must conform to itself)", m.Overall, statusPass)
	}
}

//fusa:test REQ-RELAY-095
func TestBuildManifestNotObservableDefaults(t *testing.T) {
	bin := buildTestBinary(t)
	m := buildManifest(bin)

	wantNotObservable := []int{2, 3, 4, 5, 8, 9, 10, 11, 13, 14, 15, 16}
	byID := map[int]manifestRequirement{}
	for _, r := range m.Requirements {
		byID[r.ID] = r
	}
	for _, id := range wantNotObservable {
		r, ok := byID[id]
		if !ok {
			t.Errorf("requirement %d missing from manifest", id)
			continue
		}
		if r.Status != statusNotObservable {
			t.Errorf("requirement %d status = %s, want %s", id, r.Status, statusNotObservable)
		}
	}
}

//fusa:test REQ-RELAY-095
func TestBuildManifestCapabilitiesSHA256(t *testing.T) {
	bin := buildTestBinary(t)
	capsJSON, err := runBinaryCommand(bin, []string{"capabilities"})
	if err != nil {
		t.Fatalf("capabilities: %v", err)
	}
	sum := sha256.Sum256(capsJSON)
	want := hex.EncodeToString(sum[:])

	m := buildManifest(bin)
	if m.CapabilitiesSHA256 != want {
		t.Errorf("CapabilitiesSHA256 = %q, want %q (independently computed sha256 of raw capabilities bytes)", m.CapabilitiesSHA256, want)
	}
}

//fusa:test REQ-RELAY-095
func TestBuildManifestSchemaValid(t *testing.T) {
	bin := buildTestBinary(t)
	m := buildManifest(bin)

	data, err := json.Marshal(m)
	if err != nil {
		t.Fatalf("json.Marshal(manifest): %v", err)
	}
	var asAny interface{}
	if err := json.Unmarshal(data, &asAny); err != nil {
		t.Fatalf("re-unmarshal manifest JSON: %v", err)
	}
	schemaJSON, err := relay.Schema("relay-conform-manifest")
	if err != nil {
		t.Fatalf("no embedded relay-conform-manifest schema: %v", err)
	}
	violations := validateSchema(schemaJSON, asAny)
	if len(violations) != 0 {
		t.Errorf("manifest does not conform to relay-conform-manifest schema: %v\nmanifest: %s", violations, data)
	}
}

//fusa:test REQ-RELAY-095
func TestBuildManifestFailPropagatesToOverall(t *testing.T) {
	bin := buildFailingBinary(t)
	m := buildManifest(bin)

	// version/capabilities/status all fail against a binary that always
	// exits non-zero, so requirements 1, 6, 7 must be FAIL and Overall must
	// be FAIL.
	byID := map[int]manifestRequirement{}
	for _, r := range m.Requirements {
		byID[r.ID] = r
	}
	for _, id := range []int{1, 6, 7} {
		if got := byID[id].Status; got != statusFail {
			t.Errorf("requirement %d status = %s, want %s", id, got, statusFail)
		}
	}
	if m.Overall != statusFail {
		t.Errorf("Overall = %s, want %s", m.Overall, statusFail)
	}
}

//fusa:test REQ-RELAY-095
func TestRunConformManifestFlag(t *testing.T) {
	bin := buildTestBinary(t)
	var out bytes.Buffer
	var errbuf bytes.Buffer
	err := runConform(&out, &errbuf, []string{"--manifest", bin})
	if err != nil {
		t.Fatalf("relay conform --manifest relay: unexpected error: %v\noutput: %s", err, out.String())
	}
	var m conformManifest
	if jsonErr := json.Unmarshal(out.Bytes(), &m); jsonErr != nil {
		t.Fatalf("conform --manifest output is not valid JSON: %v\noutput: %s", jsonErr, out.String())
	}
	if m.Overall != statusPass {
		t.Errorf("Overall = %s, want %s", m.Overall, statusPass)
	}
	if len(m.Requirements) != 16 {
		t.Errorf("len(Requirements) = %d, want 16", len(m.Requirements))
	}
}

//fusa:test REQ-RELAY-095
func TestRunConformManifestFlagFail(t *testing.T) {
	bin := buildFailingBinary(t)
	var out bytes.Buffer
	var errbuf bytes.Buffer
	err := runConform(&out, &errbuf, []string{"--manifest", bin})
	if err == nil {
		t.Fatalf("expected non-nil error for FAIL manifest, got nil\noutput: %s", out.String())
	}
	var code exitCode
	if !errors.As(err, &code) || int(code) != 1 {
		t.Errorf("expected exitCode(1) for FAIL manifest, got %v", err)
	}
	var m conformManifest
	if jsonErr := json.Unmarshal(out.Bytes(), &m); jsonErr != nil {
		t.Fatalf("conform --manifest output is not valid JSON: %v\noutput: %s", jsonErr, out.String())
	}
	if m.Overall != statusFail {
		t.Errorf("Overall = %s, want %s", m.Overall, statusFail)
	}
}

// --- tests for --attestation / buildAttestation (spec §20.6) ---

//fusa:test REQ-RELAY-098
func TestBuildAttestationSelf(t *testing.T) {
	bin := buildTestBinary(t)
	raw, err := os.ReadFile(bin)
	if err != nil {
		t.Fatalf("read binary: %v", err)
	}
	sum := sha256.Sum256(raw)
	wantDigest := hex.EncodeToString(sum[:])

	a, err := buildAttestation(bin)
	if err != nil {
		t.Fatalf("buildAttestation: %v", err)
	}

	if a.Type != "https://in-toto.io/Statement/v1" {
		t.Errorf("Type = %q, want in-toto Statement v1", a.Type)
	}
	if a.PredicateType != "https://relay.dev/attestation/relay-conform/1" {
		t.Errorf("PredicateType = %q, want relay-conform/1", a.PredicateType)
	}
	if len(a.Subject) != 1 {
		t.Fatalf("len(Subject) = %d, want 1", len(a.Subject))
	}
	if a.Subject[0].Digest.SHA256 != wantDigest {
		t.Errorf("Subject[0].Digest.SHA256 = %q, want %q (independently computed sha256 of the binary file)", a.Subject[0].Digest.SHA256, wantDigest)
	}
	if a.Predicate.SpecVersion == "" {
		t.Error("Predicate.SpecVersion is empty")
	}
	if a.Predicate.ConformanceManifest.Kind != "relay-conform-manifest" {
		t.Errorf("Predicate.ConformanceManifest.Kind = %q, want relay-conform-manifest", a.Predicate.ConformanceManifest.Kind)
	}
	if len(a.Predicate.ConformanceManifest.Requirements) != 16 {
		t.Errorf("len(Predicate.ConformanceManifest.Requirements) = %d, want 16", len(a.Predicate.ConformanceManifest.Requirements))
	}
	// §20.6: MUST be false — this type has no signing mechanism at all.
	if a.Predicate.Signed {
		t.Error("Predicate.Signed = true, want false (relay conform --attestation never signs)")
	}
	if a.Predicate.SafetyEvidenceSummary != nil {
		t.Errorf("Predicate.SafetyEvidenceSummary = %v, want nil (RELAY's reference tooling cannot generate one)", a.Predicate.SafetyEvidenceSummary)
	}
}

//fusa:test REQ-RELAY-098
func TestBuildAttestationVectorsVersionMatchesEmbedded(t *testing.T) {
	bin := buildTestBinary(t)
	a, err := buildAttestation(bin)
	if err != nil {
		t.Fatalf("buildAttestation: %v", err)
	}
	vm, err := relay.ParsedVectorsManifest()
	if err != nil {
		t.Fatalf("ParsedVectorsManifest: %v", err)
	}
	if a.Predicate.VectorsVersion != vm.VectorsVersion {
		t.Errorf("Predicate.VectorsVersion = %q, want %q (this tool's own embedded vectors_manifest.json)", a.Predicate.VectorsVersion, vm.VectorsVersion)
	}
}

//fusa:test REQ-RELAY-098
func TestBuildAttestationSchemaValid(t *testing.T) {
	bin := buildTestBinary(t)
	a, err := buildAttestation(bin)
	if err != nil {
		t.Fatalf("buildAttestation: %v", err)
	}
	data, err := json.Marshal(a)
	if err != nil {
		t.Fatalf("json.Marshal(attestation): %v", err)
	}
	var asAny interface{}
	if err := json.Unmarshal(data, &asAny); err != nil {
		t.Fatalf("re-unmarshal attestation JSON: %v", err)
	}
	schemaJSON, err := relay.Schema("relay-conform-attestation")
	if err != nil {
		t.Fatalf("no embedded relay-conform-attestation schema: %v", err)
	}
	violations := validateSchema(schemaJSON, asAny)
	if len(violations) != 0 {
		t.Errorf("attestation does not conform to relay-conform-attestation schema: %v\nattestation: %s", violations, data)
	}
}

//fusa:test REQ-RELAY-098
func TestBuildAttestationUnreadableBinary(t *testing.T) {
	_, err := buildAttestation(filepath.Join(t.TempDir(), "does-not-exist"))
	if err == nil {
		t.Fatal("expected an error for a nonexistent binary, got nil")
	}
}

//fusa:test REQ-RELAY-098
func TestRunConformAttestationFlag(t *testing.T) {
	bin := buildTestBinary(t)
	var out bytes.Buffer
	var errbuf bytes.Buffer
	err := runConform(&out, &errbuf, []string{"--attestation", bin})
	if err != nil {
		t.Fatalf("relay conform --attestation relay: unexpected error: %v\noutput: %s", err, out.String())
	}
	var a conformAttestation
	if jsonErr := json.Unmarshal(out.Bytes(), &a); jsonErr != nil {
		t.Fatalf("conform --attestation output is not valid JSON: %v\noutput: %s", jsonErr, out.String())
	}
	if a.Predicate.ConformanceManifest.Overall != statusPass {
		t.Errorf("Predicate.ConformanceManifest.Overall = %s, want %s", a.Predicate.ConformanceManifest.Overall, statusPass)
	}
}

//fusa:test REQ-RELAY-098
func TestRunConformAttestationFlagFail(t *testing.T) {
	bin := buildFailingBinary(t)
	var out bytes.Buffer
	var errbuf bytes.Buffer
	err := runConform(&out, &errbuf, []string{"--attestation", bin})
	if err == nil {
		t.Fatalf("expected non-nil error for a FAIL-manifest attestation, got nil\noutput: %s", out.String())
	}
	var code exitCode
	if !errors.As(err, &code) || int(code) != 1 {
		t.Errorf("expected exitCode(1) for a FAIL-manifest attestation, got %v", err)
	}
	var a conformAttestation
	if jsonErr := json.Unmarshal(out.Bytes(), &a); jsonErr != nil {
		t.Fatalf("conform --attestation output is not valid JSON: %v\noutput: %s", jsonErr, out.String())
	}
	if a.Predicate.ConformanceManifest.Overall != statusFail {
		t.Errorf("Predicate.ConformanceManifest.Overall = %s, want %s", a.Predicate.ConformanceManifest.Overall, statusFail)
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

// buildFailingBinary writes a shell script that always exits non-zero,
// regardless of the arguments passed to it, so conformBinary's every step
// (version/capabilities/status) fails and the overall result is FAIL.
func buildFailingBinary(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	bin := filepath.Join(dir, "always-fail")
	script := "#!/bin/sh\nexit 1\n"
	if err := os.WriteFile(bin, []byte(script), 0o755); err != nil {
		t.Fatalf("could not write failing test binary: %v", err)
	}
	return bin
}
