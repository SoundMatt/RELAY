// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	relay "github.com/SoundMatt/RELAY"
)

// typeProtocol maps a golden-vector canonical type to its protocol name.
var typeProtocol = map[string]string{
	"can.Frame": "CAN", "dds.Sample": "DDS", "lin.Frame": "LIN",
	"mqtt.Message": "MQTT", "rcp.Status": "RCP", "someip.Message": "SOMEIP",
}

// interopVector is the subset of a golden vector that interop drives with.
type interopVector struct {
	Name     string          `json:"name"`
	Type     string          `json:"type"`
	Protocol string          `json:"-"`
	Value    json.RawMessage `json:"value"`
}

// interopCell is one participant's result for one vector.
type interopCell struct {
	Participant string `json:"participant"`
	OK          bool   `json:"ok"`         // produced a comparable relay.Message
	Equivalent  bool   `json:"equivalent"` // matches the reference
	Skipped     bool   `json:"skipped"`    // lacks convert (non-strict)
	Detail      string `json:"detail,omitempty"`
}

// interopVectorResult is the per-vector equivalence row.
type interopVectorResult struct {
	Vector   string        `json:"vector"`
	Protocol string        `json:"protocol"`
	Cells    []interopCell `json:"cells"`
}

// interopDoc is the full interop report.
type interopDoc struct {
	Reference string                `json:"reference"`
	Result    string                `json:"result"` // PASS / FAIL
	Vectors   []interopVectorResult `json:"vectors"`
}

// runInterop implements
// `relay interop [--protocol P] [--vectors DIR] [--strict] [--format text|json|markdown] <binary>...`.
// It verifies that implementations are behaviourally interchangeable by diffing
// each binary's `convert` output against RELAY's reference conversion for every
// golden vector (spec §11.2.1).
//
//fusa:req REQ-RELAY-083
func runInterop(stdout, stderr io.Writer, args []string) error {
	fs := flag.NewFlagSet("interop", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	protocol := fs.String("protocol", "", "Restrict to a single protocol (CAN, DDS, LIN, MQTT, RCP, SOMEIP)")
	vectorsDir := fs.String("vectors", "", "Directory of vector files (default: embedded golden vectors)")
	strict := fs.Bool("strict", false, "Treat a binary that lacks convert as a failure rather than a skip")
	format := fs.String("format", "text", "Output format: text, json, or markdown")
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("relay interop: %w", err)
	}
	binaries := fs.Args()

	vecs, err := loadInteropVectors(*vectorsDir, *protocol)
	if err != nil {
		fmt.Fprintf(stderr, "relay interop: %v\n", err)
		return exitCode(2)
	}
	if len(vecs) == 0 {
		fmt.Fprintln(stderr, "relay interop: no vectors match the given protocol")
		return exitCode(2)
	}

	// A spoke that advertises `convert` in its capabilities but whose convert
	// errors is non-conformant — that is a FAILURE, not a "skip". Only a binary
	// that does not advertise convert is legitimately skipped (in non-strict).
	advertisesConvert := map[string]bool{}
	for _, bin := range binaries {
		if caps, err := fetchCaps(bin); err == nil {
			for _, c := range caps.Commands {
				if c == "convert" {
					advertisesConvert[bin] = true
				}
			}
		}
	}

	doc := interopDoc{Reference: "relay (reference)", Result: "PASS"}
	for _, v := range vecs {
		row := interopVectorResult{Vector: v.Name, Protocol: v.Protocol}
		// Reference conversion is computed in-process from RELAY's canonical types.
		ref, refErr := referenceConvert(v.Protocol, v.Value)
		if refErr != nil {
			row.Cells = append(row.Cells, interopCell{Participant: doc.Reference, Detail: "reference convert failed: " + refErr.Error()})
			doc.Result = "FAIL"
			doc.Vectors = append(doc.Vectors, row)
			continue
		}
		refJSON := canonicalJSON(ref)
		row.Cells = append(row.Cells, interopCell{Participant: doc.Reference, OK: true, Equivalent: true})

		for _, bin := range binaries {
			cell := interopCell{Participant: filepath.Base(bin)}
			got, err := runConvertBinary(bin, v.Protocol, v.Value)
			switch {
			case err != nil:
				if *strict || advertisesConvert[bin] {
					// Advertised-but-broken (or strict): a conformance failure.
					cell.Detail = "convert failed: " + err.Error()
					doc.Result = "FAIL"
				} else {
					// Genuinely absent: skip in non-strict mode.
					cell.Skipped = true
					cell.Detail = "convert not advertised (skipped)"
				}
			default:
				cell.OK = true
				cell.Equivalent = bytes.Equal(canonicalJSON(got), refJSON)
				if !cell.Equivalent {
					cell.Detail = diffMessages(ref, got)
					doc.Result = "FAIL"
				}
			}
			row.Cells = append(row.Cells, cell)
		}
		doc.Vectors = append(doc.Vectors, row)
	}

	if err := renderInterop(stdout, doc, *format); err != nil {
		return err
	}
	if doc.Result != "PASS" {
		return exitCode(1)
	}
	return nil
}

