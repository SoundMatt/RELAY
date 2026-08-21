// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	relay "github.com/SoundMatt/RELAY/v2"
)

// conformSeverity is the severity of a conformance finding.
type conformSeverity string

const (
	sevFail conformSeverity = "FAIL"
	sevWarn conformSeverity = "WARN"
	sevPass conformSeverity = "PASS"
)

// conformFinding is a single conformance check result.
type conformFinding struct {
	Severity conformSeverity `json:"severity"`
	Req      string          `json:"req"`
	Message  string          `json:"message"`
}

func fail(req, msg string, args ...interface{}) conformFinding {
	return conformFinding{sevFail, req, fmt.Sprintf(msg, args...)}
}

func warn(req, msg string, args ...interface{}) conformFinding {
	return conformFinding{sevWarn, req, fmt.Sprintf(msg, args...)}
}

func pass(req, msg string) conformFinding {
	return conformFinding{sevPass, req, msg}
}

// conformResult is the overall conformance report.
type conformResult struct {
	Binary   string           `json:"binary"`
	Result   conformSeverity  `json:"result"`
	Findings []conformFinding `json:"findings"`
}

// runConform implements
// `relay conform [--format text|json] [--strict] [--manifest] [--attestation] <binary>`.
//
//fusa:req REQ-RELAY-052
//fusa:req REQ-RELAY-095
//fusa:req REQ-RELAY-098
func runConform(stdout, stderr io.Writer, args []string) error {
	fs := flag.NewFlagSet("conform", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	format := fs.String("format", "text", "Output format: text or json")
	strict := fs.Bool("strict", false, "Treat WARN as FAIL")
	manifest := fs.Bool("manifest", false, "Emit a relay-conform/1 manifest (spec §17.2) instead of findings")
	attestation := fs.Bool("attestation", false, "Emit an unsigned relay-conform-attestation/1 predicate (spec §20.6) instead of findings")
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("relay conform: %w", err)
	}
	if fs.NArg() == 0 {
		fmt.Fprintln(stderr, "Usage: relay conform [--format text|json] [--strict] [--manifest] [--attestation] <binary>")
		return exitCode(2)
	}
	if fs.NArg() > 1 {
		// Go's flag package stops parsing at the first non-flag argument, so
		// `relay conform <binary> --strict` would otherwise silently leave
		// "--strict" as an unconsumed extra positional argument instead of
		// being parsed as a flag — silently downgrading a FAIL-worthy check
		// to a passing one. Reject it explicitly rather than ignore it.
		fmt.Fprintf(stderr, "relay conform: unexpected extra arguments %v — flags must precede <binary>\n", fs.Args()[1:])
		return exitCode(2)
	}

	if *manifest {
		m := buildManifest(fs.Arg(0))
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "    ")
		if err := enc.Encode(m); err != nil {
			return err
		}
		if m.Overall == statusFail {
			return exitCode(1)
		}
		return nil
	}

	if *attestation {
		a, err := buildAttestation(fs.Arg(0))
		if err != nil {
			return fmt.Errorf("relay conform --attestation: %w", err)
		}
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "    ")
		if err := enc.Encode(a); err != nil {
			return err
		}
		if a.Predicate.ConformanceManifest.Overall == statusFail {
			return exitCode(1)
		}
		return nil
	}

	cr := conformBinary(fs.Arg(0), *strict)

	switch *format {
	case "json":
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "    ")
		if err := enc.Encode(cr); err != nil {
			return err
		}
	case "text":
		printConformText(stdout, cr)
	default:
		return fmt.Errorf("relay conform: unknown format %q", *format)
	}

	if cr.Result == sevFail {
		return exitCode(1)
	}
	return nil
}

