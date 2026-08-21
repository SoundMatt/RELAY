// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package relay

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"sort"
	"strings"

	"embed"
)

// vectorsManifestPath is the embedded path of the vector distribution
// manifest (spec §15.8, Requirement 16). It lives alongside the golden
// vectors it describes but is metadata about the set, not a vector itself,
// so VectorNames excludes it explicitly.
const vectorsManifestPath = "spec/vectors/vectors_manifest.json"

// vectorFS embeds the golden reference vectors (§15.7 round-trip fixtures) so
// tooling such as `relay interop` can drive implementations with a known,
// deterministic input set without access to the source tree. This includes
// spec/vectors/errors/ (reject-path fixtures) — a bare "*.json" glob does not
// recurse into subdirectories, so it is listed explicitly. It also embeds
// spec/vectors/vectors_manifest.json (§15.8), the pinning manifest for this
// same distribution.
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
// spec/vectors/vectors_manifest.json (§15.8) is metadata about the vector
// set, not a vector itself, and is excluded.
//
//fusa:req REQ-RELAY-057
func VectorNames() ([]string, error) {
	var names []string
	err := fs.WalkDir(vectorFS, "spec/vectors", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".json") || path == vectorsManifestPath {
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

// VectorManifestEntry is one entry in a relay-vectors/1 manifest's "vectors"
// array (spec §15.8): a vector's name (relative to spec/vectors/, .json
// suffix stripped) paired with the SHA-256 (lowercase hex) of its raw bytes.
type VectorManifestEntry struct {
	Name   string `json:"name"`
	SHA256 string `json:"sha256"`
}

// VectorManifest is the relay-vectors/1 manifest document (spec §15.8,
// §17 Requirement 16), pinning the canonical spec/vectors/ distribution.
type VectorManifest struct {
	Kind            string                `json:"kind"`
	ManifestVersion string                `json:"manifest_version"`
	VectorsVersion  string                `json:"vectors_version"`
	Vectors         []VectorManifestEntry `json:"vectors"`
}

// VectorsManifest returns the raw bytes of the embedded
// spec/vectors/vectors_manifest.json (§15.8).
//
//fusa:req REQ-RELAY-096
func VectorsManifest() ([]byte, error) {
	return vectorFS.ReadFile(vectorsManifestPath)
}

// ParsedVectorsManifest returns the embedded vector distribution manifest,
// parsed (§15.8).
//
//fusa:req REQ-RELAY-096
func ParsedVectorsManifest() (VectorManifest, error) {
	raw, err := VectorsManifest()
	if err != nil {
		return VectorManifest{}, err
	}
	var m VectorManifest
	if err := json.Unmarshal(raw, &m); err != nil {
		return VectorManifest{}, fmt.Errorf("parse vectors_manifest.json: %w", err)
	}
	return m, nil
}

// VerifyVectorManifest recomputes the SHA-256 of every embedded golden
// vector and compares it against the embedded relay-vectors/1 manifest
// (§15.8, Requirement 16). It returns one human-readable finding per
// divergence — a vector present but not listed in the manifest, a vector
// listed but not present, or a listed vector whose SHA-256 does not match —
// and a nil/empty slice when the embedded set exactly matches the manifest.
// This is the reference implementation of the CI check Requirement 16
// mandates every implementation that embeds a local vector copy perform.
//
//fusa:req REQ-RELAY-096
func VerifyVectorManifest() ([]string, error) {
	m, err := ParsedVectorsManifest()
	if err != nil {
		return nil, err
	}
	if m.Kind != "relay-vectors-manifest" {
		return nil, fmt.Errorf("vectors_manifest.json: kind = %q, want %q", m.Kind, "relay-vectors-manifest")
	}

	declared := make(map[string]string, len(m.Vectors))
	for _, v := range m.Vectors {
		declared[v.Name] = v.SHA256
	}

	names, err := VectorNames()
	if err != nil {
		return nil, err
	}
	present := make(map[string]bool, len(names))

	var findings []string
	for _, name := range names {
		present[name] = true
		raw, err := Vector(name)
		if err != nil {
			return nil, err
		}
		sum := sha256.Sum256(raw)
		got := hex.EncodeToString(sum[:])
		want, ok := declared[name]
		if !ok {
			findings = append(findings, fmt.Sprintf("vector %q is embedded but not listed in vectors_manifest.json", name))
			continue
		}
		if got != want {
			findings = append(findings, fmt.Sprintf("vector %q: sha256 = %s, manifest declares %s", name, got, want))
		}
	}
	for name := range declared {
		if !present[name] {
			findings = append(findings, fmt.Sprintf("vector %q is listed in vectors_manifest.json but not embedded", name))
		}
	}
	sort.Strings(findings)
	return findings, nil
}
