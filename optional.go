// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package relay

import "context"

// HealthStatus is the coarse health state of a node or connection.
//
//fusa:req REQ-RELAY-023
type HealthStatus int

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

// Metrics is a set of monotonic runtime counters for a node.
//
//fusa:req REQ-RELAY-026
type Metrics struct {
	WriteCount     uint64 `json:"write_count"`
	DeliverCount   uint64 `json:"deliver_count"`
	DropCount      uint64 `json:"drop_count"`
	BytesWritten   uint64 `json:"bytes_written"`
	BytesDelivered uint64 `json:"bytes_delivered"`
	ErrorCount     uint64 `json:"error_count"`
}

// MetricsProvider is an optional interface any protocol node may implement.
// Declared in the capabilities document as "MetricsProvider" (§12.2).
//
//fusa:req REQ-RELAY-027
type MetricsProvider interface {
	Metrics() Metrics
}

// Drainer is an optional interface for graceful shutdown.
// CloseWithDrain waits for in-flight messages to be delivered before closing.
// Declared in the capabilities document as "Drainer" (§12.2).
//
//fusa:req REQ-RELAY-028
type Drainer interface {
	CloseWithDrain(ctx context.Context) error
}
