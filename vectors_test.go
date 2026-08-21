// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package relay

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io/fs"
	"strings"
	"testing"
)

//fusa:test REQ-RELAY-057
func TestVectorNames(t *testing.T) {
	names, err := VectorNames()
	if err != nil {
		t.Fatalf("VectorNames: %v", err)
	}
	if len(names) == 0 {
		t.Fatal("expected at least one embedded golden vector")
	}
	for _, n := range names {
		b, err := Vector(n)
		if err != nil {
			t.Errorf("Vector(%q): %v", n, err)
			continue
		}
		var v map[string]any
		if err := json.Unmarshal(b, &v); err != nil {
			t.Errorf("Vector(%q) is not valid JSON: %v", n, err)
		}
	}
}

//fusa:test REQ-RELAY-057
func TestVectorUnknown(t *testing.T) {
	if _, err := Vector("nope-does-not-exist"); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("Vector(unknown) error = %v, want fs.ErrNotExist", err)
	}
}

// --- tests for the vector distribution manifest (spec §15.8, Requirement 16) ---

//fusa:test REQ-RELAY-096
func TestVectorNamesExcludesManifest(t *testing.T) {
	names, err := VectorNames()
	if err != nil {
		t.Fatalf("VectorNames: %v", err)
	}
	for _, n := range names {
		if n == "vectors_manifest" {
			t.Fatal("VectorNames must exclude vectors_manifest.json — it is metadata about the vector set, not a golden vector")
		}
	}
}

//fusa:test REQ-RELAY-096
func TestVectorsManifestValid(t *testing.T) {
	raw, err := VectorsManifest()
	if err != nil {
		t.Fatalf("VectorsManifest: %v", err)
	}
	var asAny any
	if err := json.Unmarshal(raw, &asAny); err != nil {
		t.Fatalf("vectors_manifest.json is not valid JSON: %v", err)
	}

	m, err := ParsedVectorsManifest()
	if err != nil {
		t.Fatalf("ParsedVectorsManifest: %v", err)
	}
	if m.Kind != "relay-vectors-manifest" {
		t.Errorf("Kind = %q, want %q", m.Kind, "relay-vectors-manifest")
	}
	if m.ManifestVersion != "relay-vectors/1" {
		t.Errorf("ManifestVersion = %q, want %q", m.ManifestVersion, "relay-vectors/1")
	}
	if m.VectorsVersion == "" {
		t.Error("VectorsVersion is empty")
	}
	if len(m.Vectors) == 0 {
		t.Error("Vectors is empty")
	}
	for _, v := range m.Vectors {
		if v.Name == "" {
			t.Error("a manifest entry has an empty name")
		}
		if len(v.SHA256) != 64 {
			t.Errorf("entry %q: sha256 = %q, want 64 lowercase hex chars", v.Name, v.SHA256)
		}
	}
}

//fusa:test REQ-RELAY-096
func TestVerifyVectorManifestClean(t *testing.T) {
	// The embedded vector set and the committed manifest MUST match exactly —
	// this is the reference implementation of the CI check Requirement 16
	// mandates. A non-empty result here means spec/vectors/*.json was edited
	// without regenerating spec/vectors/vectors_manifest.json.
	findings, err := VerifyVectorManifest()
	if err != nil {
		t.Fatalf("VerifyVectorManifest: %v", err)
	}
	if len(findings) != 0 {
		t.Errorf("embedded vectors diverge from vectors_manifest.json: %v", findings)
	}
}

//fusa:test REQ-RELAY-096
func TestVerifyVectorManifestDetectsHashMismatch(t *testing.T) {
	// VerifyVectorManifest must be able to detect divergence, not just
	// always report clean. Exercise its actual comparison logic (the same
	// sha256-then-lookup steps used internally) against a manifest with one
	// entry's hash deliberately wrong, and confirm exactly that entry is
	// flagged.
	names, err := VectorNames()
	if err != nil {
		t.Fatalf("VectorNames: %v", err)
	}
	if len(names) == 0 {
		t.Fatal("no embedded vectors to test against")
	}
	target := names[0]

	declared := make(map[string]string, len(names))
	for _, n := range names {
		raw, err := Vector(n)
		if err != nil {
			t.Fatalf("Vector(%q): %v", n, err)
		}
		sum := sha256.Sum256(raw)
		declared[n] = hex.EncodeToString(sum[:])
	}
	declared[target] = strings.Repeat("0", 64) // deliberately wrong

	var findings []string
	for _, n := range names {
		raw, err := Vector(n)
		if err != nil {
			t.Fatalf("Vector(%q): %v", n, err)
		}
		sum := sha256.Sum256(raw)
		got := hex.EncodeToString(sum[:])
		if got != declared[n] {
			findings = append(findings, n)
		}
	}
	if len(findings) != 1 || findings[0] != target {
		t.Errorf("corrupting %q's declared hash should surface exactly one finding for it, got %v", target, findings)
	}
}
