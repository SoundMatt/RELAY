// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	relay "github.com/SoundMatt/RELAY"
)

// ndjsonOf marshals messages to newline-delimited JSON, the on-wire trace format.
func ndjsonOf(t *testing.T, msgs ...relay.Message) string {
	t.Helper()
	var b strings.Builder
	for _, m := range msgs {
		j, err := json.Marshal(m)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		b.Write(j)
		b.WriteByte('\n')
	}
	return b.String()
}

func sampleMsgs() []relay.Message {
	ts := time.Unix(1700000000, 0).UTC()
	return []relay.Message{
		{Protocol: relay.CAN, ID: "291", Payload: []byte{0xDE, 0xAD}, Timestamp: ts, Seq: 1},
		{Protocol: relay.DDS, ID: "rt/x", Payload: []byte("hi"), Timestamp: ts, Seq: 2},
		{Protocol: relay.CAN, ID: "292", Payload: []byte{0x01}, Timestamp: ts, Seq: 3},
	}
}

//fusa:test REQ-RELAY-061
func TestCaptureTraceNDJSON(t *testing.T) {
	in := ndjsonOf(t, sampleMsgs()...)
	var out bytes.Buffer
	n, err := captureTrace(strings.NewReader(in), &out, "ndjson", 0, nil)
	if err != nil {
		t.Fatalf("captureTrace: %v", err)
	}
	if n != 3 {
		t.Errorf("n = %d, want 3", n)
	}
	lines := strings.Count(strings.TrimSpace(out.String()), "\n") + 1
	if lines != 3 {
		t.Errorf("ndjson output lines = %d, want 3", lines)
	}
}

//fusa:test REQ-RELAY-061
func TestCaptureTraceJSON(t *testing.T) {
	in := ndjsonOf(t, sampleMsgs()...)
	var out bytes.Buffer
	if _, err := captureTrace(strings.NewReader(in), &out, "json", 0, nil); err != nil {
		t.Fatalf("captureTrace: %v", err)
	}
	var msgs []relay.Message
	if err := json.Unmarshal(out.Bytes(), &msgs); err != nil {
		t.Fatalf("json output not an array: %v\n%s", err, out.String())
	}
	if len(msgs) != 3 {
		t.Errorf("array len = %d, want 3", len(msgs))
	}
}

//fusa:test REQ-RELAY-061
func TestCaptureTraceText(t *testing.T) {
	in := ndjsonOf(t, sampleMsgs()...)
	var out bytes.Buffer
	if _, err := captureTrace(strings.NewReader(in), &out, "text", 0, nil); err != nil {
		t.Fatalf("captureTrace: %v", err)
	}
	if !strings.Contains(out.String(), "3 message(s)") {
		t.Errorf("text output should summarise 3 messages, got:\n%s", out.String())
	}
}

//fusa:test REQ-RELAY-061
func TestCaptureTraceCountLimit(t *testing.T) {
	in := ndjsonOf(t, sampleMsgs()...)
	var out bytes.Buffer
	n, err := captureTrace(strings.NewReader(in), &out, "ndjson", 2, nil)
	if err != nil {
		t.Fatalf("captureTrace: %v", err)
	}
	if n != 2 {
		t.Errorf("count-limited n = %d, want 2", n)
	}
}

//fusa:test REQ-RELAY-061
func TestCaptureTraceProtocolFilter(t *testing.T) {
	in := ndjsonOf(t, sampleMsgs()...)
	can := relay.CAN
	var out bytes.Buffer
	n, err := captureTrace(strings.NewReader(in), &out, "ndjson", 0, &can)
	if err != nil {
		t.Fatalf("captureTrace: %v", err)
	}
	if n != 2 {
		t.Errorf("CAN-filtered n = %d, want 2 (two CAN messages)", n)
	}
	if strings.Contains(out.String(), "rt/x") {
		t.Error("DDS message should have been filtered out")
	}
}

//fusa:test REQ-RELAY-061
func TestCaptureTraceMalformed(t *testing.T) {
	var out bytes.Buffer
	_, err := captureTrace(strings.NewReader("{not json}\n"), &out, "ndjson", 0, nil)
	if err == nil {
		t.Error("expected error on malformed message line")
	}
}

//fusa:test REQ-RELAY-061
func TestCaptureTraceBlankLinesIgnored(t *testing.T) {
	in := "\n" + ndjsonOf(t, sampleMsgs()[0]) + "\n\n"
	var out bytes.Buffer
	n, err := captureTrace(strings.NewReader(in), &out, "ndjson", 0, nil)
	if err != nil {
		t.Fatalf("captureTrace: %v", err)
	}
	if n != 1 {
		t.Errorf("n = %d, want 1 (blank lines ignored)", n)
	}
}

