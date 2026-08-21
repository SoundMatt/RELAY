// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package relay

import (
	"encoding/json"
	"regexp"
	"testing"
)

// versionJSONVersion reads spec/version.json's authoritative "version"
// field — the single source of truth per §19.4.
func versionJSONVersion(t *testing.T) string {
	t.Helper()
	raw, err := Evidence("version")
	if err != nil {
		t.Fatalf("Evidence(version): %v", err)
	}
	var meta struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal(raw, &meta); err != nil {
		t.Fatalf("unmarshal spec/version.json: %v", err)
	}
	if meta.Version == "" {
		t.Fatal("spec/version.json: version field is empty")
	}
	return meta.Version
}

// TestSpecVersionMatchesVersionJSON is the mechanism behind §19.4/§19.5:
// SpecVersion (version.go) is a hand-copied literal by construction — Go
// constants can't be computed from an embedded file at compile time — but
// this test makes any drift from spec/version.json a CI failure instead of
// a silent, forgotten bump.
//
//fusa:test REQ-RELAY-100
func TestSpecVersionMatchesVersionJSON(t *testing.T) {
	want := versionJSONVersion(t)
	if SpecVersion != want {
		t.Errorf("SpecVersion = %q, spec/version.json version = %q — one was bumped without the other", SpecVersion, want)
	}
}

// TestSpecDocumentVersionLiteralsMatchVersionJSON enforces §19.5: every
// canonical version literal in the specification document's own text —
// not just what implementations print, which Requirement 14 already
// governs — MUST match spec/version.json. The stale "0.1" example
// literals fixed by hand in v2.2.3 are exactly the class of drift this
// closes; this makes it a CI failure instead of relying on the next
// person to notice.
//
//fusa:test REQ-RELAY-100
func TestSpecDocumentVersionLiteralsMatchVersionJSON(t *testing.T) {
	want := versionJSONVersion(t)

	specRaw, err := Evidence("specification")
	if err != nil {
		t.Fatalf("Evidence(specification): %v", err)
	}
	spec := string(specRaw)

	// Each of these anchors is expected to appear exactly once in
	// §13.5/§19.4/§19.5; a missing match means the surrounding prose was
	// reworded and this regex needs to move with it, not that the check
	// should be silently skipped.
	anchors := []struct {
		name string
		re   *regexp.Regexp
	}{
		{`§19.4 "Current version" line`, regexp.MustCompile(`Current version: \*\*v([0-9]+\.[0-9]+)\*\*`)},
		{"§19.4 Go snippet", regexp.MustCompile(`const SpecVersion = "([0-9]+\.[0-9]+)"`)},
		{"§19.4 C++ snippet", regexp.MustCompile(`kRelaySpecVersion = "([0-9]+\.[0-9]+)"`)},
		{"§19.4 Rust snippet", regexp.MustCompile(`RELAY_SPEC_VERSION: &str = "([0-9]+\.[0-9]+)"`)},
		{"§13.5 Docker LABEL", regexp.MustCompile(`LABEL io\.relay\.spec-version="([0-9]+\.[0-9]+)"`)},
	}
	for _, a := range anchors {
		m := a.re.FindStringSubmatch(spec)
		if m == nil {
			t.Errorf("%s: pattern not found in spec/relay-spec.md — reworded? update this test's regex", a.name)
			continue
		}
		if m[1] != want {
			t.Errorf("%s: found %q, want %q (spec/version.json)", a.name, m[1], want)
		}
	}

	// Every `"spec_version": "X.Y"` example literal across §12.1, §12.2,
	// §12.4, §17.2's manifest example, §20.6's attestation example, and
	// any future one MUST also match. Deliberately excludes §19.3's
	// `"<targeted-version>"` placeholder (not a numeric literal) and
	// §15.8's `"vectors_version"` illustration (a different field,
	// intentionally pinned independently — see §15.8/§19.5).
	exampleRe := regexp.MustCompile(`"spec_version":\s*"([0-9]+\.[0-9]+)"`)
	matches := exampleRe.FindAllStringSubmatch(spec, -1)
	if len(matches) == 0 {
		t.Fatal(`no "spec_version": "X.Y" example literal found in spec/relay-spec.md — has the example format changed? update this test`)
	}
	for _, m := range matches {
		if m[1] != want {
			t.Errorf(`"spec_version": %q example literal found in spec/relay-spec.md, want %q`, m[1], want)
		}
	}
}
