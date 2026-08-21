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

	relay "github.com/SoundMatt/RELAY/v2"
)

// typeProtocol maps a golden-vector canonical type to its protocol name.
var typeProtocol = map[string]string{
	"can.Frame": "CAN", "dds.Sample": "DDS", "lin.Frame": "LIN",
	"mqtt.Message": "MQTT", "rcp.Message": "RCP", "someip.Message": "SOMEIP",
}

// interopVector is the subset of a golden vector that interop drives with.
// Kind is empty for a canonical (accept-path) vector or "error" for a
// spec/vectors/errors/*.json reject-path vector (ExpectedError names the
// sentinel the reference implementation returns).
type interopVector struct {
	Name          string          `json:"name"`
	Type          string          `json:"type"`
	Protocol      string          `json:"-"`
	Value         json.RawMessage `json:"value"`
	Kind          string          `json:"kind"`
	ExpectedError string          `json:"error"`
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

// interopDoc is the full interop report — the relay-interop-matrix/1
// artifact (spec §20.1 item 3, Requirement 18) when persisted by CI.
type interopDoc struct {
	Kind          string                `json:"kind"`
	MatrixVersion string                `json:"matrix_version"`
	Reference     string                `json:"reference"`
	Participants  []string              `json:"participants"`
	Result        string                `json:"result"` // PASS / FAIL
	Vectors       []interopVectorResult `json:"vectors"`
	Regressions   []string              `json:"regressions,omitempty"`
}

// runInterop implements
// `relay interop [--protocol P] [--vectors DIR] [--strict] [--baseline FILE] [--format text|json|markdown] <binary>...`.
// It verifies that implementations are behaviourally interchangeable by diffing
// each binary's `convert` output against RELAY's reference conversion for every
// golden vector (spec §11.2.1). Every participant is compared against the same
// in-process reference rather than against each other pairwise: since the
// comparison is byte-equality (a transitive relation), "A equivalent to
// reference" and "B equivalent to reference" together already establish "A
// equivalent to B" — an O(N) hub comparison carries the same information as an
// O(N²) pairwise one here, without the redundant work (spec §20.1 item 3).
//
// --baseline compares this run's matrix against a previously-committed
// relay-interop-matrix/1 document (spec §17 Requirement 18) and reports a
// regression for any (vector, participant) cell that was EQUIVALENT in the
// baseline but is not in this run — distinct from a cell that was never
// equivalent to begin with, which --baseline does not newly fail on.
//
//fusa:req REQ-RELAY-083
//fusa:req REQ-RELAY-101
func runInterop(stdout, stderr io.Writer, args []string) error {
	fs := flag.NewFlagSet("interop", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	protocol := fs.String("protocol", "", "Restrict to a single protocol (CAN, DDS, LIN, MQTT, RCP, SOMEIP)")
	vectorsDir := fs.String("vectors", "", "Directory of vector files (default: embedded golden vectors)")
	strict := fs.Bool("strict", false, "Treat a binary that lacks convert as a failure rather than a skip")
	baseline := fs.String("baseline", "", "Path to a previously-generated relay-interop-matrix/1 document; fail if any of its EQUIVALENT cells regressed")
	format := fs.String("format", "text", "Output format: text, json, or markdown")
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("relay interop: %w", err)
	}
	binaries := fs.Args()
	if len(binaries) == 0 {
		fmt.Fprintln(stderr, "Usage: relay interop [--protocol P] [--vectors DIR] [--strict] [--baseline FILE] [--format text|json|markdown] <binary>...")
		return exitCode(2)
	}

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

	participants := make([]string, len(binaries))
	for i, bin := range binaries {
		participants[i] = filepath.Base(bin)
	}
	sort.Strings(participants)

	doc := interopDoc{
		Kind:          "relay-interop-matrix",
		MatrixVersion: "relay-interop-matrix/1",
		Reference:     "relay (reference)",
		Participants:  participants,
		Result:        "PASS",
	}
	for _, v := range vecs {
		row := interopVectorResult{Vector: v.Name, Protocol: v.Protocol}

		if v.Kind == "error" {
			// Reject-path vector: referenceConvert MUST fail (that's the
			// entire point of the fixture) — if it unexpectedly succeeds,
			// that's a bug in RELAY's own reference validation, not a
			// binary's problem.
			_, refErr := referenceConvert(v.Protocol, v.Value)
			if refErr == nil {
				row.Cells = append(row.Cells, interopCell{Participant: doc.Reference, Detail: fmt.Sprintf("reference convert unexpectedly SUCCEEDED on an invalid %q fixture (expected %s)", v.Name, v.ExpectedError)})
				doc.Result = "FAIL"
				doc.Vectors = append(doc.Vectors, row)
				continue
			}
			row.Cells = append(row.Cells, interopCell{Participant: doc.Reference, OK: true, Equivalent: true, Detail: "correctly rejected: " + refErr.Error()})

			for _, bin := range binaries {
				cell := interopCell{Participant: filepath.Base(bin)}
				_, err := runConvertBinary(bin, v.Protocol, v.Value)
				switch {
				case err != nil:
					// Binary also rejected it — behavioural equivalence on
					// the reject path. We check reject-vs-accept, not literal
					// error-string equality, since sentinel names are not
					// spelled identically across languages.
					cell.OK = true
					cell.Equivalent = true
					cell.Detail = "correctly rejected (expected " + v.ExpectedError + ")"
				case !advertisesConvert[bin] && !*strict:
					cell.Skipped = true
					cell.Detail = "convert not advertised (skipped)"
				default:
					// Binary accepted input the reference correctly rejects —
					// a real conformance failure, not a skip.
					cell.Detail = fmt.Sprintf("binary ACCEPTED invalid input that should have been rejected as %s", v.ExpectedError)
					doc.Result = "FAIL"
				}
				row.Cells = append(row.Cells, cell)
			}
			doc.Vectors = append(doc.Vectors, row)
			continue
		}

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

	if *baseline != "" {
		base, err := os.ReadFile(*baseline)
		if err != nil {
			fmt.Fprintf(stderr, "relay interop: --baseline: %v\n", err)
			return exitCode(2)
		}
		var baseDoc interopDoc
		if err := json.Unmarshal(base, &baseDoc); err != nil {
			fmt.Fprintf(stderr, "relay interop: --baseline: not a relay-interop-matrix/1 document: %v\n", err)
			return exitCode(2)
		}
		doc.Regressions = interopRegressions(baseDoc, doc)
		if len(doc.Regressions) > 0 {
			doc.Result = "FAIL"
		}
	}

	if err := renderInterop(stdout, doc, *format); err != nil {
		return err
	}
	if doc.Result != "PASS" {
		return exitCode(1)
	}
	return nil
}

// interopRegressions compares a fresh matrix against a previously-committed
// baseline and reports every (vector, participant) cell that was EQUIVALENT
// in the baseline but is not EQUIVALENT now. A cell that was never
// EQUIVALENT to begin with is not a regression — --baseline only guards
// against losing ground, not against pre-existing non-conformance (spec §17
// Requirement 18).
func interopRegressions(base, cur interopDoc) []string {
	baseEquivalent := map[string]bool{}
	for _, row := range base.Vectors {
		for _, c := range row.Cells {
			if c.Equivalent {
				baseEquivalent[row.Vector+"|"+c.Participant] = true
			}
		}
	}
	var regressions []string
	for _, row := range cur.Vectors {
		for _, c := range row.Cells {
			key := row.Vector + "|" + c.Participant
			if baseEquivalent[key] && !c.Equivalent {
				detail := c.Detail
				if detail == "" {
					detail = "no longer equivalent"
				}
				regressions = append(regressions, fmt.Sprintf("%s: %s regressed from EQUIVALENT (%s)", row.Vector, c.Participant, detail))
			}
		}
	}
	sort.Strings(regressions)
	return regressions
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
		if len(doc.Regressions) > 0 {
			fmt.Fprintln(w, "\n## Regressions")
			for _, r := range doc.Regressions {
				fmt.Fprintf(w, "- %s\n", r)
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
		if len(doc.Regressions) > 0 {
			fmt.Fprintln(w, strings.Repeat("─", 60))
			fmt.Fprintln(w, "REGRESSIONS:")
			for _, r := range doc.Regressions {
				fmt.Fprintf(w, "  %s\n", r)
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
