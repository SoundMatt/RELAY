// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"html"
	"io"
	"path/filepath"
	"sort"
	"strings"
)

// reportEntry is one implementation's conformance summary.
type reportEntry struct {
	Binary   string          `json:"binary"`
	Tool     string          `json:"tool,omitempty"`
	Protocol string          `json:"protocol,omitempty"`
	Version  string          `json:"version,omitempty"`
	Result   conformSeverity `json:"result"`
	Pass     int             `json:"pass"`
	Warn     int             `json:"warn"`
	Fail     int             `json:"fail"`
}

// reportDoc is the aggregated cross-implementation conformance report.
type reportDoc struct {
	Result  conformSeverity `json:"result"`
	Summary struct {
		Pass int `json:"pass"`
		Warn int `json:"warn"`
		Fail int `json:"fail"`
	} `json:"summary"`
	Entries []reportEntry `json:"entries"`
}

// runReport implements
// `relay report [--scan] [--match glob] [--strict] [--format text|json|markdown|html] [binary...]`.
// It runs the conformance checks across every discovered implementation and
// produces a unified report (spec §17). Exit 1 if any implementation FAILs.
//
//fusa:req REQ-RELAY-062
func runReport(stdout, stderr io.Writer, args []string) error {
	fs := flag.NewFlagSet("report", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	format := fs.String("format", "text", "Output format: text, json, markdown, or html")
	scan := fs.Bool("scan", false, "Scan executables on PATH instead of taking explicit binaries")
	match := fs.String("match", "*", "Glob (matched against the file base name) used with --scan")
	strict := fs.Bool("strict", false, "Treat WARN as FAIL")
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("relay report: %w", err)
	}

	var candidates []string
	if *scan {
		candidates = scanPath(*match)
	} else {
		candidates = fs.Args()
	}
	if len(candidates) == 0 {
		fmt.Fprintln(stderr, "Usage: relay report [--scan] [--match glob] [--strict] [--format text|json|markdown|html] <binary>...")
		return exitCode(2)
	}

	doc := buildReport(candidates, *scan, *strict)

	switch *format {
	case "json":
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "    ")
		if err := enc.Encode(doc); err != nil {
			return err
		}
	case "text":
		renderReportText(stdout, doc)
	case "markdown":
		renderReportMarkdown(stdout, doc)
	case "html":
		renderReportHTML(stdout, doc)
	default:
		return fmt.Errorf("relay report: unknown format %q", *format)
	}

	if doc.Result == sevFail {
		return exitCode(1)
	}
	return nil
}

// buildReport conforms each candidate and aggregates the results. In scan mode,
// binaries that are not RELAY-conformant at all are skipped (PATH is noisy).
func buildReport(candidates []string, scan, strict bool) reportDoc {
	var doc reportDoc
	doc.Result = sevPass

	for _, c := range candidates {
		probe := probeBinary(c)
		if scan && !probe.Conformant {
			continue
		}
		cr := conformBinary(c, strict)
		e := reportEntry{Binary: c, Tool: probe.Tool, Protocol: probe.Protocol, Version: probe.Version, Result: cr.Result}
		for _, f := range cr.Findings {
			switch f.Severity {
			case sevPass:
				e.Pass++
			case sevWarn:
				e.Warn++
			case sevFail:
				e.Fail++
			}
		}
		doc.Entries = append(doc.Entries, e)

		switch cr.Result {
		case sevPass:
			doc.Summary.Pass++
		case sevWarn:
			doc.Summary.Warn++
			if doc.Result == sevPass {
				doc.Result = sevWarn
			}
		case sevFail:
			doc.Summary.Fail++
			doc.Result = sevFail
		}
	}

	sort.Slice(doc.Entries, func(i, j int) bool {
		return doc.Entries[i].Binary < doc.Entries[j].Binary
	})
	return doc
}

func renderReportText(w io.Writer, doc reportDoc) {
	if len(doc.Entries) == 0 {
		fmt.Fprintln(w, "No implementations to report.")
		return
	}
	fmt.Fprintf(w, "%-24s %-10s %-8s %-6s %s\n", "BINARY", "TOOL", "PROTO", "RESULT", "P/W/F")
	for _, e := range doc.Entries {
		fmt.Fprintf(w, "%-24s %-10s %-8s %-6s %d/%d/%d\n",
			filepath.Base(e.Binary), e.Tool, dash(e.Protocol), e.Result, e.Pass, e.Warn, e.Fail)
	}
	fmt.Fprintln(w, strings.Repeat("─", 60))
	fmt.Fprintf(w, "RESULT: %s  (%d pass, %d warn, %d fail)\n",
		doc.Result, doc.Summary.Pass, doc.Summary.Warn, doc.Summary.Fail)
}

func renderReportMarkdown(w io.Writer, doc reportDoc) {
	fmt.Fprintf(w, "# RELAY conformance report\n\n")
	fmt.Fprintf(w, "**Result: %s** — %d pass, %d warn, %d fail\n\n",
		doc.Result, doc.Summary.Pass, doc.Summary.Warn, doc.Summary.Fail)
	fmt.Fprintln(w, "| Binary | Tool | Protocol | Result | Pass | Warn | Fail |")
	fmt.Fprintln(w, "|---|---|---|---|---|---|---|")
	for _, e := range doc.Entries {
		fmt.Fprintf(w, "| %s | %s | %s | %s | %d | %d | %d |\n",
			filepath.Base(e.Binary), e.Tool, dash(e.Protocol), e.Result, e.Pass, e.Warn, e.Fail)
	}
}

func renderReportHTML(w io.Writer, doc reportDoc) {
	fmt.Fprintf(w, `<!DOCTYPE html>
<html lang="en"><head><meta charset="utf-8">
<title>RELAY conformance report</title>
<style>
 body{font-family:system-ui,sans-serif;margin:2rem;color:#1b1b1b}
 table{border-collapse:collapse;width:100%%}
 th,td{border:1px solid #ddd;padding:.4rem .6rem;text-align:left}
 th{background:#f4f4f4}
 .PASS{color:#0a7d28;font-weight:600}
 .WARN{color:#b06f00;font-weight:600}
 .FAIL{color:#c0271a;font-weight:600}
</style></head><body>
<h1>RELAY conformance report</h1>
<p>Overall result: <span class="%s">%s</span> — %d pass, %d warn, %d fail</p>
<table><thead><tr><th>Binary</th><th>Tool</th><th>Protocol</th><th>Result</th><th>Pass</th><th>Warn</th><th>Fail</th></tr></thead><tbody>
`, doc.Result, doc.Result, doc.Summary.Pass, doc.Summary.Warn, doc.Summary.Fail)
	for _, e := range doc.Entries {
		fmt.Fprintf(w, "<tr><td>%s</td><td>%s</td><td>%s</td><td class=\"%s\">%s</td><td>%d</td><td>%d</td><td>%d</td></tr>\n",
			html.EscapeString(filepath.Base(e.Binary)), html.EscapeString(e.Tool),
			html.EscapeString(dash(e.Protocol)), e.Result, e.Result, e.Pass, e.Warn, e.Fail)
	}
	fmt.Fprint(w, "</tbody></table></body></html>\n")
}

func dash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}
