// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

//fusa:test REQ-RELAY-086
func TestCrossbarForwards(t *testing.T) {
	dir := t.TempDir()
	sinkOut := filepath.Join(dir, "sink.ndjson")

	// Source spoke: on `subscribe` emit two relay.Message lines then exit.
	src := writeScript(t, `case "$1" in
subscribe) printf '%s\n' '{"protocol":1,"id":"1","payload":"AQ==","timestamp":"0001-01-01T00:00:00Z","meta":{}}' '{"protocol":1,"id":"2","payload":"Ag==","timestamp":"0001-01-01T00:00:00Z","meta":{}}' ;;
esac`)
	// Sink spoke: on `send` append stdin to the record file.
	sink := writeScript(t, `case "$1" in
send) cat >> `+sinkOut+` ;;
esac`)

	cfg := `{
	  "spokes": [
	    {"name":"src","binary":"` + src + `","protocol":"CAN"},
	    {"name":"dst","binary":"` + sink + `","protocol":"CAN"}
	  ],
	  "routes": [ {"from":"src","to":["dst"]} ]
	}`
	cfgPath := filepath.Join(dir, "crossbar.json")
	if err := os.WriteFile(cfgPath, []byte(cfg), 0o644); err != nil {
		t.Fatal(err)
	}

	var out, errb bytes.Buffer
	if err := runCrossbar(&out, &errb, []string{"--config", cfgPath, "--duration", "2s"}); err != nil {
		t.Fatalf("runCrossbar: %v (%s)", err, errb.String())
	}
	got, _ := os.ReadFile(sinkOut)
	lines := strings.Count(strings.TrimSpace(string(got)), "\n") + 1
	if len(got) == 0 || lines != 2 {
		t.Errorf("expected 2 forwarded messages at sink, got %d line(s):\n%s", lines, got)
	}
	if !strings.Contains(out.String(), "forwarded=2") {
		t.Errorf("stats should report forwarded=2:\n%s", out.String())
	}
}

//fusa:test REQ-RELAY-086
func TestCrossbarConfigErrors(t *testing.T) {
	dir := t.TempDir()
	write := func(name, body string) string {
		p := filepath.Join(dir, name)
		_ = os.WriteFile(p, []byte(body), 0o644)
		return p
	}
	good := writeScript(t, "")
	cases := []struct {
		name string
		args []string
		want int
	}{
		{"no config flag", nil, 2},
		{"missing file", []string{"--config", filepath.Join(dir, "nope.json")}, 2},
		{"bad json", []string{"--config", write("bad.json", "{not json")}, 2},
		{"empty config", []string{"--config", write("empty.json", `{"spokes":[],"routes":[]}`)}, 2},
		{"unknown protocol", []string{"--config", write("proto.json",
			`{"spokes":[{"name":"a","binary":"`+good+`","protocol":"NOPE"}],"routes":[{"from":"a","to":["a"]}]}`)}, 2},
		{"unknown converter", []string{"--config", write("conv.json",
			`{"spokes":[{"name":"a","binary":"`+good+`","protocol":"CAN"}],"routes":[{"from":"a","to":["a"],"converter":"bogus"}]}`)}, 2},
		{"unknown route spoke", []string{"--config", write("route.json",
			`{"spokes":[{"name":"a","binary":"`+good+`","protocol":"CAN"}],"routes":[{"from":"ghost","to":["a"]}]}`)}, 2},
	}
	for _, tc := range cases {
		var out, errb bytes.Buffer
		err := runCrossbar(&out, &errb, tc.args)
		var code exitCode
		if !errors.As(err, &code) || int(code) != tc.want {
			t.Errorf("%s: err=%v, want exitCode(%d)", tc.name, err, tc.want)
		}
	}
}

//fusa:test REQ-RELAY-086
func TestCliNodeProtocol(t *testing.T) {
	n := &cliNode{binary: "x", proto: 1}
	if n.Protocol() != 1 {
		t.Error("Protocol accessor")
	}
	// Close is idempotent and safe before any Subscribe.
	if err := n.Close(); err != nil {
		t.Errorf("Close: %v", err)
	}
	_ = n.Close()
}
