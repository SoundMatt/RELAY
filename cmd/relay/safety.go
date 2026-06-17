// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package main

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"runtime/debug"
	"sort"
	"strings"

	relay "github.com/SoundMatt/RELAY"
)

// --- relay sbom ---

type sbomComponent struct {
	Path    string `json:"path"`
	Version string `json:"version"`
}

type sbomDoc struct {
	Format      string          `json:"format"`
	Tool        string          `json:"tool"`
	SpecVersion string          `json:"spec_version"`
	Module      string          `json:"module"`
	Version     string          `json:"version"`
	GoVersion   string          `json:"go_version"`
	VCSRevision string          `json:"vcs_revision,omitempty"`
	VCSTime     string          `json:"vcs_time,omitempty"`
	VCSModified bool            `json:"vcs_modified,omitempty"`
	Components  []sbomComponent `json:"components"`
}

// runSBOM implements `relay sbom [--format json|text]`. It derives a software
// bill of materials from the embedded build information (spec §20 / v0.9).
//
//fusa:req REQ-RELAY-063
func runSBOM(stdout, stderr io.Writer, args []string) error {
	fs := flag.NewFlagSet("sbom", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	format := fs.String("format", "json", "Output format: json or text")
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("relay sbom: %w", err)
	}

	doc := buildSBOM()
	switch *format {
	case "json":
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "    ")
		return enc.Encode(doc)
	case "text":
		fmt.Fprintf(stdout, "module:  %s %s\n", doc.Module, doc.Version)
		fmt.Fprintf(stdout, "tool:    %s (spec %s)\n", doc.Tool, doc.SpecVersion)
		fmt.Fprintf(stdout, "go:      %s\n", doc.GoVersion)
		if doc.VCSRevision != "" {
			fmt.Fprintf(stdout, "vcs:     %s (%s) modified=%t\n", doc.VCSRevision, doc.VCSTime, doc.VCSModified)
		}
		fmt.Fprintf(stdout, "deps:    %d\n", len(doc.Components))
		for _, c := range doc.Components {
			fmt.Fprintf(stdout, "  - %s %s\n", c.Path, c.Version)
		}
	default:
		return fmt.Errorf("relay sbom: unknown format %q", *format)
	}
	return nil
}

func buildSBOM() sbomDoc {
	doc := sbomDoc{
		Format:      "relay-sbom/1",
		Tool:        "relay",
		SpecVersion: relay.SpecVersion,
		Module:      "github.com/SoundMatt/RELAY",
		GoVersion:   "unknown",
		Components:  []sbomComponent{},
	}
	bi, ok := debug.ReadBuildInfo()
	if !ok {
		return doc
	}
	doc.GoVersion = bi.GoVersion
	if bi.Main.Path != "" {
		doc.Module = bi.Main.Path
	}
	doc.Version = bi.Main.Version
	for _, d := range bi.Deps {
		doc.Components = append(doc.Components, sbomComponent{Path: d.Path, Version: d.Version})
	}
	for _, s := range bi.Settings {
		switch s.Key {
		case "vcs.revision":
			doc.VCSRevision = s.Value
		case "vcs.time":
			doc.VCSTime = s.Value
		case "vcs.modified":
			doc.VCSModified = s.Value == "true"
		}
	}
	return doc
}

// --- relay safety-case ---

type safetyCaseDoc struct {
	Tool         string `json:"tool"`
	SpecVersion  string `json:"spec_version"`
	Requirements struct {
		Total      int            `json:"total"`
		ByCategory map[string]int `json:"by_category"`
	} `json:"requirements"`
	Hazards struct {
		Total       int    `json:"total"`
		WorstASIL   string `json:"worst_asil"`
		SafetyGoals int    `json:"safety_goals"`
	} `json:"hazards"`
	Threats struct {
		Total       int    `json:"total"`
		WorstRisk   string `json:"worst_risk"`
		Mitigations int    `json:"mitigations"`
	} `json:"threats"`
	Evidence []string `json:"evidence"`
}

