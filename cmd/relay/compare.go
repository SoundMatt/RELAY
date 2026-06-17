// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strings"

	relay "github.com/SoundMatt/RELAY"
)

// capsDoc is the parsed capabilities document used by compare and versions.
type capsDoc struct {
	Kind               string   `json:"kind"`
	Tool               string   `json:"tool"`
	Protocol           string   `json:"protocol"`
	Version            string   `json:"version"`
	SpecVersion        string   `json:"spec_version"`
	Commands           []string `json:"commands"`
	Transports         []string `json:"transports"`
	Features           []string `json:"features"`
	Interfaces         []string `json:"interfaces"`
	OptionalInterfaces []string `json:"optional_interfaces"`
	Adapt              bool     `json:"adapt"`
}

func fetchCaps(binary string) (capsDoc, error) {
	var c capsDoc
	raw, err := runBinaryCommand(binary, []string{"capabilities"})
	if err != nil {
		return c, fmt.Errorf("capabilities failed: %w", err)
	}
	if err := json.Unmarshal(raw, &c); err != nil {
		return c, fmt.Errorf("capabilities is not valid JSON: %w", err)
	}
	if c.Kind != "capabilities" {
		return c, fmt.Errorf("not a RELAY capabilities document")
	}
	return c, nil
}

// compareResult is the delta report between two implementations.
type compareResult struct {
	BinaryA          string   `json:"binary_a"`
	BinaryB          string   `json:"binary_b"`
	Compatible       bool     `json:"compatible"`
	ProtocolMatch    bool     `json:"protocol_match"`
	SpecVersionMatch bool     `json:"spec_version_match"`
	CommandsOnlyA    []string `json:"commands_only_a"`
	CommandsOnlyB    []string `json:"commands_only_b"`
	FeaturesOnlyA    []string `json:"features_only_a"`
	FeaturesOnlyB    []string `json:"features_only_b"`
	InterfacesOnlyA  []string `json:"interfaces_only_a"`
	InterfacesOnlyB  []string `json:"interfaces_only_b"`
	Differences      []string `json:"differences"`
}

// runCompare implements `relay compare [--format text|json] <binaryA> <binaryB>`.
// It determines whether two implementations are interchangeable: same protocol,
// same spec version, and the same command/feature/interface surface (spec §11).
//
//fusa:req REQ-RELAY-066
func runCompare(stdout, stderr io.Writer, args []string) error {
	fs := flag.NewFlagSet("compare", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	format := fs.String("format", "text", "Output format: text or json")
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("relay compare: %w", err)
	}
	if fs.NArg() != 2 {
		fmt.Fprintln(stderr, "Usage: relay compare [--format text|json] <binaryA> <binaryB>")
		return exitCode(2)
	}

	a, errA := fetchCaps(fs.Arg(0))
	if errA != nil {
		return fmt.Errorf("relay compare: %s: %w", fs.Arg(0), errA)
	}
	b, errB := fetchCaps(fs.Arg(1))
	if errB != nil {
		return fmt.Errorf("relay compare: %s: %w", fs.Arg(1), errB)
	}

	res := compareCaps(fs.Arg(0), fs.Arg(1), a, b)

	switch *format {
	case "json":
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "    ")
		if err := enc.Encode(res); err != nil {
			return err
		}
	case "text":
		printCompareText(stdout, res)
	default:
		return fmt.Errorf("relay compare: unknown format %q", *format)
	}

	if !res.Compatible {
		return exitCode(1)
	}
	return nil
}

func compareCaps(nameA, nameB string, a, b capsDoc) compareResult {
	res := compareResult{BinaryA: nameA, BinaryB: nameB}
	res.ProtocolMatch = a.Protocol == b.Protocol
	res.SpecVersionMatch = a.SpecVersion == b.SpecVersion

	res.CommandsOnlyA, res.CommandsOnlyB = setDiff(a.Commands, b.Commands)
	res.FeaturesOnlyA, res.FeaturesOnlyB = setDiff(a.Features, b.Features)
	res.InterfacesOnlyA, res.InterfacesOnlyB = setDiff(a.Interfaces, b.Interfaces)

	if !res.ProtocolMatch {
		res.Differences = append(res.Differences, fmt.Sprintf("protocol: %q vs %q", a.Protocol, b.Protocol))
	}
	if !res.SpecVersionMatch {
		res.Differences = append(res.Differences, fmt.Sprintf("spec_version: %q vs %q", a.SpecVersion, b.SpecVersion))
	}
	for _, c := range res.CommandsOnlyA {
		res.Differences = append(res.Differences, "command only in A: "+c)
	}
	for _, c := range res.CommandsOnlyB {
		res.Differences = append(res.Differences, "command only in B: "+c)
	}
	for _, f := range res.FeaturesOnlyA {
		res.Differences = append(res.Differences, "feature only in A: "+f)
	}
	for _, f := range res.FeaturesOnlyB {
		res.Differences = append(res.Differences, "feature only in B: "+f)
	}
	for _, i := range res.InterfacesOnlyA {
		res.Differences = append(res.Differences, "interface only in A: "+i)
	}
	for _, i := range res.InterfacesOnlyB {
		res.Differences = append(res.Differences, "interface only in B: "+i)
	}

	// Interchangeable: same protocol, same spec version, and identical command,
	// feature, and interface surfaces.
	res.Compatible = res.ProtocolMatch && res.SpecVersionMatch &&
		len(res.CommandsOnlyA) == 0 && len(res.CommandsOnlyB) == 0 &&
		len(res.FeaturesOnlyA) == 0 && len(res.FeaturesOnlyB) == 0 &&
		len(res.InterfacesOnlyA) == 0 && len(res.InterfacesOnlyB) == 0
	return res
}

