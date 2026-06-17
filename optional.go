// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package relay

import "context"

// HealthStatus is the coarse health state of a node or connection.
//
//fusa:req REQ-RELAY-023
type HealthStatus int

// HealthStatus values.
const (
	HealthOK       HealthStatus = 0
	HealthDegraded HealthStatus = 1
	HealthDown     HealthStatus = 2
)

// Health is the response type for HealthProvider.Health().
//
//fusa:req REQ-RELAY-024
type Health struct {
	Status  HealthStatus `json:"status"`
	Details string       `json:"details,omitempty"`
}

// HealthProvider is an optional interface any protocol node may implement.
// Declared in the capabilities document as "HealthProvider" (§12.2).
//
//fusa:req REQ-RELAY-025
type HealthProvider interface {
	Health() Health
}

// Metrics is a set of monotonic runtime counters for a node. Field semantics
// are normative per spec §9.1 so that different protocol implementations report
// comparable numbers.
//
//fusa:req REQ-RELAY-026
type Metrics struct {
	// WriteCount counts accepted application sends (Send/Call/Publish, or LIN
	// SendHeader) that returned without error — once per call, never per subscriber.
	WriteCount uint64 `json:"write_count"`
	// DeliverCount counts successful enqueues onto a subscriber delivery channel,
	// counted once per receiving subscriber.
	DeliverCount uint64 `json:"deliver_count"`
	// DropCount counts samples discarded by back-pressure when a subscriber
	// channel is full, once per affected subscriber. Filter misses are not drops.
	DropCount uint64 `json:"drop_count"`
	// BytesWritten sums len(Payload) (application payload only, no framing) over
	// the sends counted by WriteCount.
	BytesWritten uint64 `json:"bytes_written"`
	// BytesDelivered sums len(Payload) over the deliveries counted by DeliverCount,
	// with the same per-subscriber multiplicity.
	BytesDelivered uint64 `json:"bytes_delivered"`
	// ErrorCount counts node operations that returned a non-nil error.
	ErrorCount uint64 `json:"error_count"`
}

// MetricsProvider is an optional interface any protocol node may implement.
// Declared in the capabilities document as "MetricsProvider" (§12.2).
//
//fusa:req REQ-RELAY-027
type MetricsProvider interface {
	Metrics() Metrics
}

// Drainer is an optional interface for graceful shutdown.
// Declared in the capabilities document as "Drainer" (§12.2).
//
// CloseWithDrain blocks until every message already accepted by the node has
// been delivered to all live subscribers, or until ctx is done, then closes
// (spec §9.2). A slow or abandoned consumer MUST NOT block past ctx: on ctx
// expiry the node closes immediately, undelivered messages are dropped (added
// to Metrics.DropCount), and ErrTimeout is returned. On clean drain it returns
// nil. After it returns the node is closed; a subsequent Close() is a no-op.
//
//fusa:req REQ-RELAY-028
type Drainer interface {
	CloseWithDrain(ctx context.Context) error
}
