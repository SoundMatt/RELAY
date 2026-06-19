// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package relay

import (
	"embed"
	"io/fs"
	"sort"
	"strings"
)

// vectorFS embeds the golden reference vectors (§15.7 round-trip fixtures) so
// tooling such as `relay interop` can drive implementations with a known,
// deterministic input set without access to the source tree.
//
//fusa:req REQ-RELAY-057
//go:embed spec/vectors/*.json
var vectorFS embed.FS

// Vector returns the raw bytes of the named golden vector (without the .json
// suffix), e.g. "can-standard-frame". It returns fs.ErrNotExist if absent.
//
//fusa:req REQ-RELAY-057
func Vector(name string) ([]byte, error) {
	return vectorFS.ReadFile("spec/vectors/" + name + ".json")
}

// VectorNames returns the names (without .json suffix) of all embedded golden
// vectors, sorted.
//
//fusa:req REQ-RELAY-057
func VectorNames() ([]string, error) {
	entries, err := fs.ReadDir(vectorFS, "spec/vectors")
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		n := e.Name()
		if strings.HasSuffix(n, ".json") {
			names = append(names, strings.TrimSuffix(n, ".json"))
		}
	}
	sort.Strings(names)
	return names, nil
}
