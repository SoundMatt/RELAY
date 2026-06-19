// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package relay

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

//fusa:test REQ-RELAY-064
func TestEvidenceNames(t *testing.T) {
	names := EvidenceNames()
	want := map[string]bool{
		"requirements": true, "hara": true, "tara": true,
		"version": true, "tool-safety-manual": true,
		"formal-model": true, "formal-model-doc": true,
		"asil-d-uplift": true, "specification": true,
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

//fusa:test REQ-RELAY-050
func TestHARAEvidence(t *testing.T) {
	b, err := Evidence("hara")
	if err != nil || len(b) == 0 {
		t.Fatalf("hara evidence missing: %v", err)
	}
	var h struct {
		Standard     string           `json:"standard"`
		Hazards      []map[string]any `json:"hazards"`
		SafetyGoals  []map[string]any `json:"safetyGoals"`
		OperationalS []map[string]any `json:"operationalSituations"`
	}
	if err := json.Unmarshal(b, &h); err != nil {
		t.Fatalf("hara is not valid JSON: %v", err)
	}
	if h.Standard == "" || len(h.Hazards) == 0 || len(h.SafetyGoals) == 0 {
		t.Errorf("HARA must declare a standard, hazards, and safety goals; got %+v", h)
	}
}

//fusa:test REQ-RELAY-045
//fusa:test REQ-RELAY-046
func TestSpecStatesConstructorContract(t *testing.T) {
	// REQ-045/046 are levied by the spec on conformant implementations (not on
	// the relay tool). RELAY's verification obligation is that the normative
	// §7 constructor contract is stated in the embedded specification.
	spec, err := Evidence("specification")
	if err != nil || len(spec) == 0 {
		t.Fatalf("specification evidence missing: %v", err)
	}
	s := string(spec)
	if !strings.Contains(s, "## 7. Constructor Contract") {
		t.Error("spec is missing §7 Constructor Contract")
	}
	// REQ-045: Form 1 endpoint-addressed New signature.
	if !strings.Contains(s, "New(ctx context.Context, endpoint string") {
		t.Error("spec §7 must state the Form 1 New(ctx, endpoint, ...) signature (REQ-045)")
	}
	// REQ-046: mandatory mock sub-package.
	if !strings.Contains(s, "mock") || !strings.Contains(s, "Form 2") {
		t.Error("spec §7 must require a mock sub-package with a Form 2 New (REQ-046)")
	}
}

//fusa:test REQ-RELAY-087
func TestSpecDefinesLibraryArchitecture(t *testing.T) {
	spec, err := Evidence("specification")
	if err != nil || len(spec) == 0 {
		t.Fatalf("specification evidence missing: %v", err)
	}
	s := string(spec)
	if !strings.Contains(s, "13.7 Cross-language library architecture") {
		t.Error("spec must define §13.7 cross-language library architecture (REQ-087)")
	}
	// The standard module-name registry must name the key common modules.
	for _, mod := range []string{"`adapt`", "`mock`", "`virtual`", "module-name registry"} {
		if !strings.Contains(s, mod) {
			t.Errorf("§13.7 must reference %s", mod)
		}
	}
}

//fusa:test REQ-RELAY-074
//fusa:test REQ-RELAY-075
func TestFormalModelCoversLifecycle(t *testing.T) {
	// The formal model must be embedded as evidence and be non-empty.
	model, err := Evidence("formal-model")
	if err != nil || len(model) == 0 {
		t.Fatalf("formal-model evidence missing: %v", err)
	}
	doc, err := Evidence("formal-model-doc")
	if err != nil || len(doc) == 0 {
		t.Fatalf("formal-model-doc evidence missing: %v", err)
	}
	// The model must be the lifecycle module and define a Spec for TLC.
	ms := string(model)
	if !strings.Contains(ms, "MODULE RelayLifecycle") || !strings.Contains(ms, "Spec ==") {
		t.Error("formal-model is not a well-formed RelayLifecycle TLA+ module")
	}
	// The documentation's requirement→invariant mapping must reference every
	// one of the ten §6 lifecycle requirements (6.1 … 6.10), so the mapping
	// cannot silently drop a requirement.
	ds := string(doc)
	for i := 1; i <= 10; i++ {
		ref := "6." + itoa(i)
		if !strings.Contains(ds, ref) {
			t.Errorf("formal-model-doc does not reference §6 requirement %s", ref)
		}
	}
}

//fusa:test REQ-RELAY-076
func TestAsilDUpliftEvidence(t *testing.T) {
	b, err := Evidence("asil-d-uplift")
	if err != nil || len(b) == 0 {
		t.Fatalf("asil-d-uplift evidence missing: %v", err)
	}
	s := string(b)
	// The uplift doc must address both target standards and stay framed as a
	// path, not a claim of current qualification.
	for _, want := range []string{"ASIL-D", "DAL-A", "DO-330", "TCL2"} {
		if !strings.Contains(s, want) {
			t.Errorf("asil-d-uplift does not mention %q", want)
		}
	}
}

// itoa avoids importing strconv solely for single-digit formatting.
func itoa(i int) string {
	if i < 10 {
		return string(rune('0' + i))
	}
	return string(rune('0'+i/10)) + string(rune('0'+i%10))
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
