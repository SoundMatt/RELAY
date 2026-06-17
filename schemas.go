// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package relay

import (
	"embed"
	"io/fs"
)

// schemaFS embeds the machine-readable JSON Schemas (draft 2020-12) for every
// canonical type and CLI document defined by the spec (§12, §15). Conformance
// tooling and downstream implementations consume these directly.
//
//go:embed spec/schemas/*.json
var schemaFS embed.FS

// Schema returns the raw JSON Schema bytes for the named schema, e.g.
// "cli-version", "can-frame", "someip-message" (without the .json suffix).
// It returns fs.ErrNotExist if no such schema is embedded.
//
//fusa:req REQ-RELAY-058
func Schema(name string) ([]byte, error) {
	return schemaFS.ReadFile("spec/schemas/" + name + ".json")
}

// SchemaNames returns the names (without .json suffix) of all embedded schemas.
//
//fusa:req REQ-RELAY-058
func SchemaNames() ([]string, error) {
	entries, err := fs.ReadDir(schemaFS, "spec/schemas")
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		n := e.Name()
		if len(n) > 5 && n[len(n)-5:] == ".json" {
			names = append(names, n[:len(n)-5])
		}
	}
	return names, nil
}
