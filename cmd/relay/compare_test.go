// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"path/filepath"
	"reflect"
	"testing"
)

func sampleCaps() capsDoc {
	return capsDoc{
		Kind: "capabilities", Tool: "go-can", Protocol: "CAN", Version: "1.0.0",
		SpecVersion: "0.3", Commands: []string{"version", "capabilities", "status"},
		Features: []string{"fd"}, Interfaces: []string{"Bus"}, Adapt: true,
	}
}

//fusa:test REQ-RELAY-066
func TestCompareIdentical(t *testing.T) {
	res := compareCaps("a", "b", sampleCaps(), sampleCaps())
	if !res.Compatible {
		t.Errorf("identical caps should be compatible: %+v", res.Differences)
	}
	if len(res.Differences) != 0 {
		t.Errorf("identical caps should have no differences, got %v", res.Differences)
	}
}

//fusa:test REQ-RELAY-066
func TestCompareProtocolMismatch(t *testing.T) {
	a := sampleCaps()
	b := sampleCaps()
	b.Protocol = "LIN"
	res := compareCaps("a", "b", a, b)
	if res.Compatible || res.ProtocolMatch {
		t.Error("different protocols must be incompatible")
	}
}

//fusa:test REQ-RELAY-066
func TestCompareSpecVersionMismatch(t *testing.T) {
	a := sampleCaps()
	b := sampleCaps()
	b.SpecVersion = "0.2"
	res := compareCaps("a", "b", a, b)
	if res.Compatible || res.SpecVersionMatch {
		t.Error("different spec versions must be incompatible")
	}
}

//fusa:test REQ-RELAY-066
func TestCompareCommandDelta(t *testing.T) {
	a := sampleCaps()
	b := sampleCaps()
	b.Commands = append(b.Commands, "send")
	res := compareCaps("a", "b", a, b)
	if res.Compatible {
		t.Error("differing command sets must be incompatible")
	}
	if !reflect.DeepEqual(res.CommandsOnlyB, []string{"send"}) {
		t.Errorf("CommandsOnlyB = %v, want [send]", res.CommandsOnlyB)
	}
	if len(res.CommandsOnlyA) != 0 {
		t.Errorf("CommandsOnlyA = %v, want empty", res.CommandsOnlyA)
	}
}

//fusa:test REQ-RELAY-066
func TestSetDiff(t *testing.T) {
	onlyA, onlyB := setDiff([]string{"a", "b", "c"}, []string{"b", "c", "d"})
	if !reflect.DeepEqual(onlyA, []string{"a"}) || !reflect.DeepEqual(onlyB, []string{"d"}) {
		t.Errorf("setDiff = (%v, %v), want ([a], [d])", onlyA, onlyB)
	}
}

//fusa:test REQ-RELAY-066
func TestRunCompareSelf(t *testing.T) {
	bin := buildTestBinary(t)
	var out, errb bytes.Buffer
	if err := runCompare(&out, &errb, []string{"--format", "json", bin, bin}); err != nil {
		t.Fatalf("runCompare self: %v", err)
	}
	var res compareResult
	if err := json.Unmarshal(out.Bytes(), &res); err != nil {
		t.Fatalf("compare json: %v\n%s", err, out.String())
	}
	if !res.Compatible {
		t.Errorf("relay vs relay must be compatible: %v", res.Differences)
	}
}

//fusa:test REQ-RELAY-066
func TestRunCompareText(t *testing.T) {
	bin := buildTestBinary(t)
	var out, errb bytes.Buffer
	if err := runCompare(&out, &errb, []string{bin, bin}); err != nil {
		t.Fatalf("runCompare text: %v", err)
	}
	if !bytes.Contains(out.Bytes(), []byte("VERDICT: COMPATIBLE")) {
		t.Errorf("expected COMPATIBLE verdict:\n%s", out.String())
	}
}

//fusa:test REQ-RELAY-066
func TestRunCompareUnknownFormat(t *testing.T) {
	bin := buildTestBinary(t)
	var out, errb bytes.Buffer
	if err := runCompare(&out, &errb, []string{"--format", "xml", bin, bin}); err == nil {
		t.Error("expected error for unknown format")
	}
}

//fusa:test REQ-RELAY-067
func TestRunVersionsScan(t *testing.T) {
	bin := buildTestBinary(t)
	t.Setenv("PATH", filepath.Dir(bin))
	var out, errb bytes.Buffer
	if err := runVersions(&out, &errb, []string{"--scan", "--match", "relay*", "--format", "json"}); err != nil {
		t.Fatalf("runVersions --scan: %v", err)
	}
	var entries []versionEntry
	if err := json.Unmarshal(out.Bytes(), &entries); err != nil {
		t.Fatalf("scan json: %v", err)
	}
	if len(entries) != 1 || entries[0].Tool != "relay" {
		t.Errorf("scan should find relay, got %+v", entries)
	}
}

//fusa:test REQ-RELAY-066
func TestRunCompareWrongArgCount(t *testing.T) {
	var out, errb bytes.Buffer
	err := runCompare(&out, &errb, []string{"only-one"})
	var code exitCode
	if !errors.As(err, &code) || int(code) != 2 {
		t.Errorf("compare with one arg should exit 2, got %v", err)
	}
}

//fusa:test REQ-RELAY-067
func TestRunVersionsJSON(t *testing.T) {
	bin := buildTestBinary(t)
	var out, errb bytes.Buffer
	if err := runVersions(&out, &errb, []string{"--format", "json", bin}); err != nil {
		t.Fatalf("runVersions: %v", err)
	}
	var entries []versionEntry
	if err := json.Unmarshal(out.Bytes(), &entries); err != nil {
		t.Fatalf("versions json: %v\n%s", err, out.String())
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	// relay reports its own SpecVersion, so it is aligned with itself.
	if !entries[0].Aligned {
		t.Errorf("relay should be aligned with itself: %+v", entries[0])
	}
}

//fusa:test REQ-RELAY-067
func TestRunVersionsText(t *testing.T) {
	bin := buildTestBinary(t)
	var out, errb bytes.Buffer
	if err := runVersions(&out, &errb, []string{bin}); err != nil {
		t.Fatalf("runVersions: %v", err)
	}
	if !bytes.Contains(out.Bytes(), []byte("ALIGNED")) {
		t.Errorf("versions text missing header:\n%s", out.String())
	}
}

//fusa:test REQ-RELAY-067
func TestRunVersionsNoArgs(t *testing.T) {
	var out, errb bytes.Buffer
	err := runVersions(&out, &errb, nil)
	var code exitCode
	if !errors.As(err, &code) || int(code) != 2 {
		t.Errorf("versions with no args should exit 2, got %v", err)
	}
}