// writeFakeSubscriber writes an executable /bin/sh script that emits the given
// NDJSON on stdout regardless of its arguments, mimicking `<binary> subscribe`.
func writeFakeSubscriber(t *testing.T, ndjson string) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("fake /bin/sh subscriber not available on Windows")
	}
	p := filepath.Join(t.TempDir(), "fakesub")
	script := "#!/bin/sh\ncat <<'TRACE_EOF'\n" + ndjson + "TRACE_EOF\n"
	if err := os.WriteFile(p, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake subscriber: %v", err)
	}
	return p
}

//fusa:test REQ-RELAY-061
func TestRunTraceLive(t *testing.T) {
	bin := writeFakeSubscriber(t, ndjsonOf(t, sampleMsgs()...))
	var out, errb bytes.Buffer
	if err := runTrace(&out, &errb, []string{"--format", "json", bin}); err != nil {
		t.Fatalf("runTrace live: %v\nstderr: %s", err, errb.String())
	}
	var msgs []relay.Message
	if err := json.Unmarshal(out.Bytes(), &msgs); err != nil {
		t.Fatalf("live json: %v\n%s", err, out.String())
	}
	if len(msgs) != 3 {
		t.Errorf("captured %d messages live, want 3", len(msgs))
	}
}

//fusa:test REQ-RELAY-061
func TestRunTraceLiveCount(t *testing.T) {
	bin := writeFakeSubscriber(t, ndjsonOf(t, sampleMsgs()...))
	var out, errb bytes.Buffer
	if err := runTrace(&out, &errb, []string{"--count", "1", "--format", "ndjson", bin}); err != nil {
		t.Fatalf("runTrace live --count: %v", err)
	}
	if got := strings.Count(strings.TrimSpace(out.String()), "\n") + 1; got != 1 {
		t.Errorf("live --count 1 emitted %d lines, want 1", got)
	}
}

//fusa:test REQ-RELAY-061
func TestRunTraceUnknownFormat(t *testing.T) {
	var out, errb bytes.Buffer
	if err := runTrace(&out, &errb, []string{"--replay", "--from", "x", "--format", "xml"}); err == nil {
		t.Error("expected error for unknown format")
	}
}

//fusa:test REQ-RELAY-061
func TestRunTraceReplay(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "trace.ndjson")
	if err := os.WriteFile(file, []byte(ndjsonOf(t, sampleMsgs()...)), 0o644); err != nil {
		t.Fatalf("write trace: %v", err)
	}
	var out, errb bytes.Buffer
	if err := runTrace(&out, &errb, []string{"--replay", "--from", file, "--format", "json"}); err != nil {
		t.Fatalf("runTrace replay: %v", err)
	}
	var msgs []relay.Message
	if err := json.Unmarshal(out.Bytes(), &msgs); err != nil {
		t.Fatalf("replay json: %v", err)
	}
	if len(msgs) != 3 {
		t.Errorf("replayed %d messages, want 3", len(msgs))
	}
}

//fusa:test REQ-RELAY-061
func TestRunTraceReplayToOutputFile(t *testing.T) {
	dir := t.TempDir()
	in := filepath.Join(dir, "in.ndjson")
	outFile := filepath.Join(dir, "out.ndjson")
	if err := os.WriteFile(in, []byte(ndjsonOf(t, sampleMsgs()...)), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	var out, errb bytes.Buffer
	if err := runTrace(&out, &errb, []string{"--replay", "--from", in, "--output", outFile}); err != nil {
		t.Fatalf("runTrace: %v", err)
	}
	data, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}
	if strings.Count(strings.TrimSpace(string(data)), "\n")+1 != 3 {
		t.Errorf("output file should have 3 ndjson lines, got:\n%s", data)
	}
}

//fusa:test REQ-RELAY-061
func TestRunTraceUnknownProtocol(t *testing.T) {
	var out, errb bytes.Buffer
	err := runTrace(&out, &errb, []string{"--replay", "--from", "x", "--protocol", "FOO"})
	var code exitCode
	if !errors.As(err, &code) || int(code) != 2 {
		t.Errorf("unknown protocol should return exitCode(2), got %v", err)
	}
}

//fusa:test REQ-RELAY-061
func TestRunTraceReplayNoFrom(t *testing.T) {
	var out, errb bytes.Buffer
	err := runTrace(&out, &errb, []string{"--replay"})
	var code exitCode
	if !errors.As(err, &code) || int(code) != 2 {
		t.Errorf("--replay without --from should return exitCode(2), got %v", err)
	}
}
