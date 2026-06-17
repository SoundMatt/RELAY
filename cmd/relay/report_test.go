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

//fusa:test REQ-RELAY-062
func TestRunReportJSON(t *testing.T) {
	bin := buildTestBinary(t)
	var out, errb bytes.Buffer
	if err := runReport(&out, &errb, []string{"--format", "json", bin}); err != nil {
		t.Fatalf("runReport: %v", err)
	}
	var doc reportDoc
	if err := json.Unmarshal(out.Bytes(), &doc); err != nil {
		t.Fatalf("report json: %v\n%s", err, out.String())
	}
	if len(doc.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(doc.Entries))
	}
	e := doc.Entries[0]
	// relay conforms to itself but emits WARNs (null protocol, adapt=false).
	if e.Result != sevWarn {
		t.Errorf("relay self-report result = %s, want WARN", e.Result)
	}
	if e.Fail != 0 {
		t.Errorf("relay self-report should have 0 FAIL findings, got %d", e.Fail)
	}
	if doc.Result != sevWarn {
		t.Errorf("overall result = %s, want WARN", doc.Result)
	}
}

//fusa:test REQ-RELAY-062
func TestRunReportText(t *testing.T) {
	bin := buildTestBinary(t)
	var out, errb bytes.Buffer
	if err := runReport(&out, &errb, []string{bin}); err != nil {
		t.Fatalf("runReport: %v", err)
	}
	s := out.String()
	if !strings.Contains(s, "BINARY") || !strings.Contains(s, "RESULT:") {
		t.Errorf("text report missing header/summary:\n%s", s)
	}
}

//fusa:test REQ-RELAY-062
func TestRunReportMarkdown(t *testing.T) {
	bin := buildTestBinary(t)
	var out, errb bytes.Buffer
	if err := runReport(&out, &errb, []string{"--format", "markdown", bin}); err != nil {
		t.Fatalf("runReport: %v", err)
	}
	s := out.String()
	if !strings.Contains(s, "| Binary | Tool |") || !strings.Contains(s, "|---|") {
		t.Errorf("markdown report missing GFM table:\n%s", s)
	}
}

//fusa:test REQ-RELAY-062
func TestRunReportHTML(t *testing.T) {
	bin := buildTestBinary(t)
	var out, errb bytes.Buffer
	if err := runReport(&out, &errb, []string{"--format", "html", bin}); err != nil {
		t.Fatalf("runReport: %v", err)
	}
	s := out.String()
	if !strings.HasPrefix(s, "<!DOCTYPE html>") || !strings.Contains(s, "<table>") {
		t.Errorf("html report not a self-contained document:\n%s", s[:min(120, len(s))])
	}
}

//fusa:test REQ-RELAY-062
func TestRunReportScan(t *testing.T) {
	bin := buildTestBinary(t)
	t.Setenv("PATH", filepath.Dir(bin))
	var out, errb bytes.Buffer
	if err := runReport(&out, &errb, []string{"--scan", "--match", "relay*", "--format", "json"}); err != nil {
		t.Fatalf("runReport --scan: %v", err)
	}
	var doc reportDoc
	if err := json.Unmarshal(out.Bytes(), &doc); err != nil {
		t.Fatalf("scan json: %v", err)
	}
	if len(doc.Entries) != 1 || doc.Entries[0].Tool != "relay" {
		t.Errorf("scan report should include the relay binary, got %+v", doc.Entries)
	}
}

//fusa:test REQ-RELAY-062
func TestRunReportStrictFailsOnWarn(t *testing.T) {
	bin := buildTestBinary(t)
	var out, errb bytes.Buffer
	err := runReport(&out, &errb, []string{"--strict", "--format", "json", bin})
	// relay emits WARNs; with --strict those become FAIL, so exit 1.
	var code exitCode
	if !errors.As(err, &code) || int(code) != 1 {
		t.Errorf("--strict on a WARN implementation should exit 1, got %v", err)
	}
}

//fusa:test REQ-RELAY-062
func TestRunReportNoArgs(t *testing.T) {
	var out, errb bytes.Buffer
	err := runReport(&out, &errb, nil)
	var code exitCode
	if !errors.As(err, &code) || int(code) != 2 {
		t.Errorf("report with no args should exit 2, got %v", err)
	}
}

//fusa:test REQ-RELAY-062
func TestRunReportUnknownFormat(t *testing.T) {
	bin := buildTestBinary(t)
	var out, errb bytes.Buffer
	if err := runReport(&out, &errb, []string{"--format", "pdf", bin}); err == nil {
		t.Error("expected error for unknown format")
	}
}
