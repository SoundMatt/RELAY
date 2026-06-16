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
	fmt.Fprintln(w, "  version    Print tool and spec version")
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

// exitCode is a sentinel error that carries a process exit code.
type exitCode int

func (e exitCode) Error() string { return fmt.Sprintf("exit %d", int(e)) }
