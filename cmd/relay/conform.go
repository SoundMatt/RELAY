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
	"os/exec"
	"strings"
	"time"

	relay "github.com/SoundMatt/RELAY"
)

// conformSeverity is the severity of a conformance finding.
type conformSeverity string

const (
	sevFail conformSeverity = "FAIL"
	sevWarn conformSeverity = "WARN"
	sevPass conformSeverity = "PASS"
)

// conformFinding is a single conformance check result.
type conformFinding struct {
	Severity conformSeverity `json:"severity"`
	Req      string          `json:"req"`
	Message  string          `json:"message"`
}

func fail(req, msg string, args ...interface{}) conformFinding {
	return conformFinding{sevFail, req, fmt.Sprintf(msg, args...)}
}

func warn(req, msg string, args ...interface{}) conformFinding {
	return conformFinding{sevWarn, req, fmt.Sprintf(msg, args...)}
}

func pass(req, msg string) conformFinding {
	return conformFinding{sevPass, req, msg}
}

// conformResult is the overall conformance report.
type conformResult struct {
	Binary   string           `json:"binary"`
	Result   conformSeverity  `json:"result"`
	Findings []conformFinding `json:"findings"`
}

// runConform implements `relay conform [--format text|json] [--strict] <binary>`.
//
//fusa:req REQ-RELAY-052
func runConform(stdout, stderr io.Writer, args []string) error {
	fs := flag.NewFlagSet("conform", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	format := fs.String("format", "text", "Output format: text or json")
	strict := fs.Bool("strict", false, "Treat WARN as FAIL")
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("relay conform: %w", err)
	}
	if fs.NArg() == 0 {
		fmt.Fprintln(stderr, "Usage: relay conform [--format text|json] [--strict] <binary>")
		return exitCode(2)
	}

	cr := conformBinary(fs.Arg(0), *strict)

	switch *format {
	case "json":
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "    ")
		return enc.Encode(cr)
	case "text":
		printConformText(stdout, cr)
	default:
		return fmt.Errorf("relay conform: unknown format %q", *format)
	}

	if cr.Result == sevFail {
		return exitCode(1)
	}
	return nil
}

// conformBinary runs the §12 schema checks against one binary and returns the
// aggregated result. With strict, WARN findings escalate the result to FAIL.
//
//fusa:req REQ-RELAY-052
func conformBinary(binary string, strict bool) conformResult {
	var all []conformFinding

	// --- §17.7 / §12.1 version --format json ---
	versionJSON, err := runBinaryCommand(binary, []string{"version", "--format", "json"})
	if err != nil {
		all = append(all, fail("§17.7", "version --format json failed: %v", err))
	} else {
		all = append(all, validateVersionDoc(versionJSON)...)
	}

	// --- §17.7 / §12.2 capabilities ---
	capsJSON, err := runBinaryCommand(binary, []string{"capabilities"})
	if err != nil {
		all = append(all, fail("§17.7", "capabilities failed: %v", err))
	} else {
		all = append(all, validateCapabilitiesDoc(capsJSON)...)
	}

	// --- §17.7 / §12.3 status --format json ---
	statusJSON, err := runBinaryCommand(binary, []string{"status", "--format", "json"})
	if err != nil {
		// status may not accept --format json; try without
		statusJSON, err = runBinaryCommand(binary, []string{"status"})
		if err != nil {
			all = append(all, fail("§17.7", "status command failed: %v", err))
		} else {
			all = append(all, validateStatusDoc(statusJSON)...)
		}
	} else {
		all = append(all, validateStatusDoc(statusJSON)...)
	}

	// Compute overall result.
	result := sevPass
	for _, f := range all {
		if f.Severity == sevFail {
			result = sevFail
			break
		}
		if f.Severity == sevWarn && strict {
			result = sevFail
		} else if f.Severity == sevWarn && result == sevPass {
			result = sevWarn
		}
	}
	return conformResult{Binary: binary, Result: result, Findings: all}
}

