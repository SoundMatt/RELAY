// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package router

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	relay "github.com/SoundMatt/RELAY"
)

// mockNode is an in-memory relay.Node for exercising the router.
type mockNode struct {
	proto   relay.Protocol
	in      chan relay.Message
	sendErr error

	mu   sync.Mutex
	sent []relay.Message
}

func newMock(p relay.Protocol) *mockNode {
	return &mockNode{proto: p, in: make(chan relay.Message, 16)}
}

func (m *mockNode) Protocol() relay.Protocol { return m.proto }
func (m *mockNode) Subscribe(...relay.SubscriberOption) (<-chan relay.Message, error) {
	return m.in, nil
}
func (m *mockNode) Close() error { return nil }
func (m *mockNode) Send(_ context.Context, msg relay.Message) error {
	if m.sendErr != nil {
		return m.sendErr
	}
	m.mu.Lock()
	m.sent = append(m.sent, msg)
	m.mu.Unlock()
	return nil
}
func (m *mockNode) received() []relay.Message {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]relay.Message(nil), m.sent...)
}

// waitFor polls cond until true or the deadline elapses.
func waitFor(t *testing.T, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(2 * time.Millisecond)
	}
	t.Fatal("condition not met within deadline")
}

//fusa:test REQ-RELAY-084
func TestAddSpokeValidation(t *testing.T) {
	r := New()
	if err := r.AddSpoke("", newMock(relay.CAN)); err == nil {
		t.Error("empty name must error")
	}
	if err := r.AddSpoke("x", nil); err == nil {
		t.Error("nil node must error")
	}
	if err := r.AddSpoke("a", newMock(relay.CAN)); err != nil {
		t.Fatalf("valid spoke: %v", err)
	}
	if err := r.AddSpoke("a", newMock(relay.CAN)); err == nil {
		t.Error("duplicate spoke must error")
	}
}

//fusa:test REQ-RELAY-084
func TestAddRouteValidation(t *testing.T) {
	r := New()
	_ = r.AddSpoke("a", newMock(relay.CAN))
	_ = r.AddSpoke("b", newMock(relay.CAN))
	if err := r.AddRoute(Route{From: "ghost", To: []string{"b"}}); err == nil {
		t.Error("unknown source must error")
	}
	if err := r.AddRoute(Route{From: "a", To: nil}); err == nil {
		t.Error("empty destinations must error")
	}
	if err := r.AddRoute(Route{From: "a", To: []string{"ghost"}}); err == nil {
		t.Error("unknown destination must error")
	}
	if err := r.AddRoute(Route{From: "a", To: []string{"b"}}); err != nil {
		t.Fatalf("valid route: %v", err)
	}
}

//fusa:test REQ-RELAY-084
func TestRunNoRoutes(t *testing.T) {
	r := New()
	_ = r.AddSpoke("a", newMock(relay.CAN))
	if err := r.Run(context.Background()); err == nil {
		t.Error("Run with no routes must error")
	}
}

//fusa:test REQ-RELAY-084
func TestRunForwardsFilterConvertFanout(t *testing.T) {
	src := newMock(relay.CAN)
	d1 := newMock(relay.CAN)
	d2 := newMock(relay.MQTT)
	r := New()
	_ = r.AddSpoke("src", src)
	_ = r.AddSpoke("d1", d1)
	_ = r.AddSpoke("d2", d2)
	// Fan-out to two destinations; drop messages whose ID == "drop"; the MQTT
	// leg re-tags the protocol (bridge).
	_ = r.AddRoute(Route{
		From: "src", To: []string{"d1", "d2"},
		Filter:  func(m relay.Message) bool { return m.ID != "drop" },
		Convert: nil, // exercised separately below
	})

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { _ = r.Run(ctx); close(done) }()

	src.in <- relay.Message{Protocol: relay.CAN, ID: "1", Payload: []byte{1}}
	src.in <- relay.Message{Protocol: relay.CAN, ID: "drop", Payload: []byte{9}}
	src.in <- relay.Message{Protocol: relay.CAN, ID: "2", Payload: []byte{2}}

	waitFor(t, func() bool { return len(d1.received()) == 2 && len(d2.received()) == 2 })
	cancel()
	<-done

	for _, d := range []*mockNode{d1, d2} {
		for _, m := range d.received() {
			if m.ID == "drop" {
				t.Error("filtered message must not be forwarded")
			}
		}
	}
	st := r.Stats()
	if st.Forwarded != 4 || st.Filtered != 1 {
		t.Errorf("stats = %+v, want Forwarded=4 Filtered=1", st)
	}
}

//fusa:test REQ-RELAY-084
//fusa:test REQ-RELAY-085
func TestRunConverterAndErrors(t *testing.T) {
	src := newMock(relay.CAN)
	good := newMock(relay.MQTT)
	bad := newMock(relay.MQTT)
	bad.sendErr = errors.New("sink down")
	r := New()
	_ = r.AddSpoke("src", src)
	_ = r.AddSpoke("good", good)
	_ = r.AddSpoke("bad", bad)
	// Converter that errors on ID "boom"; otherwise re-tags to MQTT.
	_ = r.AddRoute(Route{From: "src", To: []string{"good"}, Convert: func(m relay.Message) (relay.Message, error) {
		if m.ID == "boom" {
			return m, errors.New("convert fail")
		}
		return Retag(relay.MQTT)(m)
	}})
	_ = r.AddRoute(Route{From: "src", To: []string{"bad"}, Convert: Identity})

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { _ = r.Run(ctx); close(done) }()

	src.in <- relay.Message{Protocol: relay.CAN, ID: "ok"}
	src.in <- relay.Message{Protocol: relay.CAN, ID: "boom"}

	waitFor(t, func() bool { return len(good.received()) == 1 && r.Stats().Errors >= 2 })
	cancel()
	<-done

	if got := good.received()[0]; got.Protocol != relay.MQTT {
		t.Errorf("converter should have re-tagged to MQTT, got %v", got.Protocol)
	}
	// "ok" -> bad sink send error (1) + "boom" convert error (1) = 2 errors.
	if r.Stats().Errors < 2 {
		t.Errorf("expected >=2 errors, got %d", r.Stats().Errors)
	}
}
