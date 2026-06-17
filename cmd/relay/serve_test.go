// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func serveTestHandler(t *testing.T) http.Handler {
	t.Helper()
	bin := buildTestBinary(t)
	return newServeHandler(serveConfig{binaries: []string{bin}})
}

func get(t *testing.T, h http.Handler, path string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	h.ServeHTTP(rec, req)
	return rec
}

//fusa:test REQ-RELAY-068
func TestServeImplementations(t *testing.T) {
	rec := get(t, serveTestHandler(t), "/api/v1/implementations")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var results []probeResult
	if err := json.Unmarshal(rec.Body.Bytes(), &results); err != nil {
		t.Fatalf("implementations json: %v\n%s", err, rec.Body.String())
	}
	if len(results) != 1 || results[0].Tool != "relay" {
		t.Errorf("expected the relay implementation, got %+v", results)
	}
}

//fusa:test REQ-RELAY-068
func TestServeStatus(t *testing.T) {
	rec := get(t, serveTestHandler(t), "/api/v1/status")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var doc reportDoc
	if err := json.Unmarshal(rec.Body.Bytes(), &doc); err != nil {
		t.Fatalf("status json: %v", err)
	}
	if len(doc.Entries) != 1 {
		t.Errorf("expected 1 entry, got %d", len(doc.Entries))
	}
}

//fusa:test REQ-RELAY-068
func TestServeDashboard(t *testing.T) {
	rec := get(t, serveTestHandler(t), "/")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
		t.Errorf("content-type = %q, want text/html", ct)
	}
	if !strings.Contains(rec.Body.String(), "RELAY dashboard") {
		t.Errorf("dashboard missing title:\n%s", rec.Body.String())
	}
}

//fusa:test REQ-RELAY-069
func TestServeBadge(t *testing.T) {
	rec := get(t, serveTestHandler(t), "/badge/status.svg")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "image/svg+xml" {
		t.Errorf("content-type = %q, want image/svg+xml", ct)
	}
	body := rec.Body.String()
	if !strings.HasPrefix(body, "<svg") || !strings.Contains(body, "relay") {
		t.Errorf("badge not an SVG mentioning relay:\n%s", body)
	}
}

//fusa:test REQ-RELAY-068
func TestServeNotFound(t *testing.T) {
	rec := get(t, serveTestHandler(t), "/nope")
	if rec.Code != http.StatusNotFound {
		t.Errorf("unknown path status = %d, want 404", rec.Code)
	}
}

//fusa:test REQ-RELAY-069
func TestStatusBadgeColors(t *testing.T) {
	cases := map[conformSeverity]string{
		sevPass: "#2ea043", sevWarn: "#bf8700", sevFail: "#cf222e",
	}
	for sev, want := range cases {
		svg := statusBadgeSVG(sev)
		if !strings.Contains(svg, want) {
			t.Errorf("badge for %s missing colour %s", sev, want)
		}
		if !strings.Contains(svg, string(sev)) {
			t.Errorf("badge for %s missing status text", sev)
		}
	}
}

//fusa:test REQ-RELAY-068
func TestRunServeNoArgs(t *testing.T) {
	var out, errb bytes.Buffer
	err := runServe(&out, &errb, nil)
	var code exitCode
	if !errors.As(err, &code) || int(code) != 2 {
		t.Errorf("serve with no args should exit 2, got %v", err)
	}
}
