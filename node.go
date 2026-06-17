// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package relay

import "context"

// Node is the protocol-agnostic application interface for pub/sub protocols.
// Applications program against Node; protocol choice is a constructor concern.
// All six protocol packages provide an Adapt() function returning Node or Caller.
//
// Lifecycle invariants (spec §6): Send after Close returns ErrClosed; Subscribe
// after Close returns ErrClosed; Close is idempotent; concurrent Send is safe.
//
//fusa:req REQ-RELAY-013
type Node interface {
	// Protocol returns the network protocol this node speaks.
	Protocol() Protocol

	// Send transmits msg. msg.ID carries the routing key per spec §4.2.
	// Returns ErrClosed, ErrNotConnected, ErrTimeout, or ErrPayloadTooLarge.
	Send(ctx context.Context, msg Message) error

	// Subscribe returns a channel of inbound messages.
	// The channel is closed when the node closes (spec §6.3).
	Subscribe(opts ...SubscriberOption) (<-chan Message, error)

	// Close closes the node. Idempotent per spec §6.1.
	Close() error
}

// Caller extends Node for protocols with request/response semantics (RCP, SOME/IP).
// Applications can probe: if c, ok := node.(relay.Caller); ok { ... }
//
//fusa:req REQ-RELAY-014
type Caller interface {
	Node

	// Call sends req and blocks until a response arrives or ctx expires.
	// Returns ErrTimeout if ctx expires before a response.
	Call(ctx context.Context, req Message) (Message, error)
}

// BackPressurePolicy controls what happens when a subscription channel is full.
//
//fusa:req REQ-RELAY-015
type BackPressurePolicy int

// BackPressurePolicy values.
const (
	DropNewest BackPressurePolicy = iota // drop the arriving sample
	DropOldest                           // drop the oldest buffered sample
	Block                                // block the sender until space is available
)

// SubscriberConfig holds resolved subscriber options.
//
//fusa:req REQ-RELAY-016
type SubscriberConfig struct {
	ChannelDepth int                // 0 means use implementation default (64)
	BackPressure BackPressurePolicy // default: DropNewest
	// EventID carries a protocol-specific subscription routing key.
	// Required by SOMEIP adapters (Adapt(Service).Subscribe must know which
	// event group to subscribe to). Set via WithEventID; ignored by all other
	// protocols. Zero means "not set".
	EventID uint32
	// TopicName carries the DDS topic name for a subscription.
	// Required by DDS adapters (Adapt(Participant).Subscribe must know which
	// topic to create a subscriber for). Set via WithTopic; ignored by all
	// other protocols. Empty string means "not set".
	TopicName string
}

// SubscriberOption configures a subscription.
//
//fusa:req REQ-RELAY-017
type SubscriberOption func(*SubscriberConfig)

// WithChannelDepth sets the subscription channel buffer depth.
//
//fusa:req REQ-RELAY-017
func WithChannelDepth(n int) SubscriberOption {
	return func(c *SubscriberConfig) { c.ChannelDepth = n }
}

// WithBackPressure sets the back-pressure policy applied when the channel is full.
//
//fusa:req REQ-RELAY-017
func WithBackPressure(p BackPressurePolicy) SubscriberOption {
	return func(c *SubscriberConfig) { c.BackPressure = p }
}

// WithEventID sets the protocol-specific subscription routing key.
// SOMEIP adapters (Adapt(Service).Subscribe) MUST read this option to
// determine which event group to subscribe to. All other protocol adapters
// ignore it. Returns ErrNotConnected if EventID is zero and the protocol
// requires it.
//
//fusa:req REQ-RELAY-051
func WithEventID(id uint32) SubscriberOption {
	return func(c *SubscriberConfig) { c.EventID = id }
}

// WithTopic sets the DDS topic name for a subscription.
// DDS adapters (Adapt(Participant).Subscribe) MUST read this option to
// determine which topic to subscribe to. All other protocol adapters ignore it.
// A DDS adapter MUST return ErrNotConnected if TopicName is empty.
//
//fusa:req REQ-RELAY-056
func WithTopic(name string) SubscriberOption {
	return func(c *SubscriberConfig) { c.TopicName = name }
}

// ApplySubscriberOpts applies opts in order to a zero SubscriberConfig and returns it.
//
//fusa:req REQ-RELAY-018
func ApplySubscriberOpts(opts []SubscriberOption) SubscriberConfig {
	var c SubscriberConfig
	for _, o := range opts {
		o(&c)
	}
	return c
}

// ChanDepth returns c.ChannelDepth if explicitly set (> 0), otherwise defaultDepth.
//
//fusa:req REQ-RELAY-019
func (c SubscriberConfig) ChanDepth(defaultDepth int) int {
	if c.ChannelDepth > 0 {
		return c.ChannelDepth
	}
	return defaultDepth
}
