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
	"net/http"
)

// serveConfig holds the resolved set of implementations a server reports on.
type serveConfig struct {
	binaries []string
	strict   bool
}

// runServe implements `relay serve [--addr :8080] [--scan] [--match glob] [--strict] [binary...]`.
// It serves a web dashboard plus JSON APIs and an SVG status badge for the
// configured RELAY implementations (spec §11.2.1).
//
//fusa:req REQ-RELAY-068
func runServe(stdout, stderr io.Writer, args []string) error {
	fs := flag.NewFlagSet("serve", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	addr := fs.String("addr", ":8080", "Listen address")
	scan := fs.Bool("scan", false, "Scan executables on PATH instead of taking explicit binaries")
	match := fs.String("match", "*", "Glob (matched against the file base name) used with --scan")
	strict := fs.Bool("strict", false, "Treat WARN as FAIL")
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("relay serve: %w", err)
	}

	var binaries []string
	if *scan {
		binaries = scanPath(*match)
	} else {
		binaries = fs.Args()
	}
	if len(binaries) == 0 {
		fmt.Fprintln(stderr, "Usage: relay serve [--addr :8080] [--scan] [--match glob] [--strict] <binary>...")
		return exitCode(2)
	}

	h := newServeHandler(serveConfig{binaries: binaries, strict: *strict})
	fmt.Fprintf(stdout, "relay serve listening on %s (%d implementation(s))\n", *addr, len(binaries))
	return http.ListenAndServe(*addr, h)
}

// newServeHandler builds the dashboard/API/badge routes for cfg. It is split
// from runServe so it can be exercised with httptest.
//
//fusa:req REQ-RELAY-068
func newServeHandler(cfg serveConfig) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/api/v1/implementations", func(w http.ResponseWriter, r *http.Request) {
		results := make([]probeResult, 0, len(cfg.binaries))
		for _, b := range cfg.binaries {
			results = append(results, probeBinary(b))
		}
		writeJSON(w, results)
	})

	mux.HandleFunc("/api/v1/status", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, buildReport(cfg.binaries, false, cfg.strict))
	})

	mux.HandleFunc("/badge/status.svg", func(w http.ResponseWriter, r *http.Request) {
		doc := buildReport(cfg.binaries, false, cfg.strict)
		w.Header().Set("Content-Type", "image/svg+xml")
		w.Header().Set("Cache-Control", "no-cache")
		_, _ = io.WriteString(w, statusBadgeSVG(doc.Result))
	})

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		doc := buildReport(cfg.binaries, false, cfg.strict)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = io.WriteString(w, dashboardHTML(doc))
	})

	return mux
}

func writeJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	enc.SetIndent("", "    ")
	_ = enc.Encode(v)
}

// resultColor maps a conformance result to a badge colour.
func resultColor(s conformSeverity) string {
	switch s {
	case sevPass:
		return "#2ea043" // green
	case sevWarn:
		return "#bf8700" // amber
	default:
		return "#cf222e" // red
	}
}

// statusBadgeSVG renders a minimal two-segment status badge.
//
//fusa:req REQ-RELAY-069
func statusBadgeSVG(result conformSeverity) string {
	label := "relay"
	status := string(result)
	color := resultColor(result)
	// Fixed widths keep the SVG self-contained and predictable.
	lw, sw := 44, 60
	total := lw + sw
	return fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="20" role="img" aria-label="%s: %s">
<linearGradient id="s" x2="0" y2="100%%"><stop offset="0" stop-color="#bbb" stop-opacity=".1"/><stop offset="1" stop-opacity=".1"/></linearGradient>
<rect width="%d" height="20" fill="#555"/>
<rect x="%d" width="%d" height="20" fill="%s"/>
<rect width="%d" height="20" fill="url(#s)"/>
<g fill="#fff" text-anchor="middle" font-family="Verdana,Geneva,DejaVu Sans,sans-serif" font-size="11">
<text x="%d" y="14">%s</text>
<text x="%d" y="14">%s</text>
</g></svg>
`, total, label, status, lw, lw, sw, color, total, lw/2, label, lw+sw/2, status)
}

// dashboardHTML renders per-implementation status cards.
//
//fusa:req REQ-RELAY-068
func dashboardHTML(doc reportDoc) string {
	var b []byte
	add := func(s string) { b = append(b, s...) }
	add(`<!DOCTYPE html><html lang="en"><head><meta charset="utf-8">
<title>RELAY dashboard</title><meta http-equiv="refresh" content="10">
<style>
 body{font-family:system-ui,sans-serif;margin:2rem;background:#fafafa;color:#1b1b1b}
 .cards{display:flex;flex-wrap:wrap;gap:1rem}
 .card{border:1px solid #ddd;border-radius:8px;padding:1rem;min-width:200px;background:#fff}
 .r{font-weight:700} .PASS{color:#2ea043}.WARN{color:#bf8700}.FAIL{color:#cf222e}
 h1 small{font-weight:400;color:#666}
</style></head><body>`)
	add(fmt.Sprintf(`<h1>RELAY dashboard <small>overall: <span class="r %s">%s</span></small></h1>`, doc.Result, doc.Result))
	add(`<div class="cards">`)
	for _, e := range doc.Entries {
		add(fmt.Sprintf(`<div class="card"><div class="r %s">%s</div><div><b>%s</b></div><div>protocol: %s</div><div>version: %s</div><div>checks: %d✓ %d⚠ %d✗</div></div>`,
			e.Result, e.Result, html.EscapeString(e.Tool),
			html.EscapeString(dash(e.Protocol)), html.EscapeString(dash(e.Version)),
			e.Pass, e.Warn, e.Fail))
	}
	if len(doc.Entries) == 0 {
		add(`<p>No implementations configured.</p>`)
	}
	add(`</div></body></html>`)
	return string(b)
}
