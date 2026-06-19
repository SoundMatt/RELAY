// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package relay

import "testing"

//fusa:test REQ-RELAY-020
func TestSpecVersion(t *testing.T) {
	if SpecVersion != "1.11" {
		t.Errorf("SpecVersion = %q, want %q", SpecVersion, "1.11")
	}
}
