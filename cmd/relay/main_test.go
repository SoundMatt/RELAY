// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	relay "github.com/SoundMatt/RELAY"
)

func TestVersionText(t *testing.T) {
	var out bytes.Buffer
	if err := runVersion(&out, nil); err != nil {
		t.Fatalf("runVersion text: %v", err)
	}
	got := out.String()
	if !strings.HasPrefix(got, "relay ") {
		t.Errorf("text output must start with \"relay \", got %q", got)
	}
	if !strings.Contains(got, relay.SpecVersion) {
		t.Errorf("text output must contain spec version %q, got %q", relay.SpecVersion, got)
	}
}

func TestVersionTextExplicit(t *testing.T) {
	var out bytes.Buffer
	if err := runVersion(&out, []string{"--format", "text"}); err != nil {
		t.Fatalf("runVersion --format text: %v", err)
	}
	if !strings.Contains(out.String(), "relay") {
		t.Error("text output must contain \"relay\"")
	}
}

func TestVersionJSON(t *testing.T) {
	var out bytes.Buffer
	if err := runVersion(&out, []string{"--format", "json"}); err != nil {
		t.Fatalf("runVersion json: %v", err)
	}
	var doc struct {
		Tool        string `json:"tool"`
		SpecVersion string `json:"spec_version"`
		Version     string `json:"version"`
		Language    string `json:"language"`
		Runtime     string `json:"runtime"`
	}
	if err := json.Unmarshal(out.Bytes(), &doc); err != nil {
		t.Fatalf("json.Unmarshal: %v\noutput: %s", err, out.String())
	}
	if doc.Tool != "relay" {
		t.Errorf("tool = %q, want %q", doc.Tool, "relay")
	}
	if doc.SpecVersion != relay.SpecVersion {
		t.Errorf("spec_version = %q, want %q", doc.SpecVersion, relay.SpecVersion)
	}
	if doc.Language != "go" {
		t.Errorf("language = %q, want %q", doc.Language, "go")
	}
	if doc.Version == "" {
		t.Error("version must not be empty")
	}
	if doc.Runtime == "" {
		t.Error("runtime must not be empty")
	}
}

func TestVersionUnknownFormat(t *testing.T) {
	var out bytes.Buffer
	err := runVersion(&out, []string{"--format", "xml"})
	if err == nil {
		t.Fatal("expected error for unknown format")
	}
	if !strings.Contains(err.Error(), "xml") {
		t.Errorf("error should mention the bad format, got %q", err.Error())
	}
}

func TestRunNoArgs(t *testing.T) {
	var out, errOut bytes.Buffer
	err := run(&out, &errOut, nil)
	if err == nil {
		t.Fatal("expected error with no args")
	}
	var code exitCode
	if !errors.As(err, &code) || int(code) != 2 {
		t.Errorf("expected exitCode(2), got %v", err)
	}
}

func TestRunHelp(t *testing.T) {
	var out, errOut bytes.Buffer
	if err := run(&out, &errOut, []string{"help"}); err != nil {
		t.Fatalf("run help: %v", err)
	}
	if !strings.Contains(out.String(), "version") {
		t.Error("help output must mention 'version'")
	}
}

func TestRunUnknownCommand(t *testing.T) {
	var out, errOut bytes.Buffer
	err := run(&out, &errOut, []string{"bogus"})
	if err == nil {
		t.Fatal("expected error for unknown command")
	}
	var code exitCode
	if !errors.As(err, &code) || int(code) != 2 {
		t.Errorf("expected exitCode(2), got %v", err)
	}
}

func TestRunVersion(t *testing.T) {
	var out, errOut bytes.Buffer
	if err := run(&out, &errOut, []string{"version"}); err != nil {
		t.Fatalf("run version: %v", err)
	}
	if !strings.Contains(out.String(), "relay") {
		t.Error("run version output must contain \"relay\"")
	}
}
