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
