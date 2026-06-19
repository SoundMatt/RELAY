// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

// Command relay is the RELAY CLI.
// Usage: relay <command> [flags]
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"runtime"

	relay "github.com/SoundMatt/RELAY"
)

const toolVersion = "0.1.0"

func main() {
	if err := run(os.Stdout, os.Stderr, os.Args[1:]); err != nil {
		var code exitCode
		if errors.As(err, &code) {
			os.Exit(int(code))
		}
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// run is the testable entry point. stdout/stderr are injected for testing.
func run(stdout, stderr io.Writer, args []string) error {
	if len(args) == 0 {
		printUsage(stderr)
		return exitCode(2)
	}
	switch args[0] {
	case "version":
		return runVersion(stdout, args[1:])
	case "capabilities":
		return runCapabilities(stdout, args[1:])
	case "status":
		return runStatus(stdout, args[1:])
	case "conform":
		return runConform(stdout, stderr, args[1:])
	case "convert":
		return runConvert(os.Stdin, stdout, stderr, args[1:])
	case "interop":
		return runInterop(stdout, stderr, args[1:])
	case "crossbar":
		return runCrossbar(stdout, stderr, args[1:])
	case "probe":
		return runProbe(stdout, stderr, args[1:])
	case "trace":
		return runTrace(stdout, stderr, args[1:])
	case "report":
		return runReport(stdout, stderr, args[1:])
	case "sbom":
		return runSBOM(stdout, stderr, args[1:])
	case "safety-case":
		return runSafetyCase(stdout, stderr, args[1:])
	case "audit-pack":
		return runAuditPack(stdout, stderr, args[1:])
	case "compare":
		return runCompare(stdout, stderr, args[1:])
	case "versions":
		return runVersions(stdout, stderr, args[1:])
	case "serve":
		return runServe(stdout, stderr, args[1:])
	case "--help", "-h", "help":
		printUsage(stdout)
		return nil
	default:
		fmt.Fprintf(stderr, "relay: unknown command %q\n\n", args[0])
		printUsage(stderr)
		return exitCode(2)
	}
}

func printUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage: relay <command> [flags]")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Commands:")
	fmt.Fprintln(w, "  version         Print tool and spec version")
	fmt.Fprintln(w, "  capabilities    Print RELAY tooling capabilities document")
	fmt.Fprintln(w, "  status          Print RELAY tooling status document")
	fmt.Fprintln(w, "  conform <bin>   Verify that <bin> conforms to the RELAY spec")
	fmt.Fprintln(w, "  convert         Reference canonical-value → relay.Message conversion (stdin→stdout)")
	fmt.Fprintln(w, "  interop <bin>   Check implementations are behaviourally interchangeable")
	fmt.Fprintln(w, "  crossbar        Route relay.Messages between protocol spokes (--config)")
	fmt.Fprintln(w, "  probe           Discover RELAY-conformant binaries")
	fmt.Fprintln(w, "  trace           Capture or replay a relay.Message stream")
	fmt.Fprintln(w, "  report          Cross-implementation conformance report")
	fmt.Fprintln(w, "  sbom            Print the software bill of materials")
	fmt.Fprintln(w, "  safety-case     Summarise the safety evidence set")
	fmt.Fprintln(w, "  audit-pack      Bundle all safety evidence into a zip")
	fmt.Fprintln(w, "  compare         Compare two implementations for interchangeability")
	fmt.Fprintln(w, "  versions        List implementations and their spec alignment")
	fmt.Fprintln(w, "  serve           Serve a web dashboard, JSON API, and status badge")
}

// runVersion implements `relay version [--format text|json]`.
//
//fusa:req REQ-RELAY-021
//fusa:req REQ-RELAY-022
func runVersion(w io.Writer, args []string) error {
	fs := flag.NewFlagSet("version", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	format := fs.String("format", "text", "Output format: text or json")
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("relay version: %w", err)
	}
	switch *format {
	case "text":
		fmt.Fprintf(w, "relay %s (spec %s, %s)\n", toolVersion, relay.SpecVersion, runtime.Version())
	case "json":
		doc := struct {
			Tool        string `json:"tool"`
			SpecVersion string `json:"spec_version"`
			Version     string `json:"version"`
			Language    string `json:"language"`
			Runtime     string `json:"runtime"`
		}{
			Tool:        "relay",
			SpecVersion: relay.SpecVersion,
			Version:     toolVersion,
			Language:    "go",
			Runtime:     runtime.Version(),
		}
		enc := json.NewEncoder(w)
		enc.SetIndent("", "    ")
		return enc.Encode(doc)
	default:
		return fmt.Errorf("relay version: unknown format %q: must be text or json", *format)
	}
	return nil
}

// runCapabilities implements `relay capabilities`.
// RELAY is a multi-protocol spec and tooling layer, not a protocol implementation,
// so protocol and protocol_int are omitted and adapt is false.
//
//fusa:req REQ-RELAY-029
func runCapabilities(w io.Writer, _ []string) error {
	doc := struct {
		Kind               string   `json:"kind"`
		Tool               string   `json:"tool"`
		Version            string   `json:"version"`
		SpecVersion        string   `json:"spec_version"`
		Commands           []string `json:"commands"`
		Transports         []string `json:"transports"`
		Features           []string `json:"features"`
		Interfaces         []string `json:"interfaces"`
		OptionalInterfaces []string `json:"optional_interfaces"`
		Adapt              bool     `json:"adapt"`
	}{
		Kind:               "capabilities",
		Tool:               "relay",
		Version:            toolVersion,
		SpecVersion:        relay.SpecVersion,
		Commands:           []string{"version", "capabilities", "status", "conform", "convert", "interop", "crossbar", "probe", "trace", "report", "sbom", "safety-case", "audit-pack", "compare", "versions", "serve"},
		Transports:         []string{},
		Features:           []string{},
		Interfaces:         []string{},
		OptionalInterfaces: []string{},
		Adapt:              false,
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "    ")
	return enc.Encode(doc)
}

// runStatus implements `relay status`. RELAY itself is always healthy and
// has no network connection to report (it is a spec/tooling layer).
//
//fusa:req REQ-RELAY-044
func runStatus(w io.Writer, _ []string) error {
	doc := struct {
		Protocol  interface{} `json:"protocol"`
		Tool      string      `json:"tool"`
		Version   string      `json:"version"`
		Healthy   bool        `json:"healthy"`
		Connected bool        `json:"connected"`
		Endpoint  string      `json:"endpoint"`
		Details   struct{}    `json:"details"`
	}{
		Protocol:  nil,
		Tool:      "relay",
		Version:   toolVersion,
		Healthy:   true,
		Connected: false,
		Endpoint:  "",
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "    ")
	return enc.Encode(doc)
}

// exitCode is a sentinel error that carries a process exit code.
type exitCode int

func (e exitCode) Error() string { return fmt.Sprintf("exit %d", int(e)) }
