// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

// Package mqtt defines the canonical MQTT message types and relay.Message
// conversion per RELAY spec §15.4.
package mqtt

import (
	"strconv"
	"strings"
	"time"

	relay "github.com/SoundMatt/RELAY"
)

// QoS is the MQTT quality-of-service level.
//
//fusa:req REQ-RELAY-038
type QoS int

const (
	AtMostOnce  QoS = 0
	AtLeastOnce QoS = 1
	ExactlyOnce QoS = 2
)

// UserProperty is an MQTT v5 user-defined key/value pair.
//
//fusa:req REQ-RELAY-038
type UserProperty struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// Message is the canonical MQTT message.
//
//fusa:req REQ-RELAY-038
type Message struct {
	Topic           string         `json:"topic"`
	Payload         []byte         `json:"payload"`
	QoS             QoS            `json:"qos"`
	Retained        bool           `json:"retained,omitempty"`
	PacketID        uint16         `json:"packet_id,omitempty"`
	ResponseTopic   string         `json:"response_topic,omitempty"`
	CorrelationData []byte         `json:"correlation_data,omitempty"`
	UserProperties  []UserProperty `json:"user_properties,omitempty"`
	ContentType     string         `json:"content_type,omitempty"`
	ExpiryInterval  uint32         `json:"expiry_interval,omitempty"`
}

// ToMessage converts m to a relay.Message per §15.7.4.
//
//fusa:req REQ-RELAY-039
func (m Message) ToMessage() relay.Message {
	return relay.Message{
		Protocol:  relay.MQTT,
		ID:        m.Topic,
		Payload:   m.Payload,
		Timestamp: time.Now(),
		Meta: map[string]string{
			"mqtt.qos":      strconv.Itoa(int(m.QoS)),
			"mqtt.retained": strconv.FormatBool(m.Retained),
		},
	}
}

// FromMessage converts a relay.Message to a Message per §15.7.4.
//
//fusa:req REQ-RELAY-039
func FromMessage(msg relay.Message) (Message, error) {
	m := Message{
		Topic:   msg.ID,
		Payload: msg.Payload,
	}
	if q := msg.Meta["mqtt.qos"]; q != "" {
		v, _ := strconv.Atoi(q)
		m.QoS = QoS(v)
	}
	if r := msg.Meta["mqtt.retained"]; r != "" {
		m.Retained, _ = strconv.ParseBool(r)
	}
	return m, nil
}

// MatchTopic reports whether topic matches filter using MQTT §4.7 wildcard semantics.
// Topics beginning with '$' do not match wildcard subscriptions.
//
//fusa:req REQ-RELAY-038
func MatchTopic(filter, topic string) bool {
	if strings.HasPrefix(topic, "$") && (strings.HasPrefix(filter, "#") || strings.HasPrefix(filter, "+")) {
		return false
	}
	return matchSegments(strings.Split(filter, "/"), strings.Split(topic, "/"))
}

func matchSegments(filterParts, topicParts []string) bool {
	for i, fp := range filterParts {
		if fp == "#" {
			return true
		}
		if i >= len(topicParts) {
			return false
		}
		if fp != "+" && fp != topicParts[i] {
			return false
		}
	}
	return len(filterParts) == len(topicParts)
}
