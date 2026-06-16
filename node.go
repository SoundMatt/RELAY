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
type Caller interface {
	Node

	// Call sends req and blocks until a response arrives or ctx expires.
	// Returns ErrTimeout if ctx expires before a response.
	Call(ctx context.Context, req Message) (Message, error)
}

// BackPressurePolicy controls what happens when a subscription channel is full.
type BackPressurePolicy int

const (
	DropNewest BackPressurePolicy = iota // drop the arriving sample
	DropOldest                           // drop the oldest buffered sample
	Block                                // block the sender until space is available
)

// SubscriberConfig holds resolved subscriber options.
type SubscriberConfig struct {
	ChannelDepth int                // 0 means use implementation default (64)
	BackPressure BackPressurePolicy // default: DropNewest
}

// SubscriberOption configures a subscription.
type SubscriberOption func(*SubscriberConfig)

// WithChannelDepth sets the subscription channel buffer depth.
func WithChannelDepth(n int) SubscriberOption {
	return func(c *SubscriberConfig) { c.ChannelDepth = n }
}

// WithBackPressure sets the back-pressure policy applied when the channel is full.
func WithBackPressure(p BackPressurePolicy) SubscriberOption {
	return func(c *SubscriberConfig) { c.BackPressure = p }
}

// ApplySubscriberOpts applies opts in order to a zero SubscriberConfig and returns it.
func ApplySubscriberOpts(opts []SubscriberOption) SubscriberConfig {
	var c SubscriberConfig
	for _, o := range opts {
		o(&c)
	}
	return c
}

// ChanDepth returns c.ChannelDepth if explicitly set (> 0), otherwise defaultDepth.
func (c SubscriberConfig) ChanDepth(defaultDepth int) int {
	if c.ChannelDepth > 0 {
		return c.ChannelDepth
	}
	return defaultDepth
}
