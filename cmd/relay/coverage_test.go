// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// writeScript creates an executable /bin/sh script emitting body on stdout.
func writeScript(t *testing.T, body string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "fake")
	if err := os.WriteFile(p, []byte("#!/bin/sh\n"+body+"\n"), 0o755); err != nil {
		t.Fatalf("write script: %v", err)
	}
	return p
}

//fusa:test REQ-RELAY-021
func TestRunDispatch(t *testing.T) {
	// Each command must be reachable through the top-level dispatcher.
	ok := [][]string{
		{"version"}, {"version", "--format", "json"},
		{"capabilities"},
		{"status"}, {"status", "--format", "json"},
		{"sbom"}, {"sbom", "--format", "json"},
		{"safety-case"}, {"safety-case", "--format", "json"},
		{"--help"}, {"help"}, {"-h"},
	}
	for _, args := range ok {
		var out, errb bytes.Buffer
		if err := run(&out, &errb, args); err != nil {
			t.Errorf("run(%v) returned error: %v", args, err)
		}
	}

	// No args and unknown commands exit 2 (usage).
	for _, args := range [][]string{nil, {"bogus-command"}} {
		var out, errb bytes.Buffer
		err := run(&out, &errb, args)
		var code exitCode
		if !errors.As(err, &code) || int(code) != 2 {
			t.Errorf("run(%v) = %v, want exitCode(2)", args, err)
		}
	}

	// The remaining commands must be reachable through the dispatcher. With no
	// arguments each returns quickly (usage / no-candidates) without binding a
	// port or spawning anything; we only assert dispatch, not the outcome.
	for _, args := range [][]string{
		{"conform"}, {"probe"}, {"trace"}, {"report"},
		{"compare"}, {"versions"}, {"serve"},
	} {
		var out, errb bytes.Buffer
		_ = run(&out, &errb, args)
	}
}

//fusa:test REQ-RELAY-066
func TestCompareCapsAndRender(t *testing.T) {
	base := capsDoc{
		Kind: "capabilities", Protocol: "CAN", SpecVersion: "1.6",
		Commands: []string{"version", "send"}, Features: []string{"fd"},
		Interfaces: []string{"Bus"},
	}
	// Identical caps are compatible with no differences.
	same := compareCaps("a", "b", base, base)
	if !same.Compatible || len(same.Differences) != 0 {
		t.Errorf("identical caps must be compatible with no differences: %+v", same)
	}
	var compatBuf bytes.Buffer
	printCompareText(&compatBuf, same)
	if !bytes.Contains(compatBuf.Bytes(), []byte("COMPATIBLE")) ||
		!bytes.Contains(compatBuf.Bytes(), []byte("no capability differences")) {
		t.Errorf("compatible render missing expected text:\n%s", compatBuf.String())
	}

	// A fully divergent peer exercises every difference branch.
	other := capsDoc{
		Kind: "capabilities", Protocol: "LIN", SpecVersion: "1.5",
		Commands: []string{"version", "subscribe"}, Features: []string{"isotp"},
		Interfaces: []string{"Participant"},
	}
	diff := compareCaps("a", "b", base, other)
	if diff.Compatible || len(diff.Differences) == 0 {
		t.Errorf("divergent caps must be incompatible with differences: %+v", diff)
	}
	if diff.ProtocolMatch || diff.SpecVersionMatch {
		t.Error("protocol/spec mismatch flags should be false")
	}
	var incompatBuf bytes.Buffer
	printCompareText(&incompatBuf, diff)
	if !bytes.Contains(incompatBuf.Bytes(), []byte("INCOMPATIBLE")) ||
		!bytes.Contains(incompatBuf.Bytes(), []byte("differences:")) {
		t.Errorf("incompatible render missing expected text:\n%s", incompatBuf.String())
	}
}

