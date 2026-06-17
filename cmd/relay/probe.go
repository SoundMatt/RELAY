// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// probeResult describes one probed binary (spec §11/§12).
type probeResult struct {
	Binary      string   `json:"binary"`
	Conformant  bool     `json:"conformant"`
	Tool        string   `json:"tool,omitempty"`
	Protocol    string   `json:"protocol,omitempty"`
	Version     string   `json:"version,omitempty"`
	SpecVersion string   `json:"spec_version,omitempty"`
	Transports  []string `json:"transports,omitempty"`
	Adapt       bool     `json:"adapt,omitempty"`
	Error       string   `json:"error,omitempty"`
}

// runProbe implements `relay probe [--scan] [--match glob] [--format text|json] [binary...]`.
// With explicit binaries it probes each. With --scan it walks PATH for executables
// matching --match and reports the RELAY-conformant ones (spec §11).
//
//fusa:req REQ-RELAY-060
func runProbe(stdout, stderr io.Writer, args []string) error {
	fs := flag.NewFlagSet("probe", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	format := fs.String("format", "text", "Output format: text or json")
	scan := fs.Bool("scan", false, "Scan executables on PATH instead of taking explicit binaries")
	match := fs.String("match", "*", "Glob (matched against the file base name) used with --scan")
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("relay probe: %w", err)
	}

	var candidates []string
	if *scan {
		candidates = scanPath(*match)
	} else {
		candidates = fs.Args()
	}
	if len(candidates) == 0 {
		if *scan {
			fmt.Fprintln(stderr, "relay probe: no executables on PATH matched --match")
		} else {
			fmt.Fprintln(stderr, "Usage: relay probe [--scan] [--match glob] [--format text|json] <binary>...")
		}
		return exitCode(2)
	}

	var results []probeResult
	for _, c := range candidates {
		r := probeBinary(c)
		// In scan mode, silently drop non-conformant binaries (PATH is noisy);
		// in explicit mode, report them so the user sees why.
		if *scan && !r.Conformant {
			continue
		}
		results = append(results, r)
	}

	switch *format {
	case "json":
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "    ")
		return enc.Encode(results)
	case "text":
		printProbeText(stdout, results)
	default:
		return fmt.Errorf("relay probe: unknown format %q", *format)
	}
	return nil
}

// probeBinary runs version+capabilities against one binary and classifies it.
func probeBinary(binary string) probeResult {
	r := probeResult{Binary: binary}

	capsRaw, err := runBinaryCommand(binary, []string{"capabilities"})
	if err != nil {
		r.Error = fmt.Sprintf("capabilities failed: %v", err)
		return r
	}
	var caps struct {
		Kind        string   `json:"kind"`
		Tool        string   `json:"tool"`
		Version     string   `json:"version"`
		SpecVersion string   `json:"spec_version"`
		Transports  []string `json:"transports"`
		Adapt       bool     `json:"adapt"`
	}
	if err := json.Unmarshal(capsRaw, &caps); err != nil || caps.Kind != "capabilities" {
		r.Error = "not a RELAY-conformant capabilities document"
		return r
	}
	r.Conformant = true
	r.Tool = caps.Tool
	r.Version = caps.Version
	r.SpecVersion = caps.SpecVersion
	r.Transports = caps.Transports
	r.Adapt = caps.Adapt

	// Protocol comes from the version document (may be null for multi-protocol tools).
	if verRaw, err := runBinaryCommand(binary, []string{"version", "--format", "json"}); err == nil {
		var ver struct {
			Protocol *string `json:"protocol"`
		}
		if json.Unmarshal(verRaw, &ver) == nil && ver.Protocol != nil {
			r.Protocol = *ver.Protocol
		}
	}
	return r
}

// scanPath returns absolute paths of executable files on PATH whose base name
// matches the glob. Results are de-duplicated by base name (first on PATH wins).
func scanPath(glob string) []string {
	seen := map[string]bool{}
	var out []string
	for _, dir := range filepath.SplitList(os.Getenv("PATH")) {
		if dir == "" {
			continue
		}
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			name := e.Name()
			if seen[name] {
				continue
			}
			if ok, _ := filepath.Match(glob, name); !ok {
				continue
			}
			info, err := e.Info()
			if err != nil || info.IsDir() || !isExecutable(info.Mode()) {
				continue
			}
			seen[name] = true
			out = append(out, filepath.Join(dir, name))
		}
	}
	sort.Strings(out)
	return out
}

func isExecutable(mode fs.FileMode) bool {
	return mode&0o111 != 0
}

func printProbeText(w io.Writer, results []probeResult) {
	if len(results) == 0 {
		fmt.Fprintln(w, "No RELAY-conformant binaries found.")
		return
	}
	fmt.Fprintf(w, "%-28s %-10s %-8s %-8s %-6s %s\n", "BINARY", "TOOL", "PROTO", "VERSION", "SPEC", "TRANSPORTS")
	for _, r := range results {
		if !r.Conformant {
			fmt.Fprintf(w, "%-28s  not conformant: %s\n", filepath.Base(r.Binary), r.Error)
			continue
		}
		proto := r.Protocol
		if proto == "" {
			proto = "-"
		}
		fmt.Fprintf(w, "%-28s %-10s %-8s %-8s %-6s %s\n",
			filepath.Base(r.Binary), r.Tool, proto, r.Version, r.SpecVersion, strings.Join(r.Transports, ","))
	}
}