// conformBinary runs the §12 schema checks against one binary and returns the
// aggregated result. With strict, WARN findings escalate the result to FAIL.
//
//fusa:req REQ-RELAY-052
func conformBinary(binary string, strict bool) conformResult {
	var all []conformFinding

	// --- §17.7 / §12.1 version --format json ---
	versionJSON, err := runBinaryCommand(binary, []string{"version", "--format", "json"})
	if err != nil {
		all = append(all, fail("§17.7", "version --format json failed: %v", err))
	} else {
		all = append(all, validateVersionDoc(versionJSON)...)
	}

	// --- §17.7 / §12.2 capabilities ---
	capsJSON, err := runBinaryCommand(binary, []string{"capabilities"})
	if err != nil {
		all = append(all, fail("§17.7", "capabilities failed: %v", err))
	} else {
		all = append(all, validateCapabilitiesDoc(capsJSON)...)
	}

	// --- §17.7 / §12.3 status --format json ---
	statusJSON, err := runBinaryCommand(binary, []string{"status", "--format", "json"})
	if err != nil {
		// status may not accept --format json; try without
		statusJSON, err = runBinaryCommand(binary, []string{"status"})
		if err != nil {
			all = append(all, fail("§17.7", "status command failed: %v", err))
		} else {
			all = append(all, validateStatusDoc(statusJSON)...)
		}
	} else {
		all = append(all, validateStatusDoc(statusJSON)...)
	}

	// Compute overall result.
	result := sevPass
	for _, f := range all {
		if f.Severity == sevFail {
			result = sevFail
			break
		}
		if f.Severity == sevWarn && strict {
			result = sevFail
		} else if f.Severity == sevWarn && result == sevPass {
			result = sevWarn
		}
	}
	return conformResult{Binary: binary, Result: result, Findings: all}
}

// manifestStatus is a per-requirement status in a relay-conform/1 manifest
// (spec §17.2). It is deliberately a narrower vocabulary than conformSeverity:
// a manifest never reports PASS for a requirement relay conform cannot
// actually observe.
type manifestStatus string

const (
	statusPass            manifestStatus = "PASS"
	statusFail            manifestStatus = "FAIL"
	statusShapeOnly       manifestStatus = "SHAPE_ONLY"
	statusNotObservable   manifestStatus = "NOT_OBSERVABLE"
	verifierRelayConform                 = "relay conform"
	verifierTestSuite                    = "implementation test suite"
	verifierBuildAndTest                 = "implementation's own build process and test suite"
	verifierCI                           = "implementation's own CI"
	verifierManifestSplit                = "relay conform (manifest content); implementation's own CI (regenerate + diff)"
)

// manifestRequirement is one entry in a relay-conform/1 manifest's
// "requirements" array (spec §17.2).
type manifestRequirement struct {
	ID       int            `json:"id"`
	Status   manifestStatus `json:"status"`
	Verifier string         `json:"verifier"`
}

// conformManifest is the relay-conform/1 manifest document (spec §17.2,
// §17 Requirement 15).
type conformManifest struct {
	Kind               string                `json:"kind"`
	ManifestVersion    string                `json:"manifest_version"`
	Tool               string                `json:"tool"`
	BinaryVersion      string                `json:"binary_version"`
	SpecVersion        string                `json:"spec_version"`
	GitSHA             string                `json:"git_sha"`
	CapabilitiesSHA256 string                `json:"capabilities_sha256"`
	Requirements       []manifestRequirement `json:"requirements"`
	Overall            manifestStatus        `json:"overall"`
}