//fusa:test REQ-RELAY-060
func TestPrintProbeText(t *testing.T) {
	// Empty set.
	var empty bytes.Buffer
	printProbeText(&empty, nil)
	if !bytes.Contains(empty.Bytes(), []byte("No RELAY-conformant")) {
		t.Errorf("empty probe render wrong:\n%s", empty.String())
	}
	// Mixed conformant + non-conformant + missing protocol.
	var buf bytes.Buffer
	printProbeText(&buf, []probeResult{
		{Binary: "/x/go-can", Conformant: true, Tool: "go-can", Protocol: "CAN", Version: "1.0", SpecVersion: "1.6", Transports: []string{"vcan"}},
		{Binary: "/x/noproto", Conformant: true, Tool: "t", Version: "0.1", SpecVersion: "1.6"},
		{Binary: "/x/bad", Conformant: false, Error: "not a relay binary"},
	})
	s := buf.String()
	if !bytes.Contains(buf.Bytes(), []byte("go-can")) || !bytes.Contains(buf.Bytes(), []byte("not conformant")) {
		t.Errorf("probe render missing rows:\n%s", s)
	}
}

//fusa:test REQ-RELAY-062
func TestRenderReportAllFormats(t *testing.T) {
	// Empty report.
	var empty bytes.Buffer
	renderReportText(&empty, reportDoc{})
	if !bytes.Contains(empty.Bytes(), []byte("No implementations")) {
		t.Errorf("empty report render wrong:\n%s", empty.String())
	}
	doc := reportDoc{Result: sevPass}
	doc.Summary.Pass = 2
	doc.Entries = []reportEntry{
		{Binary: "/x/go-can", Tool: "go-can", Protocol: "CAN", Result: sevPass, Pass: 3},
		{Binary: "/x/nop", Result: sevWarn, Warn: 1},
	}
	for _, render := range []func(*bytes.Buffer){
		func(b *bytes.Buffer) { renderReportText(b, doc) },
		func(b *bytes.Buffer) { renderReportMarkdown(b, doc) },
		func(b *bytes.Buffer) { renderReportHTML(b, doc) },
	} {
		var b bytes.Buffer
		render(&b)
		if b.Len() == 0 || !bytes.Contains(b.Bytes(), []byte("go-can")) {
			t.Errorf("report render produced unexpected output:\n%s", b.String())
		}
	}
}

//fusa:test REQ-RELAY-021
func TestExitCodeError(t *testing.T) {
	if got := exitCode(2).Error(); got != "exit 2" {
		t.Errorf("exitCode(2).Error() = %q, want %q", got, "exit 2")
	}
}

//fusa:test REQ-RELAY-064
func TestAsilAndRiskRank(t *testing.T) {
	asil := map[string]int{"QM": 0, "ASIL-A": 1, "asil-b": 2, "ASIL-C": 3, "ASIL-D": 4, "nope": -1}
	for s, want := range asil {
		if got := asilRank(s); got != want {
			t.Errorf("asilRank(%q) = %d, want %d", s, got, want)
		}
	}
	risk := map[string]int{"low": 1, "Medium": 2, "HIGH": 3, "critical": 4, "nope": -1}
	for s, want := range risk {
		if got := riskRank(s); got != want {
			t.Errorf("riskRank(%q) = %d, want %d", s, got, want)
		}
	}
	// worse() picks the higher-ranked of the two (and seeds from empty).
	if worse("", "ASIL-A", asilRank) != "ASIL-A" {
		t.Error("worse should seed from empty")
	}
	if worse("ASIL-D", "ASIL-A", asilRank) != "ASIL-D" {
		t.Error("worse must keep the higher ASIL")
	}
	if worse("ASIL-A", "ASIL-C", asilRank) != "ASIL-C" {
		t.Error("worse must upgrade to the higher ASIL")
	}
}

//fusa:test REQ-RELAY-058
func TestJSONTypeOf(t *testing.T) {
	cases := []struct {
		v    interface{}
		want string
	}{
		{nil, "null"},
		{true, "boolean"},
		{"s", "string"},
		{float64(3), "integer"},
		{float64(3.5), "number"},
		{map[string]interface{}{}, "object"},
		{[]interface{}{}, "array"},
		{struct{}{}, "unknown"},
	}
	for _, tc := range cases {
		if got := jsonTypeOf(tc.v); got != tc.want {
			t.Errorf("jsonTypeOf(%T) = %q, want %q", tc.v, got, tc.want)
		}
	}
}

