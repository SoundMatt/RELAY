// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package relay

import (
	"context"
	"testing"
)

//fusa:test REQ-RELAY-023
//fusa:test REQ-RELAY-024
//fusa:test REQ-RELAY-025
func TestHealthProvider(t *testing.T) {
	// Verify the constants are distinct and in range.
	statuses := []HealthStatus{HealthOK, HealthDegraded, HealthDown}
	seen := map[HealthStatus]bool{}
	for _, s := range statuses {
		if seen[s] {
			t.Errorf("HealthStatus %d is not unique", s)
		}
		seen[s] = true
	}
	if HealthOK != 0 || HealthDegraded != 1 || HealthDown != 2 {
		t.Errorf("HealthStatus values must be 0, 1, 2; got %d, %d, %d",
			HealthOK, HealthDegraded, HealthDown)
	}

	// Verify Health struct JSON fields.
	h := Health{Status: HealthOK, Details: "all good"}
	if h.Status != HealthOK {
		t.Errorf("Health.Status = %d, want %d", h.Status, HealthOK)
	}

	// Verify a type can satisfy HealthProvider.
	var _ HealthProvider = stubHealth{}
}

//fusa:test REQ-RELAY-026
//fusa:test REQ-RELAY-027
func TestMetricsProvider(t *testing.T) {
	m := Metrics{WriteCount: 5, DeliverCount: 4, DropCount: 1}
	if m.WriteCount != 5 {
		t.Errorf("WriteCount = %d, want 5", m.WriteCount)
	}
	var _ MetricsProvider = stubMetrics{}
}

//fusa:test REQ-RELAY-028
func TestDrainer(t *testing.T) {
	var _ Drainer = stubDrain{}
}

// stub types used only for compile-time interface assertions.

type stubHealth struct{}

func (stubHealth) Health() Health { return Health{Status: HealthOK} }

type stubMetrics struct{}

func (stubMetrics) Metrics() Metrics { return Metrics{} }

type stubDrain struct{}

func (stubDrain) CloseWithDrain(_ context.Context) error { return nil }