// buildManifest generates a relay-conform/1 manifest (spec §17.2) for one
// binary. It fetches version/capabilities/status exactly once each — the
// same single-fetch discipline as conformBinary — so a manifest and a
// findings report generated from the same invocation never see different
// binary output.
//
//fusa:req REQ-RELAY-095
func buildManifest(binary string) conformManifest {
	m := conformManifest{
		Kind:            "relay-conform-manifest",
		ManifestVersion: "relay-conform/1",
	}

	versionJSON, verErr := runBinaryCommand(binary, []string{"version", "--format", "json"})
	var vFindings []conformFinding
	if verErr != nil {
		vFindings = []conformFinding{fail("§17.7", "version --format json failed: %v", verErr)}
	} else {
		vFindings = validateVersionDoc(versionJSON)
		var vdoc struct {
			Tool        string `json:"tool"`
			Version     string `json:"version"`
			SpecVersion string `json:"spec_version"`
			Commit      string `json:"commit"`
		}
		_ = json.Unmarshal(versionJSON, &vdoc)
		m.Tool = vdoc.Tool
		m.BinaryVersion = vdoc.Version
		m.SpecVersion = vdoc.SpecVersion
		m.GitSHA = vdoc.Commit
	}

	capsJSON, capsErr := runBinaryCommand(binary, []string{"capabilities"})
	var cFindings []conformFinding
	if capsErr != nil {
		cFindings = []conformFinding{fail("§17.7", "capabilities failed: %v", capsErr)}
	} else {
		cFindings = validateCapabilitiesDoc(capsJSON)
		sum := sha256.Sum256(capsJSON)
		m.CapabilitiesSHA256 = hex.EncodeToString(sum[:])
	}

	statusJSON, statusErr := runBinaryCommand(binary, []string{"status", "--format", "json"})
	if statusErr != nil {
		statusJSON, statusErr = runBinaryCommand(binary, []string{"status"})
	}
	var sFindings []conformFinding
	if statusErr != nil {
		sFindings = []conformFinding{fail("§17.7", "status command failed: %v", statusErr)}
	} else {
		sFindings = validateStatusDoc(statusJSON)
	}

	// capsUnreachable is true when the capabilities command couldn't even be
	// run/parsed at all (§17.7) — a total failure that subsumes every
	// narrower capabilities-doc check below.
	capsUnreachable := hasFailWithReq(cFindings, "§17.7")

	req1 := statusPass
	// Requirement 1 (protocol declaration) spans both documents: spec_version
	// shape (version doc) and the capabilities doc's own protocol-null check
	// (§12.2, gated on multi_protocol — see validateCapabilitiesDoc).
	if hasFail(vFindings) || capsUnreachable || hasFailWithReq(cFindings, "§12.2") {
		req1 = statusFail
	}

	req6 := statusPass
	// Requirement 6 (application interface) is specifically the adapt=false
	// check (§17.6) — filtered so an unrelated capabilities-doc FAIL (e.g.
	// Requirement 17's retirement check) doesn't misreport this requirement,
	// but still FAILs if the capabilities doc couldn't be fetched at all.
	if capsUnreachable || hasFailWithReq(cFindings, "§17.6") {
		req6 = statusFail
	}
	req7 := statusPass
	if hasFail(vFindings) || hasFail(cFindings) || hasFail(sFindings) {
		req7 = statusFail
	}
	req12 := statusShapeOnly
	if hasFail(vFindings) {
		req12 = statusFail
	}
	req17 := statusPass
	if capsUnreachable || hasFailWithReq(cFindings, "§3.2") {
		req17 = statusFail
	}

	m.Requirements = []manifestRequirement{
		{1, req1, verifierRelayConform},
		{2, statusNotObservable, verifierTestSuite},
		{3, statusNotObservable, verifierTestSuite},
		{4, statusNotObservable, verifierTestSuite},
		{5, statusNotObservable, verifierTestSuite},
		{6, req6, verifierRelayConform},
		{7, req7, verifierRelayConform},
		{8, statusNotObservable, verifierTestSuite},
		{9, statusNotObservable, verifierTestSuite},
		{10, statusNotObservable, verifierTestSuite},
		{11, statusNotObservable, verifierTestSuite},
		{12, req12, verifierRelayConform},
		{13, statusNotObservable, verifierBuildAndTest},
		{14, statusNotObservable, verifierCI},
		{15, statusNotObservable, verifierManifestSplit},
		{16, statusNotObservable, verifierCI},
		{17, req17, verifierRelayConform},
		{18, statusNotObservable, verifierCI},
	}

	m.Overall = statusPass
	for _, r := range m.Requirements {
		if r.Status == statusFail {
			m.Overall = statusFail
			break
		}
	}
	return m
}