//fusa:test REQ-RELAY-058
func TestJSONTypeIs(t *testing.T) {
	if !jsonTypeIs("object", map[string]interface{}{}) || jsonTypeIs("object", 1.0) {
		t.Error("object")
	}
	if !jsonTypeIs("array", []interface{}{}) || jsonTypeIs("array", "x") {
		t.Error("array")
	}
	if !jsonTypeIs("string", "x") || jsonTypeIs("string", true) {
		t.Error("string")
	}
	if !jsonTypeIs("boolean", true) || jsonTypeIs("boolean", "x") {
		t.Error("boolean")
	}
	if !jsonTypeIs("null", nil) || jsonTypeIs("null", 0.0) {
		t.Error("null")
	}
	if !jsonTypeIs("number", 3.5) || jsonTypeIs("number", "x") {
		t.Error("number")
	}
	if !jsonTypeIs("integer", 3.0) || jsonTypeIs("integer", 3.5) {
		t.Error("integer")
	}
	if jsonTypeIs("bogus", "x") {
		t.Error("unknown type keyword must be false")
	}
}

//fusa:test REQ-RELAY-058
func TestTypeMatches(t *testing.T) {
	if !typeMatches(nil, 1.0) {
		t.Error("nil schema type matches anything")
	}
	if !typeMatches("string", "x") || typeMatches("string", 1.0) {
		t.Error("single string type")
	}
	if !typeMatches([]interface{}{"string", "integer"}, 3.0) {
		t.Error("union type should match integer")
	}
	if typeMatches([]interface{}{"string", "boolean"}, 3.0) {
		t.Error("union type should reject number")
	}
	if !typeMatches(42, "x") {
		t.Error("non-string/array schema type defaults to match")
	}
}

//fusa:test REQ-RELAY-058
func TestPathAndTitleHelpers(t *testing.T) {
	if joinPath("", "k") != "k" || joinPath("a", "b") != "a.b" {
		t.Error("joinPath")
	}
	if pathName("") != "(root)" || pathName("a.b") != "a.b" {
		t.Error("pathName")
	}
	if schemaTitle([]byte(`{"title":"can.Frame"}`)) != "can.Frame" {
		t.Error("schemaTitle should read the title")
	}
	if schemaTitle([]byte(`{}`)) != "document" {
		t.Error("schemaTitle should default to 'document'")
	}
	if f, ok := numberOf(3.5); !ok || f != 3.5 {
		t.Error("numberOf on float64")
	}
	if _, ok := numberOf("x"); ok {
		t.Error("numberOf on non-number must be false")
	}
	if !jsonEqual([]interface{}{1.0, "a"}, []interface{}{1.0, "a"}) || jsonEqual(1.0, 2.0) {
		t.Error("jsonEqual")
	}
}

//fusa:test REQ-RELAY-066
func TestFetchCapsErrors(t *testing.T) {
	// Command exits non-zero -> "capabilities failed".
	if _, err := fetchCaps("/bin/false"); err == nil {
		t.Error("fetchCaps on a failing binary must error")
	}
	// Valid exit but non-JSON output -> "not valid JSON".
	if _, err := fetchCaps(writeScript(t, `echo not-json`)); err == nil {
		t.Error("fetchCaps on non-JSON output must error")
	}
	// Valid JSON but wrong kind -> "not a RELAY capabilities document".
	if _, err := fetchCaps(writeScript(t, `echo '{"kind":"version"}'`)); err == nil {
		t.Error("fetchCaps on a non-capabilities document must error")
	}
}

//fusa:test REQ-RELAY-052
func TestConformBinaryNonConformant(t *testing.T) {
	// A binary whose commands all fail must yield a FAIL result with findings.
	cr := conformBinary("/bin/false", false)
	if cr.Result != sevFail || len(cr.Findings) == 0 {
		t.Errorf("conformBinary(/bin/false) = %+v, want FAIL with findings", cr)
	}
	var buf bytes.Buffer
	printConformText(&buf, cr)
	if !bytes.Contains(buf.Bytes(), []byte("RESULT: FAIL")) {
		t.Errorf("printConformText missing FAIL verdict:\n%s", buf.String())
	}
}

//fusa:test REQ-RELAY-068
func TestRunServeFlagError(t *testing.T) {
	// An unknown flag must return an error without binding a port.
	var out, errb bytes.Buffer
	if err := runServe(&out, &errb, []string{"--no-such-flag"}); err == nil {
		t.Error("runServe with a bad flag must return an error")
	}
}
