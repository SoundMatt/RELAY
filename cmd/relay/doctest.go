// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package main

import (
	"fmt"
	"regexp"
	"strings"
)

// doctestMarkerRe matches a `<!-- doctest:KIND -->` comment line marking the
// fenced ```json block that immediately follows as a normative example this
// document commits to keeping genuinely valid (spec §20.7, §17 Requirement 19).
var doctestMarkerRe = regexp.MustCompile(`<!-- doctest:([a-z-]+) -->`)

// extractDoctestExamples scans a spec document line-by-line for
// `<!-- doctest:KIND -->` markers and returns the JSON bytes of the fenced
// ```json block that follows each one (skipping blank lines in between). A
// marker not followed by a properly-terminated fence is reported as an
// error — a silently-skipped marker would defeat the whole point of this
// mechanism: an example that's supposed to be checked but never actually is.
//
//fusa:req REQ-RELAY-102
func extractDoctestExamples(spec string) (map[string][][]byte, error) {
	lines := strings.Split(spec, "\n")
	out := map[string][][]byte{}
	for i := 0; i < len(lines); i++ {
		m := doctestMarkerRe.FindStringSubmatch(lines[i])
		if m == nil {
			continue
		}
		kind := m[1]
		j := i + 1
		for j < len(lines) && strings.TrimSpace(lines[j]) == "" {
			j++
		}
		if j >= len(lines) || strings.TrimSpace(lines[j]) != "```json" {
			return nil, fmt.Errorf("doctest:%s marker at line %d is not followed by a ```json fence", kind, i+1)
		}
		j++
		start := j
		for j < len(lines) && strings.TrimSpace(lines[j]) != "```" {
			j++
		}
		if j >= len(lines) {
			return nil, fmt.Errorf("doctest:%s marker at line %d: unterminated json fence", kind, i+1)
		}
		out[kind] = append(out[kind], []byte(strings.Join(lines[start:j], "\n")))
		i = j
	}
	return out, nil
}