// attestationDigest is one subject's content-addressed digest in an in-toto
// Statement (spec §20.6).
type attestationDigest struct {
	SHA256 string `json:"sha256"`
}

// attestationSubject identifies the artifact an attestation is about.
type attestationSubject struct {
	Name   string            `json:"name"`
	Digest attestationDigest `json:"digest"`
}

// attestationPredicate is the relay-conform-attestation/1 predicate body
// (spec §20.6).
type attestationPredicate struct {
	SpecVersion           string          `json:"spec_version"`
	VectorsVersion        string          `json:"vectors_version"`
	ConformanceManifest   conformManifest `json:"conformance_manifest"`
	SafetyEvidenceSummary interface{}     `json:"safety_evidence_summary"`
	Signed                bool            `json:"signed"`
}

// conformAttestation is an in-toto Statement carrying a relay-conform-attestation/1
// predicate (spec §20.6). It is unsigned: this type has no signature field,
// and Signed is always false — signing and publication are explicitly out of
// scope for RELAY's own reference tooling (spec §20.6).
type conformAttestation struct {
	Type          string               `json:"_type"`
	PredicateType string               `json:"predicateType"`
	Subject       []attestationSubject `json:"subject"`
	Predicate     attestationPredicate `json:"predicate"`
}

// buildAttestation generates an unsigned relay-conform-attestation/1
// predicate (spec §20.6) for one binary: an in-toto Statement bundling the
// §17.2 conformance manifest for that binary, this tool's own embedded
// vectors_version (§15.8 — NOT observed from the attested binary, which a
// black-box invocation cannot introspect), and the attested binary's own
// SHA-256 as the subject digest.
//
//fusa:req REQ-RELAY-098
func buildAttestation(binary string) (conformAttestation, error) {
	raw, err := os.ReadFile(binary)
	if err != nil {
		return conformAttestation{}, fmt.Errorf("read binary: %w", err)
	}
	sum := sha256.Sum256(raw)

	m := buildManifest(binary)

	vectorsVersion := ""
	if vm, err := relay.ParsedVectorsManifest(); err == nil {
		vectorsVersion = vm.VectorsVersion
	}

	return conformAttestation{
		Type:          "https://in-toto.io/Statement/v1",
		PredicateType: "https://relay.dev/attestation/relay-conform/1",
		Subject: []attestationSubject{
			{Name: m.Tool, Digest: attestationDigest{SHA256: hex.EncodeToString(sum[:])}},
		},
		Predicate: attestationPredicate{
			SpecVersion:           m.SpecVersion,
			VectorsVersion:        vectorsVersion,
			ConformanceManifest:   m,
			SafetyEvidenceSummary: nil,
			Signed:                false,
		},
	}, nil
}

// hasFail reports whether any finding in fs is FAIL-severity.
func hasFail(fs []conformFinding) bool {
	for _, f := range fs {
		if f.Severity == sevFail {
			return true
		}
	}
	return false
}

// hasFailWithReq reports whether any finding in fs is FAIL-severity and cites
// req (spec section) exactly.
func hasFailWithReq(fs []conformFinding, req string) bool {
	for _, f := range fs {
		if f.Severity == sevFail && f.Req == req {
			return true
		}
	}
	return false
}

func printConformText(w io.Writer, cr conformResult) {
	for _, f := range cr.Findings {
		fmt.Fprintf(w, "%-4s  %s  %s\n", f.Severity, f.Req, f.Message)
	}
	fmt.Fprintln(w, strings.Repeat("─", 60))
	fmt.Fprintf(w, "RESULT: %s  binary=%s\n", cr.Result, cr.Binary)
}

