// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package mqtt

import (
	"testing"

	relay "github.com/SoundMatt/RELAY"
)

//fusa:test REQ-RELAY-038
func TestQoSConstants(t *testing.T) {
	if AtMostOnce != 0 || AtLeastOnce != 1 || ExactlyOnce != 2 {
		t.Errorf("QoS constants wrong: %d %d %d", AtMostOnce, AtLeastOnce, ExactlyOnce)
	}
}

//fusa:test REQ-RELAY-039
func TestMessageRoundTrip(t *testing.T) {
	orig := Message{
		Topic:    "sensors/temp",
		Payload:  []byte{42},
		QoS:      AtLeastOnce,
		Retained: true,
	}
	msg := orig.ToMessage()
	if msg.Protocol != relay.MQTT {
		t.Errorf("Protocol = %v, want MQTT", msg.Protocol)
	}
	if msg.ID != "sensors/temp" {
		t.Errorf("ID = %q", msg.ID)
	}
	if msg.Meta["mqtt.qos"] != "1" {
		t.Errorf("mqtt.qos = %q, want %q", msg.Meta["mqtt.qos"], "1")
	}
	if msg.Meta["mqtt.retained"] != "true" {
		t.Errorf("mqtt.retained = %q, want %q", msg.Meta["mqtt.retained"], "true")
	}

	got, err := FromMessage(msg)
	if err != nil {
		t.Fatalf("FromMessage: %v", err)
	}
	if got.Topic != orig.Topic || got.QoS != orig.QoS || got.Retained != orig.Retained {
		t.Errorf("round-trip mismatch: %+v", got)
	}
}

//fusa:test REQ-RELAY-038
func TestMatchTopic(t *testing.T) {
	cases := []struct {
		filter, topic string
		want          bool
	}{
		{"a/b/c", "a/b/c", true},
		{"a/b/c", "a/b/d", false},
		{"a/+/c", "a/b/c", true},
		{"a/+/c", "a/b/d", false},
		{"a/#", "a/b/c/d", true},
		{"a/#", "a", true}, // §4.7.1.2: '#' matches the parent level too
		{"#", "any/thing", true},
		{"+/b", "a/b", true},
		// $ topics must not match wildcards
		{"#", "$SYS/test", false},
		{"+/test", "$SYS/test", false},
	}
	for _, tc := range cases {
		got := MatchTopic(tc.filter, tc.topic)
		if got != tc.want {
			t.Errorf("MatchTopic(%q, %q) = %v, want %v", tc.filter, tc.topic, got, tc.want)
		}
	}
}
