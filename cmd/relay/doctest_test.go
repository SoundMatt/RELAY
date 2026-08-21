// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package main

import (
	"encoding/json"
	"testing"

	relay "github.com/SoundMatt/RELAY/v2"
)

// TestSpecDoctestExamplesValidate is spec §20.7's mechanism: every JSON
// example marked `<!-- doctest:KIND -->` in relay-spec.md is extracted and
// run through the exact same validator relay conform itself uses on a real
// binary's output — not a relaxed or hand-copied check. A stale or
// hand-broken example (the class of drift that produced NEW-SPEC-6's "0.1"
// incident) fails this test, not just a human reader's attention.
//
//fusa:test REQ-RELAY-102
func TestSpecDoctestExamplesValidate(t *testing.T) {
	specRaw, err := relay.Evidence("specification")
	if err != nil {
		t.Fatalf("Evidence(specification): %v", err)
	}
	examples, err := extractDoctestExamples(string(specRaw))
	if err != nil {
		t.Fatal(err)
	}

	// Sanity: catches the marker convention silently falling out of sync
	// with the document (all markers removed, or the kind renamed) rather
	// than the loop below quietly validating zero examples and passing.
	for _, kind := range []string{"version", "capabilities", "status", "manifest", "attestation", "vectors-manifest"} {
		if len(examples[kind]) == 0 {
			t.Errorf("no doctest:%s example found in spec/relay-spec.md", kind)
		}
	}

	for _, ex := range examples["version"] {
		assertDoctestNoFail(t, "version", ex, validateVersionDoc(ex))
	}
	for _, ex := range examples["capabilities"] {
		assertDoctestNoFail(t, "capabilities", ex, validateCapabilitiesDoc(ex))
	}
	for _, ex := range examples["status"] {
		assertDoctestNoFail(t, "status", ex, validateStatusDoc(ex))
	}
	for _, ex := range examples["manifest"] {
		assertDoctestSchemaValid(t, "manifest", "relay-conform-manifest", ex)
	}
	for _, ex := range examples["attestation"] {
		assertDoctestSchemaValid(t, "attestation", "relay-conform-attestation", ex)
	}
	for _, ex := range examples["vectors-manifest"] {
		assertDoctestSchemaValid(t, "vectors-manifest", "vectors-manifest", ex)
	}
}

//fusa:test REQ-RELAY-102
func TestExtractDoctestExamplesUnterminatedFence(t *testing.T) {
	if _, err := extractDoctestExamples("<!-- doctest:version -->\n```json\n{\"a\":1}\n"); err == nil {
		t.Error("expected an error for an unterminated json fence")
	}
	if _, err := extractDoctestExamples("<!-- doctest:version -->\nnot a fence\n"); err == nil {
		t.Error("expected an error when the marker isn't followed by a ```json fence")
	}
}

func assertDoctestNoFail(t *testing.T, kind string, example []byte, findings []conformFinding) {
	t.Helper()
	if hasFail(findings) {
		t.Errorf("doctest:%s example failed validation: %+v\n%s", kind, findings, example)
	}
}

func assertDoctestSchemaValid(t *testing.T, kind, schemaName string, example []byte) {
	t.Helper()
	var asAny interface{}
	if err := json.Unmarshal(example, &asAny); err != nil {
		t.Errorf("doctest:%s example is not valid JSON: %v\n%s", kind, err, example)
		return
	}
	schemaJSON, err := relay.Schema(schemaName)
	if err != nil {
		t.Fatalf("doctest:%s: no embedded schema %q: %v", kind, schemaName, err)
	}
	violations := validateSchema(schemaJSON, asAny)
	if len(violations) > 0 {
		t.Errorf("doctest:%s example violates %s schema: %v\n%s", kind, schemaName, violations, example)
	}
}