// runSafetyCase implements `relay safety-case [--format text|json|markdown]`.
// It assembles the embedded requirements, HARA, and TARA into a summary.
//
//fusa:req REQ-RELAY-064
func runSafetyCase(stdout, stderr io.Writer, args []string) error {
	fs := flag.NewFlagSet("safety-case", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	format := fs.String("format", "text", "Output format: text, json, or markdown")
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("relay safety-case: %w", err)
	}

	doc, err := buildSafetyCase()
	if err != nil {
		return fmt.Errorf("relay safety-case: %w", err)
	}

	switch *format {
	case "json":
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "    ")
		return enc.Encode(doc)
	case "text":
		fmt.Fprintf(stdout, "RELAY safety case (tool %s, spec %s)\n", doc.Tool, doc.SpecVersion)
		fmt.Fprintf(stdout, "  Requirements: %d\n", doc.Requirements.Total)
		fmt.Fprintf(stdout, "  Hazards:      %d (worst %s), %d safety goals\n", doc.Hazards.Total, doc.Hazards.WorstASIL, doc.Hazards.SafetyGoals)
		fmt.Fprintf(stdout, "  Threats:      %d (worst risk %s), %d mitigations\n", doc.Threats.Total, doc.Threats.WorstRisk, doc.Threats.Mitigations)
		fmt.Fprintf(stdout, "  Evidence:     %s\n", strings.Join(doc.Evidence, ", "))
	case "markdown":
		fmt.Fprintf(stdout, "# RELAY safety case\n\n")
		fmt.Fprintf(stdout, "Tool **%s**, spec **%s**.\n\n", doc.Tool, doc.SpecVersion)
		fmt.Fprintln(stdout, "| Evidence | Summary |")
		fmt.Fprintln(stdout, "|---|---|")
		fmt.Fprintf(stdout, "| Requirements | %d total |\n", doc.Requirements.Total)
		fmt.Fprintf(stdout, "| HARA | %d hazards (worst %s), %d safety goals |\n", doc.Hazards.Total, doc.Hazards.WorstASIL, doc.Hazards.SafetyGoals)
		fmt.Fprintf(stdout, "| TARA | %d threats (worst risk %s), %d mitigations |\n", doc.Threats.Total, doc.Threats.WorstRisk, doc.Threats.Mitigations)
	default:
		return fmt.Errorf("relay safety-case: unknown format %q", *format)
	}
	return nil
}

func buildSafetyCase() (safetyCaseDoc, error) {
	var doc safetyCaseDoc
	doc.Tool = "relay"
	doc.SpecVersion = relay.SpecVersion
	doc.Requirements.ByCategory = map[string]int{}
	doc.Evidence = relay.EvidenceNames()

	// Requirements.
	reqRaw, err := relay.Evidence("requirements")
	if err != nil {
		return doc, err
	}
	var reqs struct {
		Requirements []struct {
			Category string `json:"category"`
		} `json:"requirements"`
	}
	if err := json.Unmarshal(reqRaw, &reqs); err != nil {
		return doc, err
	}
	doc.Requirements.Total = len(reqs.Requirements)
	for _, r := range reqs.Requirements {
		doc.Requirements.ByCategory[r.Category]++
	}

	// HARA.
	haraRaw, err := relay.Evidence("hara")
	if err != nil {
		return doc, err
	}
	var hara struct {
		Hazards []struct {
			Risk struct {
				ASIL string `json:"asil"`
			} `json:"risk"`
		} `json:"hazards"`
		SafetyGoals []json.RawMessage `json:"safetyGoals"`
	}
	if err := json.Unmarshal(haraRaw, &hara); err != nil {
		return doc, err
	}
	doc.Hazards.Total = len(hara.Hazards)
	doc.Hazards.SafetyGoals = len(hara.SafetyGoals)
	for _, h := range hara.Hazards {
		doc.Hazards.WorstASIL = worse(doc.Hazards.WorstASIL, h.Risk.ASIL, asilRank)
	}

	// TARA.
	taraRaw, err := relay.Evidence("tara")
	if err != nil {
		return doc, err
	}
	var tara struct {
		Threats []struct {
			Risk struct {
				Level string `json:"level"`
			} `json:"risk"`
		} `json:"threats"`
		Mitigations []json.RawMessage `json:"mitigations"`
	}
	if err := json.Unmarshal(taraRaw, &tara); err != nil {
		return doc, err
	}
	doc.Threats.Total = len(tara.Threats)
	doc.Threats.Mitigations = len(tara.Mitigations)
	for _, th := range tara.Threats {
		doc.Threats.WorstRisk = worse(doc.Threats.WorstRisk, th.Risk.Level, riskRank)
	}
	return doc, nil
}

