// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package main

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"strings"
	"testing"
)

//fusa:test REQ-RELAY-063
//fusa:test REQ-RELAY-077
//fusa:test REQ-RELAY-078
func TestRunSBOMJSON(t *testing.T) {
	var out, errb bytes.Buffer
	if err := runSBOM(&out, &errb, []string{"--format", "json"}); err != nil {
		t.Fatalf("runSBOM: %v", err)
	}
	var doc sbomDoc
	if err := json.Unmarshal(out.Bytes(), &doc); err != nil {
		t.Fatalf("sbom json: %v\n%s", err, out.String())
	}
	if doc.Format != "relay-sbom/1" {
		t.Errorf("format = %q, want relay-sbom/1", doc.Format)
	}
	if doc.Tool != "relay" || doc.GoVersion == "" {
		t.Errorf("sbom missing tool/go version: %+v", doc)
	}
	if doc.Components == nil {
		t.Error("components must be a (possibly empty) array, not null")
	}
}

//fusa:test REQ-RELAY-063
func TestRunSBOMUnknownFormat(t *testing.T) {
	var out, errb bytes.Buffer
	if err := runSBOM(&out, &errb, []string{"--format", "xml"}); err == nil {
		t.Error("expected error for unknown format")
	}
}

//fusa:test REQ-RELAY-064
func TestRunSafetyCaseJSON(t *testing.T) {
	var out, errb bytes.Buffer
	if err := runSafetyCase(&out, &errb, []string{"--format", "json"}); err != nil {
		t.Fatalf("runSafetyCase: %v", err)
	}
	var doc safetyCaseDoc
	if err := json.Unmarshal(out.Bytes(), &doc); err != nil {
		t.Fatalf("safety-case json: %v\n%s", err, out.String())
	}
	if doc.Requirements.Total < 60 {
		t.Errorf("requirements total = %d, want >= 60", doc.Requirements.Total)
	}
	if doc.Hazards.Total != 6 || doc.Hazards.WorstASIL != "ASIL-C" {
		t.Errorf("hazards = %d worst %q, want 6 ASIL-C", doc.Hazards.Total, doc.Hazards.WorstASIL)
	}
	if doc.Threats.Total != 5 {
		t.Errorf("threats total = %d, want 5", doc.Threats.Total)
	}
	if doc.Threats.WorstRisk != "high" {
		t.Errorf("worst risk = %q, want high", doc.Threats.WorstRisk)
	}
}

//fusa:test REQ-RELAY-063
func TestRunSBOMText(t *testing.T) {
	var out, errb bytes.Buffer
	if err := runSBOM(&out, &errb, []string{"--format", "text"}); err != nil {
		t.Fatalf("runSBOM text: %v", err)
	}
	s := out.String()
	if !strings.Contains(s, "module:") || !strings.Contains(s, "go:") {
		t.Errorf("sbom text missing fields:\n%s", s)
	}
}

//fusa:test REQ-RELAY-064
func TestRunSafetyCaseUnknownFormat(t *testing.T) {
	var out, errb bytes.Buffer
	if err := runSafetyCase(&out, &errb, []string{"--format", "pdf"}); err == nil {
		t.Error("expected error for unknown format")
	}
}

//fusa:test REQ-RELAY-064
func TestRunSafetyCaseMarkdown(t *testing.T) {
	var out, errb bytes.Buffer
	if err := runSafetyCase(&out, &errb, []string{"--format", "markdown"}); err != nil {
		t.Fatalf("runSafetyCase: %v", err)
	}
	if !strings.Contains(out.String(), "| Evidence | Summary |") {
		t.Errorf("markdown safety case missing table:\n%s", out.String())
	}
}

//fusa:test REQ-RELAY-064
func TestRunSafetyCaseText(t *testing.T) {
	var out, errb bytes.Buffer
	if err := runSafetyCase(&out, &errb, nil); err != nil {
		t.Fatalf("runSafetyCase: %v", err)
	}
	if !strings.Contains(out.String(), "RELAY safety case") {
		t.Errorf("text safety case unexpected:\n%s", out.String())
	}
}

//fusa:test REQ-RELAY-065
//fusa:test REQ-RELAY-080
func TestWriteAuditPackManifestHashes(t *testing.T) {
	var buf bytes.Buffer
	n, err := writeAuditPack(&buf)
	if err != nil {
		t.Fatalf("writeAuditPack: %v", err)
	}
	if n < 10 {
		t.Errorf("expected several artifacts, got %d", n)
	}

	zr, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		t.Fatalf("open zip: %v", err)
	}

	// Read every entry's bytes.
	contents := map[string][]byte{}
	for _, f := range zr.File {
		rc, err := f.Open()
		if err != nil {
			t.Fatalf("open %s: %v", f.Name, err)
		}
		b, _ := io.ReadAll(rc)
		_ = rc.Close()
		contents[f.Name] = b
	}

	manifestRaw, ok := contents["manifest.json"]
	if !ok {
		t.Fatal("manifest.json missing from audit pack")
	}
	var manifest struct {
		Files []manifestEntry `json:"files"`
	}
	if err := json.Unmarshal(manifestRaw, &manifest); err != nil {
		t.Fatalf("manifest json: %v", err)
	}
	if len(manifest.Files) != n {
		t.Errorf("manifest lists %d files, pack has %d artifacts", len(manifest.Files), n)
	}
	for _, e := range manifest.Files {
		data, ok := contents[e.Name]
		if !ok {
			t.Errorf("manifest references missing entry %q", e.Name)
			continue
		}
		sum := sha256.Sum256(data)
		if got := hex.EncodeToString(sum[:]); got != e.SHA256 {
			t.Errorf("%s: manifest hash %s != actual %s", e.Name, e.SHA256, got)
		}
		if e.Bytes != len(data) {
			t.Errorf("%s: manifest bytes %d != actual %d", e.Name, e.Bytes, len(data))
		}
	}
}

//fusa:test REQ-RELAY-065
func TestRunAuditPackWritesFile(t *testing.T) {
	out := t.TempDir() + "/pack.zip"
	var so, se bytes.Buffer
	if err := runAuditPack(&so, &se, []string{"--output", out}); err != nil {
		t.Fatalf("runAuditPack: %v", err)
	}
	if !strings.Contains(so.String(), "wrote") {
		t.Errorf("expected confirmation, got %q", so.String())
	}
}
