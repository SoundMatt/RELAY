// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package router

import (
	"fmt"
	"strings"

	relay "github.com/SoundMatt/RELAY"
)

// Identity is the same-protocol converter used by a repeat route: it forwards
// the message unchanged.
//
//fusa:req REQ-RELAY-085
func Identity(m relay.Message) (relay.Message, error) { return m, nil }

// Retag returns a converter that re-tags a message for a different protocol,
// preserving the ID, payload, and Meta. It is the payload-preserving default
// for a cross-protocol (bridge) route; richer field/topic mapping is layered on
// top by registering a custom converter.
//
//fusa:req REQ-RELAY-085
func Retag(p relay.Protocol) Converter {
	return func(m relay.Message) (relay.Message, error) {
		m.Protocol = p
		return m, nil
	}
}

// Converters is the registry of named converters addressable from crossbar
// configuration. "identity" repeats; "to-can", "to-dds", … re-tag to the named
// protocol.
//
//fusa:req REQ-RELAY-085
var Converters = map[string]Converter{
	"identity":  Identity,
	"to-can":    Retag(relay.CAN),
	"to-dds":    Retag(relay.DDS),
	"to-lin":    Retag(relay.LIN),
	"to-mqtt":   Retag(relay.MQTT),
	"to-rcp":    Retag(relay.RCP),
	"to-someip": Retag(relay.SOMEIP),
}

// Lookup returns the named converter, or an error listing the known names.
//
//fusa:req REQ-RELAY-085
func Lookup(name string) (Converter, error) {
	if c, ok := Converters[name]; ok {
		return c, nil
	}
	known := make([]string, 0, len(Converters))
	for n := range Converters {
		known = append(known, n)
	}
	return nil, fmt.Errorf("router: unknown converter %q (known: %s)", name, strings.Join(known, ", "))
}

// DefaultConverter returns the converter to use for a route from a source
// protocol to a destination protocol when none is named: Identity when the
// protocols match (repeat), otherwise a Retag (bridge).
//
//fusa:req REQ-RELAY-085
func DefaultConverter(from, to relay.Protocol) Converter {
	if from == to {
		return Identity
	}
	return Retag(to)
}