// runBinaryCommand executes binary with args and returns stdout.
// Returns an error if the binary exits non-zero or times out.
func runBinaryCommand(binary string, args []string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	var out bytes.Buffer
	cmd := exec.CommandContext(ctx, binary, args...)
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

// schemaCheck validates raw JSON output against the named embedded schema and
// returns one finding per result: a single FAIL for malformed JSON or a missing
// schema, one FAIL per schema violation, or a single PASS when the document
// conforms. ref is the spec section to attribute findings to.
//
//fusa:req REQ-RELAY-058
func schemaCheck(name, ref string, data []byte) (doc map[string]interface{}, findings []conformFinding) {
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, []conformFinding{fail(ref, "%s output is not valid JSON: %v", name, err)}
	}
	schemaJSON, err := relay.Schema(name)
	if err != nil {
		return doc, []conformFinding{fail(ref, "no embedded schema %q: %v", name, err)}
	}
	var asAny interface{}
	_ = json.Unmarshal(data, &asAny)
	violations := validateSchema(schemaJSON, asAny)
	if len(violations) == 0 {
		return doc, []conformFinding{pass(ref, fmt.Sprintf("conforms to %s schema", schemaTitle(schemaJSON)))}
	}
	for _, v := range violations {
		findings = append(findings, fail(ref, "%s schema: %s", schemaTitle(schemaJSON), v))
	}
	return doc, findings
}

// validateVersionDoc validates a version --format json response per §12.1.
//
//fusa:req REQ-RELAY-053
//fusa:req REQ-RELAY-047
//fusa:req REQ-RELAY-079
func validateVersionDoc(data []byte) []conformFinding {
	doc, fs := schemaCheck("cli-version", "§12.1", data)
	if doc == nil {
		return fs
	}
	// protocol / protocol_int are required for single-protocol implementations;
	// null or absent is acceptable for multi-protocol tooling (WARN not FAIL).
	if doc["protocol"] == nil {
		fs = append(fs, warn("§12.1", "version doc: protocol is null (acceptable for multi-protocol tools)"))
	}
	return fs
}

// validateCapabilitiesDoc validates a capabilities response per §12.2.
//
//fusa:req REQ-RELAY-054
//fusa:req REQ-RELAY-048
//fusa:req REQ-RELAY-097
func validateCapabilitiesDoc(data []byte) []conformFinding {
	doc, fs := schemaCheck("cli-capabilities", "§12.2", data)
	if doc == nil {
		return fs
	}

	// The schema only asserts that commands contains "version"; §17.7 also
	// requires "capabilities" and "status".
	if cmds, ok := doc["commands"].([]interface{}); ok {
		cmdSet := map[string]bool{}
		for _, c := range cmds {
			if s, ok := c.(string); ok {
				cmdSet[s] = true
			}
		}
		for _, required := range []string{"capabilities", "status"} {
			if !cmdSet[required] {
				fs = append(fs, fail("§17.7", "capabilities doc: commands does not include %q", required))
			}
		}
	}

	// multi_protocol (§12.2, §17 Requirements 1 and 6) self-declares that this
	// binary is inherently multi-protocol tooling, not a single-protocol
	// implementation. Absent/false is the default (single-protocol).
	multiProtocol, _ := doc["multi_protocol"].(bool)

	// A null protocol/protocol_int is only legitimate for a self-declared
	// multi-protocol tool (§10.3, §17 Requirement 1); otherwise it's a real gap.
	if !multiProtocol && doc["protocol"] == nil {
		fs = append(fs, fail("§12.2", "capabilities doc: protocol is null but multi_protocol is not true"))
	}

	// adapt=false is only legitimate for a self-declared multi-protocol tool —
	// §10.3 scopes Adapt() to protocol packages, so a multi-protocol aggregator
	// has no per-protocol Adapt() to export (§17 Requirement 6).
	if adapt, ok := doc["adapt"].(bool); ok && !adapt {
		if multiProtocol {
			// Legitimate: no per-protocol Adapt() applies to this tool.
		} else {
			fs = append(fs, fail("§17.6", "capabilities doc: adapt=false (Adapt() not exported) but multi_protocol is not true"))
		}
	}

	// §3.2 / §17 Requirement 17: a capabilities doc MUST NOT still advertise a
	// retired capability at or past its removal version.
	if specVersion, ok := doc["spec_version"].(string); ok {
		fs = append(fs, checkRetiredCapabilities(specVersion, doc, loadRetiredCapabilities())...)
	}

	return fs
}

