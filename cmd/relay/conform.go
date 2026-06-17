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
	binary := fs.Arg(0)

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
		if f.Severity == sevWarn && *strict {
			result = sevFail
		} else if f.Severity == sevWarn && result == sevPass {
			result = sevWarn
		}
	}

	cr := conformResult{Binary: binary, Result: result, Findings: all}

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

	if result == sevFail {
		return exitCode(1)
	}
	return nil
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

// validateVersionDoc validates a version --format json response per §12.1.
//
//fusa:req REQ-RELAY-053
func validateVersionDoc(data []byte) []conformFinding {
	var doc map[string]interface{}
	if err := json.Unmarshal(data, &doc); err != nil {
		return []conformFinding{fail("§12.1", "version output is not valid JSON: %v", err)}
	}
	var fs []conformFinding
	for _, field := range []string{"tool", "version", "spec_version", "language", "runtime"} {
		if v, ok := doc[field]; !ok || v == nil || v == "" {
			fs = append(fs, fail("§12.1", "version doc missing required field %q", field))
		} else {
			fs = append(fs, pass("§12.1", fmt.Sprintf("version doc: %s=%v", field, v)))
		}
	}
	// protocol / protocol_int are required for single-protocol implementations;
	// null is acceptable for multi-protocol tooling (WARN not FAIL).
	if doc["protocol"] == nil {
		fs = append(fs, warn("§12.1", "version doc: protocol is null (acceptable for multi-protocol tools)"))
	} else {
		fs = append(fs, pass("§12.1", fmt.Sprintf("version doc: protocol=%v", doc["protocol"])))
	}
	if lang, ok := doc["language"].(string); ok {
		switch lang {
		case "go", "cpp", "rust":
			fs = append(fs, pass("§12.1", "version doc: language is a recognised value"))
		default:
			fs = append(fs, fail("§12.1", "version doc: language %q not in {go, cpp, rust}", lang))
		}
	}
	return fs
}

// validateCapabilitiesDoc validates a capabilities response per §12.2.
//
//fusa:req REQ-RELAY-054
func validateCapabilitiesDoc(data []byte) []conformFinding {
	var doc map[string]interface{}
	if err := json.Unmarshal(data, &doc); err != nil {
		return []conformFinding{fail("§12.2", "capabilities output is not valid JSON: %v", err)}
	}
	var fs []conformFinding

	// kind must be "capabilities"
	if doc["kind"] != "capabilities" {
		fs = append(fs, fail("§12.2", "capabilities doc: kind=%v, want \"capabilities\"", doc["kind"]))
	} else {
		fs = append(fs, pass("§12.2", "capabilities doc: kind=capabilities"))
	}

	for _, field := range []string{"tool", "version", "spec_version"} {
		if v, ok := doc[field]; !ok || v == nil || v == "" {
			fs = append(fs, fail("§12.2", "capabilities doc missing required field %q", field))
		} else {
			fs = append(fs, pass("§12.2", fmt.Sprintf("capabilities doc: %s=%v", field, v)))
		}
	}

	// commands must include version, capabilities, status.
	cmds, _ := doc["commands"].([]interface{})
	if cmds == nil {
		fs = append(fs, fail("§12.2", "capabilities doc: commands is missing or not an array"))
	} else {
		cmdSet := map[string]bool{}
		for _, c := range cmds {
			if s, ok := c.(string); ok {
				cmdSet[s] = true
			}
		}
		for _, required := range []string{"version", "capabilities", "status"} {
			if cmdSet[required] {
				fs = append(fs, pass("§17.7", fmt.Sprintf("capabilities doc: commands includes %q", required)))
			} else {
				fs = append(fs, fail("§17.7", "capabilities doc: commands does not include %q", required))
			}
		}
	}

	// adapt field (§10.3)
	if adapt, ok := doc["adapt"].(bool); ok {
		if adapt {
			fs = append(fs, pass("§17.6", "capabilities doc: adapt=true"))
		} else {
			fs = append(fs, warn("§17.6", "capabilities doc: adapt=false (Adapt() not exported)"))
		}
	} else {
		fs = append(fs, fail("§17.6", "capabilities doc: adapt field missing or wrong type"))
	}

	// spec_version must be non-empty (§17.12)
	if sv, ok := doc["spec_version"].(string); ok && sv != "" {
		fs = append(fs, pass("§17.12", fmt.Sprintf("spec_version=%q declared", sv)))
	} else {
		fs = append(fs, fail("§17.12", "capabilities doc: spec_version missing or empty"))
	}

	return fs
}

// validateStatusDoc validates a status response per §12.3.
//
//fusa:req REQ-RELAY-055
func validateStatusDoc(data []byte) []conformFinding {
	var doc map[string]interface{}
	if err := json.Unmarshal(data, &doc); err != nil {
		return []conformFinding{fail("§12.3", "status output is not valid JSON: %v", err)}
	}
	var fs []conformFinding
	for _, field := range []string{"tool", "version"} {
		if v, ok := doc[field]; !ok || v == nil || v == "" {
			fs = append(fs, fail("§12.3", "status doc missing required field %q", field))
		} else {
			fs = append(fs, pass("§12.3", fmt.Sprintf("status doc: %s=%v", field, v)))
		}
	}
	// healthy is a bool field.
	if _, ok := doc["healthy"].(bool); !ok {
		fs = append(fs, fail("§12.3", "status doc: healthy field missing or not a bool"))
	} else {
		fs = append(fs, pass("§12.3", "status doc: healthy field present"))
	}
	// connected is a bool field.
	if _, ok := doc["connected"].(bool); !ok {
		fs = append(fs, fail("§12.3", "status doc: connected field missing or not a bool"))
	} else {
		fs = append(fs, pass("§12.3", "status doc: connected field present"))
	}
	return fs
}