// setDiff returns the elements only in a and only in b.
func setDiff(a, b []string) (onlyA, onlyB []string) {
	setA, setB := map[string]bool{}, map[string]bool{}
	for _, x := range a {
		setA[x] = true
	}
	for _, x := range b {
		setB[x] = true
	}
	for x := range setA {
		if !setB[x] {
			onlyA = append(onlyA, x)
		}
	}
	for x := range setB {
		if !setA[x] {
			onlyB = append(onlyB, x)
		}
	}
	sort.Strings(onlyA)
	sort.Strings(onlyB)
	return onlyA, onlyB
}

func printCompareText(w io.Writer, r compareResult) {
	verdict := "COMPATIBLE"
	if !r.Compatible {
		verdict = "INCOMPATIBLE"
	}
	fmt.Fprintf(w, "%s vs %s\n", filepath.Base(r.BinaryA), filepath.Base(r.BinaryB))
	fmt.Fprintf(w, "  protocol match:     %t\n", r.ProtocolMatch)
	fmt.Fprintf(w, "  spec_version match: %t\n", r.SpecVersionMatch)
	if len(r.Differences) == 0 {
		fmt.Fprintln(w, "  no capability differences")
	} else {
		fmt.Fprintln(w, "  differences:")
		for _, d := range r.Differences {
			fmt.Fprintf(w, "    - %s\n", d)
		}
	}
	fmt.Fprintln(w, strings.Repeat("─", 50))
	fmt.Fprintf(w, "VERDICT: %s\n", verdict)
}

// --- relay versions ---

type versionEntry struct {
	Binary      string `json:"binary"`
	Tool        string `json:"tool"`
	Protocol    string `json:"protocol"`
	Version     string `json:"version"`
	SpecVersion string `json:"spec_version"`
	Aligned     bool   `json:"aligned"`
}

// runVersions implements `relay versions [--scan] [--match glob] [--format text|json] [binary...]`.
// It lists implementations and whether each is aligned with the spec version
// this relay tool implements (spec §11).
//
//fusa:req REQ-RELAY-067
func runVersions(stdout, stderr io.Writer, args []string) error {
	fs := flag.NewFlagSet("versions", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	format := fs.String("format", "text", "Output format: text or json")
	scan := fs.Bool("scan", false, "Scan executables on PATH instead of taking explicit binaries")
	match := fs.String("match", "*", "Glob (matched against the file base name) used with --scan")
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("relay versions: %w", err)
	}

	var candidates []string
	if *scan {
		candidates = scanPath(*match)
	} else {
		candidates = fs.Args()
	}
	if len(candidates) == 0 {
		fmt.Fprintln(stderr, "Usage: relay versions [--scan] [--match glob] [--format text|json] <binary>...")
		return exitCode(2)
	}

	var entries []versionEntry
	for _, c := range candidates {
		p := probeBinary(c)
		if *scan && !p.Conformant {
			continue
		}
		if !p.Conformant {
			entries = append(entries, versionEntry{Binary: c})
			continue
		}
		entries = append(entries, versionEntry{
			Binary: c, Tool: p.Tool, Protocol: p.Protocol, Version: p.Version,
			SpecVersion: p.SpecVersion, Aligned: p.SpecVersion == relay.SpecVersion,
		})
	}

	switch *format {
	case "json":
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "    ")
		return enc.Encode(entries)
	case "text":
		fmt.Fprintf(stdout, "RELAY tool implements spec %s\n", relay.SpecVersion)
		fmt.Fprintf(stdout, "%-24s %-10s %-8s %-8s %-6s %s\n", "BINARY", "TOOL", "PROTO", "VERSION", "SPEC", "ALIGNED")
		for _, e := range entries {
			if e.Tool == "" {
				fmt.Fprintf(stdout, "%-24s  (not conformant)\n", filepath.Base(e.Binary))
				continue
			}
			fmt.Fprintf(stdout, "%-24s %-10s %-8s %-8s %-6s %t\n",
				filepath.Base(e.Binary), e.Tool, dash(e.Protocol), e.Version, e.SpecVersion, e.Aligned)
		}
	default:
		return fmt.Errorf("relay versions: unknown format %q", *format)
	}
	return nil
}