func asilRank(s string) int {
	switch strings.ToUpper(s) {
	case "QM":
		return 0
	case "ASIL-A":
		return 1
	case "ASIL-B":
		return 2
	case "ASIL-C":
		return 3
	case "ASIL-D":
		return 4
	}
	return -1
}

func riskRank(s string) int {
	switch strings.ToLower(s) {
	case "low":
		return 1
	case "medium":
		return 2
	case "high":
		return 3
	case "critical":
		return 4
	}
	return -1
}

// worse returns whichever of cur/next ranks higher under rank.
func worse(cur, next string, rank func(string) int) string {
	if cur == "" || rank(next) > rank(cur) {
		return next
	}
	return cur
}

// --- relay audit-pack ---

// runAuditPack implements `relay audit-pack [--output FILE]`. It bundles every
// embedded evidence artifact and JSON schema into a zip with a SHA-256 manifest.
//
//fusa:req REQ-RELAY-065
func runAuditPack(stdout, stderr io.Writer, args []string) error {
	fs := flag.NewFlagSet("audit-pack", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	output := fs.String("output", "relay-audit-pack.zip", "Output zip file")
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("relay audit-pack: %w", err)
	}

	f, err := os.Create(*output)
	if err != nil {
		return fmt.Errorf("relay audit-pack: %w", err)
	}
	defer func() { _ = f.Close() }()

	n, err := writeAuditPack(f)
	if err != nil {
		return fmt.Errorf("relay audit-pack: %w", err)
	}
	fmt.Fprintf(stdout, "wrote %s (%d artifacts + manifest)\n", *output, n)
	return nil
}

type manifestEntry struct {
	Name   string `json:"name"`
	SHA256 string `json:"sha256"`
	Bytes  int    `json:"bytes"`
}

// writeAuditPack writes the evidence zip to w and returns the artifact count.
// The manifest lists a SHA-256 over every other entry so post-hoc tampering is
// detectable.
//
//fusa:req REQ-RELAY-065
func writeAuditPack(w io.Writer) (int, error) {
	zw := zip.NewWriter(w)

	type item struct {
		zipPath string
		data    []byte
	}
	var items []item

	for _, name := range relay.EvidenceNames() {
		data, err := relay.Evidence(name)
		if err != nil {
			return 0, err
		}
		items = append(items, item{"evidence/" + name + evidenceExt(name), data})
	}
	schemas, err := relay.SchemaNames()
	if err != nil {
		return 0, err
	}
	for _, name := range schemas {
		data, err := relay.Schema(name)
		if err != nil {
			return 0, err
		}
		items = append(items, item{"schemas/" + name + ".json", data})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].zipPath < items[j].zipPath })

	manifest := struct {
		Format      string          `json:"format"`
		Tool        string          `json:"tool"`
		SpecVersion string          `json:"spec_version"`
		Files       []manifestEntry `json:"files"`
	}{Format: "relay-audit-pack/1", Tool: "relay", SpecVersion: relay.SpecVersion}

	for _, it := range items {
		sum := sha256.Sum256(it.data)
		manifest.Files = append(manifest.Files, manifestEntry{
			Name: it.zipPath, SHA256: hex.EncodeToString(sum[:]), Bytes: len(it.data),
		})
	}

	manifestJSON, err := json.MarshalIndent(manifest, "", "    ")
	if err != nil {
		return 0, err
	}
	if err := addZipEntry(zw, "manifest.json", manifestJSON); err != nil {
		return 0, err
	}
	for _, it := range items {
		if err := addZipEntry(zw, it.zipPath, it.data); err != nil {
			return 0, err
		}
	}
	if err := zw.Close(); err != nil {
		return 0, err
	}
	return len(items), nil
}

func addZipEntry(zw *zip.Writer, name string, data []byte) error {
	hw, err := zw.Create(name)
	if err != nil {
		return err
	}
	_, err = hw.Write(data)
	return err
}

func evidenceExt(name string) string {
	if name == "tool-safety-manual" {
		return ".md"
	}
	return ".json"
}
