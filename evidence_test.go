// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package relay

import (
	"encoding/json"
	"errors"
	"testing"
)

//fusa:test REQ-RELAY-064
func TestEvidenceNames(t *testing.T) {
	names := EvidenceNames()
	want := map[string]bool{
		"requirements": true, "hara": true, "tara": true,
		"version": true, "tool-safety-manual": true,
	}
	if len(names) != len(want) {
		t.Fatalf("EvidenceNames = %v, want %d entries", names, len(want))
	}
	for _, n := range names {
		if !want[n] {
			t.Errorf("unexpected evidence name %q", n)
		}
	}
}

//fusa:test REQ-RELAY-064
func TestEvidenceContent(t *testing.T) {
	// The JSON artifacts must be valid JSON and non-empty.
	for _, n := range []string{"requirements", "hara", "tara", "version"} {
		b, err := Evidence(n)
		if err != nil {
			t.Fatalf("Evidence(%q): %v", n, err)
		}
		var v interface{}
		if err := json.Unmarshal(b, &v); err != nil {
			t.Errorf("Evidence(%q) is not valid JSON: %v", n, err)
		}
	}
	if b, err := Evidence("tool-safety-manual"); err != nil || len(b) == 0 {
		t.Errorf("tool-safety-manual evidence missing: %v", err)
	}
}

//fusa:test REQ-RELAY-064
func TestEvidenceUnknown(t *testing.T) {
	_, err := Evidence("nope")
	var ee *EvidenceError
	if !errors.As(err, &ee) {
		t.Errorf("Evidence(unknown) error = %v, want *EvidenceError", err)
	}
}