// loadInteropVectors loads the embedded golden vectors (or those in dir) and
// filters to protocol if non-empty.
func loadInteropVectors(dir, protocol string) ([]interopVector, error) {
	var raws [][]byte
	if dir != "" {
		entries, err := os.ReadDir(dir)
		if err != nil {
			return nil, fmt.Errorf("read vectors dir: %w", err)
		}
		for _, e := range entries {
			if strings.HasSuffix(e.Name(), ".json") {
				b, err := os.ReadFile(filepath.Join(dir, e.Name()))
				if err != nil {
					return nil, err
				}
				raws = append(raws, b)
			}
		}
	} else {
		names, err := relay.VectorNames()
		if err != nil {
			return nil, err
		}
		for _, n := range names {
			b, err := relay.Vector(n)
			if err != nil {
				return nil, err
			}
			raws = append(raws, b)
		}
	}

	want := strings.ToUpper(protocol)
	var out []interopVector
	for _, b := range raws {
		var v interopVector
		if err := json.Unmarshal(b, &v); err != nil {
			continue // not a canonical vector (e.g. an error vector); skip
		}
		p, ok := typeProtocol[v.Type]
		if !ok {
			continue
		}
		v.Protocol = p
		if want != "" && p != want {
			continue
		}
		out = append(out, v)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

// runConvertBinary runs `<binary> convert --protocol P --format json`, piping
// value to stdin, and returns the parsed relay.Message.
func runConvertBinary(binary, protocol string, value []byte) (relay.Message, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	var out bytes.Buffer
	cmd := exec.CommandContext(ctx, binary, "convert", "--protocol", protocol, "--format", "json")
	cmd.Stdin = bytes.NewReader(value)
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return relay.Message{}, err
	}
	var m relay.Message
	if err := json.Unmarshal(out.Bytes(), &m); err != nil {
		return relay.Message{}, fmt.Errorf("convert output is not a relay.Message: %w", err)
	}
	return m, nil
}

// canonicalJSON marshals m with a zeroed timestamp for stable comparison.
func canonicalJSON(m relay.Message) []byte {
	m.Timestamp = time.Time{}
	b, _ := json.Marshal(m)
	return b
}

// diffMessages produces a short field-level difference summary.
func diffMessages(ref, got relay.Message) string {
	var d []string
	if ref.ID != got.ID {
		d = append(d, fmt.Sprintf("id %q!=%q", ref.ID, got.ID))
	}
	if !bytes.Equal(ref.Payload, got.Payload) {
		d = append(d, "payload differs")
	}
	for k, rv := range ref.Meta {
		if gv, ok := got.Meta[k]; !ok {
			d = append(d, "missing meta "+k)
		} else if gv != rv {
			d = append(d, fmt.Sprintf("meta %s %q!=%q", k, rv, gv))
		}
	}
	for k := range got.Meta {
		if _, ok := ref.Meta[k]; !ok {
			d = append(d, "extra meta "+k)
		}
	}
	if len(d) == 0 {
		return "differs"
	}
	return strings.Join(d, "; ")
}

func renderInterop(w io.Writer, doc interopDoc, format string) error {
	switch format {
	case "json":
		enc := json.NewEncoder(w)
		enc.SetIndent("", "    ")
		return enc.Encode(doc)
	case "markdown":
		fmt.Fprintf(w, "# RELAY interop report\n\n**Result: %s** (reference: %s)\n\n", doc.Result, doc.Reference)
		fmt.Fprintln(w, "| Vector | Protocol | Participant | Equivalent |")
		fmt.Fprintln(w, "|---|---|---|---|")
		for _, r := range doc.Vectors {
			for _, c := range r.Cells {
				fmt.Fprintf(w, "| %s | %s | %s | %s |\n", r.Vector, r.Protocol, c.Participant, interopVerdict(c))
			}
		}
		return nil
	case "text":
		for _, r := range doc.Vectors {
			fmt.Fprintf(w, "%s (%s)\n", r.Vector, r.Protocol)
			for _, c := range r.Cells {
				line := fmt.Sprintf("  %-22s %s", c.Participant, interopVerdict(c))
				if c.Detail != "" {
					line += "  — " + c.Detail
				}
				fmt.Fprintln(w, line)
			}
		}
		fmt.Fprintln(w, strings.Repeat("─", 60))
		fmt.Fprintf(w, "RESULT: %s\n", doc.Result)
		return nil
	default:
		return fmt.Errorf("relay interop: unsupported format %q", format)
	}
}

func interopVerdict(c interopCell) string {
	switch {
	case c.Skipped:
		return "SKIP"
	case !c.OK:
		return "ERROR"
	case c.Equivalent:
		return "EQUIVALENT"
	default:
		return "MISMATCH"
	}
}
