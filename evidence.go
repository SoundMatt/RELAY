// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package relay

import (
	"embed"
	"sort"
)

// evidenceFS embeds the safety-evidence artifacts so the relay binary can
// assemble a safety case and audit pack without access to the source tree.
// This includes the TLA+ formal lifecycle model (§6), which is embedded and
// bundled like the other evidence artifacts.
//
//fusa:req REQ-RELAY-074
//fusa:req REQ-RELAY-075
//go:embed .fusa-reqs.json .fusa-hara.json .fusa-tara.json
//go:embed spec/version.json docs/tool-safety-manual.md
//go:embed docs/formal/RelayLifecycle.tla docs/formal/README.md
var evidenceFS embed.FS

// evidencePaths maps the logical evidence name to its embedded path.
var evidencePaths = map[string]string{
	"requirements":       ".fusa-reqs.json",
	"hara":               ".fusa-hara.json",
	"tara":               ".fusa-tara.json",
	"version":            "spec/version.json",
	"tool-safety-manual": "docs/tool-safety-manual.md",
	"formal-model":       "docs/formal/RelayLifecycle.tla",
	"formal-model-doc":   "docs/formal/README.md",
}

// Evidence returns the raw bytes of a named safety-evidence artifact, e.g.
// "requirements", "hara", "tara", "version", "tool-safety-manual",
// "formal-model", "formal-model-doc".
//
//fusa:req REQ-RELAY-064
func Evidence(name string) ([]byte, error) {
	p, ok := evidencePaths[name]
	if !ok {
		return nil, &EvidenceError{Name: name}
	}
	return evidenceFS.ReadFile(p)
}

// EvidenceNames returns the logical names of all embedded evidence artifacts.
//
//fusa:req REQ-RELAY-064
func EvidenceNames() []string {
	names := make([]string, 0, len(evidencePaths))
	for n := range evidencePaths {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

// EvidenceError indicates an unknown evidence artifact name.
type EvidenceError struct{ Name string }

func (e *EvidenceError) Error() string { return "relay: unknown evidence artifact " + e.Name }
