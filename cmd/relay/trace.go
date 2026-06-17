// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	relay "github.com/SoundMatt/RELAY"
)

// runTrace implements:
//
//	relay trace <binary> [--protocol P] [--count N] [--output FILE] [--format ndjson|json|text] [-- subscribe-args...]
//	relay trace --replay --from FILE [--protocol P] [--format ndjson|json|text]
//
// Live mode spawns `<binary> subscribe --format json` and captures the
// relay.Message NDJSON stream (spec §11.2). Replay mode renders a captured file.
//
//fusa:req REQ-RELAY-061
func runTrace(stdout, stderr io.Writer, args []string) error {
	fs := flag.NewFlagSet("trace", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	replay := fs.Bool("replay", false, "Replay a captured trace file instead of capturing live")
	from := fs.String("from", "", "Trace file to replay (with --replay)")
	protocol := fs.String("protocol", "", "Only include messages from this protocol (CAN, DDS, LIN, MQTT, RCP, SOMEIP)")
	count := fs.Int("count", 0, "Stop after N messages (0 = unlimited)")
	output := fs.String("output", "", "Write the trace to FILE instead of stdout")
	format := fs.String("format", "ndjson", "Output format: ndjson, json, or text")
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("relay trace: %w", err)
	}

	// Resolve the protocol filter, if any.
	var filter *relay.Protocol
	if *protocol != "" {
		p, ok := relay.ParseProtocol(*protocol)
		if !ok {
			fmt.Fprintf(stderr, "relay trace: unknown protocol %q\n", *protocol)
			return exitCode(2)
		}
		filter = &p
	}

	switch *format {
	case "ndjson", "json", "text":
	default:
		return fmt.Errorf("relay trace: unknown format %q", *format)
	}

	// Resolve the output writer.
	out := stdout
	if *output != "" {
		f, err := os.Create(*output)
		if err != nil {
			return fmt.Errorf("relay trace: %w", err)
		}
		defer func() { _ = f.Close() }()
		out = f
	}

	// --- Replay mode ---
	if *replay {
		if *from == "" {
			fmt.Fprintln(stderr, "relay trace --replay requires --from FILE")
			return exitCode(2)
		}
		f, err := os.Open(*from)
		if err != nil {
			return fmt.Errorf("relay trace: %w", err)
		}
		defer func() { _ = f.Close() }()
		_, err = captureTrace(f, out, *format, *count, filter)
		return err
	}

	// --- Live mode ---
	if fs.NArg() == 0 {
		fmt.Fprintln(stderr, "Usage: relay trace <binary> [--protocol P] [--count N] [--output FILE] [--format ndjson|json|text] [-- subscribe-args...]")
		return exitCode(2)
	}
	binary := fs.Arg(0)
	extra := fs.Args()[1:]

	subArgs := append([]string{"subscribe", "--format", "json"}, extra...)
	if *count > 0 {
		subArgs = append(subArgs, "--count", fmt.Sprintf("%d", *count))
	}

	cmd := exec.CommandContext(context.Background(), binary, subArgs...)
	pipe, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("relay trace: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("relay trace: cannot start %q: %w", binary, err)
	}
	_, capErr := captureTrace(pipe, out, *format, *count, filter)
	waitErr := cmd.Wait()
	if capErr != nil {
		return capErr
	}
	if waitErr != nil {
		return fmt.Errorf("relay trace: %q subscribe exited: %w", binary, waitErr)
	}
	return nil
}

// captureTrace reads relay.Message NDJSON from r, optionally filters by protocol,
// renders to w in the requested format, and stops after count messages (0 =
// unlimited). It returns the number of messages emitted.
//
// For ndjson it streams each accepted message as it arrives; for json and text
// it buffers and renders at the end (json needs a closing array, text aligns).
//
//fusa:req REQ-RELAY-061
func captureTrace(r io.Reader, w io.Writer, format string, count int, filter *relay.Protocol) (int, error) {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), 16*1024*1024)

	var buffered []relay.Message
	n := 0
	enc := json.NewEncoder(w)

	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		var msg relay.Message
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			return n, fmt.Errorf("relay trace: malformed message: %w", err)
		}
		if filter != nil && msg.Protocol != *filter {
			continue
		}
		n++
		switch format {
		case "ndjson":
			if err := enc.Encode(msg); err != nil {
				return n, err
			}
		default:
			buffered = append(buffered, msg)
		}
		if count > 0 && n >= count {
			break
		}
	}
	if err := sc.Err(); err != nil {
		return n, fmt.Errorf("relay trace: read error: %w", err)
	}

	switch format {
	case "json":
		enc.SetIndent("", "    ")
		if buffered == nil {
			buffered = []relay.Message{}
		}
		if err := enc.Encode(buffered); err != nil {
			return n, err
		}
	case "text":
		renderTraceText(w, buffered)
	}
	return n, nil
}

func renderTraceText(w io.Writer, msgs []relay.Message) {
	for _, m := range msgs {
		fmt.Fprintf(w, "%s  %-6s id=%-20s seq=%-6d bytes=%d\n",
			m.Timestamp.Format("15:04:05.000"), m.Protocol, m.ID, m.Seq, len(m.Payload))
	}
	fmt.Fprintf(w, "── %d message(s) ──\n", len(msgs))
}
