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
// deterministic input set without access to the source tree. This includes
// spec/vectors/errors/ (reject-path fixtures) — a bare "*.json" glob does not
// recurse into subdirectories, so it is listed explicitly.
//
//fusa:req REQ-RELAY-057
//go:embed spec/vectors/*.json spec/vectors/errors/*.json
var vectorFS embed.FS

// Vector returns the raw bytes of the named golden vector (without the .json
// suffix), e.g. "can-standard-frame" or "errors/can-rtr-with-fd". It returns
// fs.ErrNotExist if absent.
//
//fusa:req REQ-RELAY-057
func Vector(name string) ([]byte, error) {
	return vectorFS.ReadFile("spec/vectors/" + name + ".json")
}

// VectorNames returns the names (without .json suffix, relative to
// spec/vectors/) of all embedded golden vectors, sorted — including the
// errors/ subdirectory's reject-path fixtures, e.g. "errors/can-rtr-with-fd".
//
//fusa:req REQ-RELAY-057
func VectorNames() ([]string, error) {
	var names []string
	err := fs.WalkDir(vectorFS, "spec/vectors", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".json") {
			return nil
		}
		rel := strings.TrimPrefix(path, "spec/vectors/")
		names = append(names, strings.TrimSuffix(rel, ".json"))
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(names)
	return names, nil
}
