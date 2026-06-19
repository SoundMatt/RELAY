// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package relay

import (
	"encoding/json"
	"errors"
	"io/fs"
	"testing"
)

//fusa:test REQ-RELAY-058
func TestSchemaNames(t *testing.T) {
	names, err := SchemaNames()
	if err != nil {
		t.Fatalf("SchemaNames: %v", err)
	}
	if len(names) == 0 {
		t.Fatal("expected at least one embedded schema")
	}
	// Every named schema must resolve to valid JSON via Schema().
	for _, n := range names {
		b, err := Schema(n)
		if err != nil {
			t.Errorf("Schema(%q): %v", n, err)
			continue
		}
		var v any
		if err := json.Unmarshal(b, &v); err != nil {
			t.Errorf("Schema(%q) is not valid JSON: %v", n, err)
		}
	}
}

//fusa:test REQ-RELAY-058
func TestSchemaUnknown(t *testing.T) {
	_, err := Schema("does-not-exist")
	if !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("Schema(unknown) error = %v, want fs.ErrNotExist", err)
	}
}

//fusa:test REQ-RELAY-064
func TestEvidenceErrorMessage(t *testing.T) {
	e := &EvidenceError{Name: "nope"}
	if got := e.Error(); got == "" || !contains(got, "nope") {
		t.Errorf("EvidenceError.Error() = %q, want it to name the artifact", got)
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