// retiredEntry is one entry in spec/version.json's retired[]/deprecated[]
// arrays (§3.2).
type retiredEntry struct {
	Name    string `json:"name"`
	Since   string `json:"since"`
	Removal string `json:"removal"`
	Reason  string `json:"reason"`
}

// loadRetiredCapabilities parses RELAY's own embedded spec/version.json and
// returns its retired[] entries (§3.2). Returns nil, not an error, if the
// embedded document is unparseable — this check degrades to a no-op rather
// than failing every conform invocation on an unrelated defect.
func loadRetiredCapabilities() []retiredEntry {
	raw, err := relay.Evidence("version")
	if err != nil {
		return nil
	}
	var meta struct {
		Retired []retiredEntry `json:"retired"`
	}
	if err := json.Unmarshal(raw, &meta); err != nil {
		return nil
	}
	return meta.Retired
}

// parseSpecVersion parses a "MAJOR.MINOR" spec version string.
func parseSpecVersion(s string) (major, minor int, ok bool) {
	parts := strings.SplitN(s, ".", 2)
	if len(parts) != 2 {
		return 0, 0, false
	}
	maj, err1 := strconv.Atoi(parts[0])
	min, err2 := strconv.Atoi(parts[1])
	if err1 != nil || err2 != nil {
		return 0, 0, false
	}
	return maj, min, true
}

// specVersionAtLeast reports whether a >= b, both "MAJOR.MINOR" strings.
// Returns false (fail-safe: does not trigger retirement enforcement) if
// either string cannot be parsed.
func specVersionAtLeast(a, b string) bool {
	aMaj, aMin, aOK := parseSpecVersion(a)
	bMaj, bMin, bOK := parseSpecVersion(b)
	if !aOK || !bOK {
		return false
	}
	if aMaj != bMaj {
		return aMaj > bMaj
	}
	return aMin >= bMin
}

// checkRetiredCapabilities cross-references a binary's declared spec_version
// and capabilities (features/commands) against a retired[] list (§3.2, §17
// Requirement 17), FAILing any capability still advertised at or past its
// recorded removal version.
//
//fusa:req REQ-RELAY-099
func checkRetiredCapabilities(specVersion string, capsDoc map[string]interface{}, retired []retiredEntry) []conformFinding {
	if specVersion == "" || len(retired) == 0 {
		return nil
	}
	named := map[string]bool{}
	if cmds, ok := capsDoc["commands"].([]interface{}); ok {
		for _, c := range cmds {
			if s, ok := c.(string); ok {
				named[s] = true
			}
		}
	}
	if feats, ok := capsDoc["features"].([]interface{}); ok {
		for _, f := range feats {
			if s, ok := f.(string); ok {
				named[s] = true
			}
		}
	}
	var fs []conformFinding
	for _, r := range retired {
		if !specVersionAtLeast(specVersion, r.Removal) {
			continue
		}
		if named[r.Name] {
			fs = append(fs, fail("§3.2", "capabilities doc: declares retired capability %q, which MUST be removed at spec_version %s (removal version %s)", r.Name, specVersion, r.Removal))
		}
	}
	return fs
}

// validateStatusDoc validates a status response per §12.3.
//
//fusa:req REQ-RELAY-055
//fusa:req REQ-RELAY-049
func validateStatusDoc(data []byte) []conformFinding {
	_, fs := schemaCheck("cli-status", "§12.3", data)
	return fs
}