func printConformText(w io.Writer, cr conformResult) {
	for _, f := range cr.Findings {
		fmt.Fprintf(w, "%-4s  %s  %s\n", f.Severity, f.Req, f.Message)
	}
	fmt.Fprintln(w, strings.Repeat("─", 60))
	fmt.Fprintf(w, "RESULT: %s  binary=%s\n", cr.Result, cr.Binary)
}

// runBinaryCommand executes binary with args and returns stdout.
// Returns an error if the binary exits non-zero or times out.
func runBinaryCommand(binary string, args []string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	var out bytes.Buffer
	cmd := exec.CommandContext(ctx, binary, args...)
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

// schemaCheck validates raw JSON output against the named embedded schema and
// returns one finding per result: a single FAIL for malformed JSON or a missing
// schema, one FAIL per schema violation, or a single PASS when the document
// conforms. ref is the spec section to attribute findings to.
//
//fusa:req REQ-RELAY-058
func schemaCheck(name, ref string, data []byte) (doc map[string]interface{}, findings []conformFinding) {
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, []conformFinding{fail(ref, "%s output is not valid JSON: %v", name, err)}
	}
	schemaJSON, err := relay.Schema(name)
	if err != nil {
		return doc, []conformFinding{fail(ref, "no embedded schema %q: %v", name, err)}
	}
	var asAny interface{}
	_ = json.Unmarshal(data, &asAny)
	violations := validateSchema(schemaJSON, asAny)
	if len(violations) == 0 {
		return doc, []conformFinding{pass(ref, fmt.Sprintf("conforms to %s schema", schemaTitle(schemaJSON)))}
	}
	for _, v := range violations {
		findings = append(findings, fail(ref, "%s schema: %s", schemaTitle(schemaJSON), v))
	}
	return doc, findings
}

// validateVersionDoc validates a version --format json response per §12.1.
//
//fusa:req REQ-RELAY-053
//fusa:req REQ-RELAY-047
//fusa:req REQ-RELAY-079
func validateVersionDoc(data []byte) []conformFinding {
	doc, fs := schemaCheck("cli-version", "§12.1", data)
	if doc == nil {
		return fs
	}
	// protocol / protocol_int are required for single-protocol implementations;
	// null or absent is acceptable for multi-protocol tooling (WARN not FAIL).
	if doc["protocol"] == nil {
		fs = append(fs, warn("§12.1", "version doc: protocol is null (acceptable for multi-protocol tools)"))
	}
	return fs
}

// validateCapabilitiesDoc validates a capabilities response per §12.2.
//
//fusa:req REQ-RELAY-054
//fusa:req REQ-RELAY-048
func validateCapabilitiesDoc(data []byte) []conformFinding {
	doc, fs := schemaCheck("cli-capabilities", "§12.2", data)
	if doc == nil {
		return fs
	}

	// The schema only asserts that commands contains "version"; §17.7 also
	// requires "capabilities" and "status".
	if cmds, ok := doc["commands"].([]interface{}); ok {
		cmdSet := map[string]bool{}
		for _, c := range cmds {
			if s, ok := c.(string); ok {
				cmdSet[s] = true
			}
		}
		for _, required := range []string{"capabilities", "status"} {
			if !cmdSet[required] {
				fs = append(fs, fail("§17.7", "capabilities doc: commands does not include %q", required))
			}
		}
	}

	// adapt=false is valid (no Adapt() exported) but worth flagging (§10.3).
	if adapt, ok := doc["adapt"].(bool); ok && !adapt {
		fs = append(fs, warn("§17.6", "capabilities doc: adapt=false (Adapt() not exported)"))
	}

	return fs
}

// validateStatusDoc validates a status response per §12.3.
//
//fusa:req REQ-RELAY-055
//fusa:req REQ-RELAY-049
func validateStatusDoc(data []byte) []conformFinding {
	_, fs := schemaCheck("cli-status", "§12.3", data)
	return fs
}
