// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"strings"
	"time"

	relay "github.com/SoundMatt/RELAY/v2"
	"github.com/SoundMatt/RELAY/v2/can"
	"github.com/SoundMatt/RELAY/v2/dds"
	"github.com/SoundMatt/RELAY/v2/lin"
	"github.com/SoundMatt/RELAY/v2/mqtt"
	"github.com/SoundMatt/RELAY/v2/rcp"
	"github.com/SoundMatt/RELAY/v2/someip"
)

// maxConvertInputBytes bounds how much of stdin `relay convert` will read
// before validation, so an unbounded/adversarial input stream can't exhaust
// memory. 512 MiB is an intentional practical DoS bound: it exceeds MQTT v5's
// 268,435,455-byte payload (spec §16) with room for base64/JSON overhead, but
// is deliberately below SOME/IP's theoretical 2³²−16-byte maximum (spec §16),
// which no real convert input approaches. A SOME/IP canonical value larger
// than this bound is rejected with exit 2.
const maxConvertInputBytes = 512 * 1024 * 1024

// referenceConvert is RELAY's reference implementation of the §11.2 convert
// driver: it decodes a canonical-type value for protocol p, validates it where
// a validator exists, and returns the lossless relay.Message with a zeroed
// timestamp so outputs are comparable across implementations. It is shared by
// the `relay convert` command and the `relay interop` reference participant.
//
//fusa:req REQ-RELAY-082
func referenceConvert(protocol string, value []byte) (relay.Message, error) {
	var msg relay.Message
	switch strings.ToUpper(protocol) {
	case "CAN":
		f, err := decode[can.Frame](value)
		if err != nil {
			return msg, err
		}
		if err := can.ValidateFrame(f); err != nil {
			return msg, err
		}
		msg = f.ToMessage()
	case "DDS":
		s, err := decode[dds.Sample](value)
		if err != nil {
			return msg, err
		}
		msg = s.ToMessage()
	case "LIN":
		f, err := decode[lin.Frame](value)
		if err != nil {
			return msg, err
		}
		if err := lin.ValidateFrame(f); err != nil {
			return msg, err
		}
		msg = f.ToMessage()
	case "MQTT":
		m, err := decode[mqtt.Message](value)
		if err != nil {
			return msg, err
		}
		msg = m.ToMessage()
	case "RCP":
		m, err := decode[rcp.Message](value)
		if err != nil {
			return msg, err
		}
		msg = m.ToMessage()
	case "SOMEIP":
		m, err := decode[someip.Message](value)
		if err != nil {
			return msg, err
		}
		if err := m.Validate(); err != nil {
			return msg, err
		}
		msg = m.ToMessage()
	default:
		return msg, fmt.Errorf("unknown protocol %q (want CAN, DDS, LIN, MQTT, RCP, or SOMEIP)", protocol)
	}
	msg.Timestamp = time.Time{} // normalise for cross-implementation comparison
	return msg, nil
}

// decode unmarshals JSON into a fresh T.
func decode[T any](data []byte) (T, error) {
	var v T
	if err := json.Unmarshal(data, &v); err != nil {
		return v, fmt.Errorf("invalid canonical value: %w", err)
	}
	return v, nil
}

// runConvert implements `relay convert --protocol P [--format json]`: the
// reference §11.2 convert driver. It reads one canonical-type value as JSON on
// stdin and writes the resulting relay.Message as JSON on stdout.
//
//fusa:req REQ-RELAY-082
func runConvert(stdin io.Reader, stdout, stderr io.Writer, args []string) error {
	fs := flag.NewFlagSet("convert", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	protocol := fs.String("protocol", "", "Protocol of the canonical value (CAN, DDS, LIN, MQTT, RCP, SOMEIP)")
	format := fs.String("format", "json", "Output format: json")
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("relay convert: %w", err)
	}
	if *protocol == "" {
		fmt.Fprintln(stderr, "relay convert: --protocol is required")
		return exitCode(2)
	}
	if *format != "json" {
		fmt.Fprintf(stderr, "relay convert: unsupported format %q\n", *format)
		return exitCode(2)
	}
	// Cap stdin at maxConvertInputBytes so a runaway/adversarial input stream
	// can't exhaust memory before validation ever runs. The limit is set well
	// above the largest legitimate single-message payload spec §16 allows
	// (MQTT v5's 268,435,455 bytes), plus room for base64/JSON overhead.
	limited := io.LimitReader(stdin, maxConvertInputBytes+1)
	value, err := io.ReadAll(limited)
	if err != nil {
		fmt.Fprintf(stderr, "relay convert: read stdin: %v\n", err)
		return exitCode(1)
	}
	if int64(len(value)) > maxConvertInputBytes {
		fmt.Fprintf(stderr, "relay convert: stdin exceeds %d byte limit\n", maxConvertInputBytes)
		return exitCode(2)
	}
	msg, err := referenceConvert(*protocol, value)
	if err != nil {
		fmt.Fprintf(stderr, "relay convert: %v\n", err)
		return exitCode(1)
	}
	out, err := json.MarshalIndent(msg, "", "    ")
	if err != nil {
		return fmt.Errorf("relay convert: %w", err)
	}
	fmt.Fprintln(stdout, string(out))
	return nil
}
