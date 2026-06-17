// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package relay

import (
	"context"
	"testing"
)

// Compile-time checks: ensure the interface shapes are correct.
var _ Node = (*nodeStub)(nil)
var _ Caller = (*callerStub)(nil)

type nodeStub struct{}

func (nodeStub) Protocol() Protocol                                   { return CAN }
func (nodeStub) Send(_ context.Context, _ Message) error              { return nil }
func (nodeStub) Subscribe(...SubscriberOption) (<-chan Message, error) { return nil, nil }
func (nodeStub) Close() error                                         { return nil }

type callerStub struct{ nodeStub }

func (callerStub) Call(_ context.Context, _ Message) (Message, error) { return Message{}, nil }

//fusa:test REQ-RELAY-015
func TestBackPressurePolicyValues(t *testing.T) {
	if int(DropNewest) != 0 {
		t.Errorf("DropNewest = %d, want 0", DropNewest)
	}
	if int(DropOldest) != 1 {
		t.Errorf("DropOldest = %d, want 1", DropOldest)
	}
	if int(Block) != 2 {
		t.Errorf("Block = %d, want 2", Block)
	}
}

//fusa:test REQ-RELAY-016
//fusa:test REQ-RELAY-018
func TestApplySubscriberOptsDefaults(t *testing.T) {
	cfg := ApplySubscriberOpts(nil)
	if cfg.ChannelDepth != 0 {
		t.Errorf("default ChannelDepth = %d, want 0 (unset)", cfg.ChannelDepth)
	}
	if cfg.BackPressure != DropNewest {
		t.Errorf("default BackPressure = %d, want DropNewest", cfg.BackPressure)
	}
}

//fusa:test REQ-RELAY-017
func TestWithChannelDepth(t *testing.T) {
	cfg := ApplySubscriberOpts([]SubscriberOption{WithChannelDepth(128)})
	if cfg.ChannelDepth != 128 {
		t.Errorf("ChannelDepth = %d, want 128", cfg.ChannelDepth)
	}
}

//fusa:test REQ-RELAY-017
func TestWithBackPressure(t *testing.T) {
	cfg := ApplySubscriberOpts([]SubscriberOption{WithBackPressure(Block)})
	if cfg.BackPressure != Block {
		t.Errorf("BackPressure = %d, want Block", cfg.BackPressure)
	}
}

//fusa:test REQ-RELAY-019
func TestChanDepthDefault(t *testing.T) {
	cfg := SubscriberConfig{}
	if got := cfg.ChanDepth(64); got != 64 {
		t.Errorf("unset ChanDepth(64) = %d, want 64", got)
	}
}

//fusa:test REQ-RELAY-019
func TestChanDepthOverride(t *testing.T) {
	cfg := SubscriberConfig{ChannelDepth: 32}
	if got := cfg.ChanDepth(64); got != 32 {
		t.Errorf("set ChanDepth(64) = %d, want 32", got)
	}
}

//fusa:test REQ-RELAY-017
//fusa:test REQ-RELAY-018
func TestApplyMultipleOpts(t *testing.T) {
	cfg := ApplySubscriberOpts([]SubscriberOption{
		WithChannelDepth(10),
		WithBackPressure(DropOldest),
		WithChannelDepth(20),
	})
	if cfg.ChannelDepth != 20 {
		t.Errorf("last-write-wins ChannelDepth = %d, want 20", cfg.ChannelDepth)
	}
	if cfg.BackPressure != DropOldest {
		t.Errorf("BackPressure = %d, want DropOldest", cfg.BackPressure)
	}
}

//fusa:test REQ-RELAY-051
func TestWithEventID(t *testing.T) {
	cfg := ApplySubscriberOpts([]SubscriberOption{WithEventID(42)})
	if cfg.EventID != 42 {
		t.Errorf("EventID = %d, want 42", cfg.EventID)
	}
	// Default EventID is zero.
	dflt := ApplySubscriberOpts(nil)
	if dflt.EventID != 0 {
		t.Errorf("default EventID = %d, want 0", dflt.EventID)
	}
}

//fusa:test REQ-RELAY-056
func TestWithTopic(t *testing.T) {
	cfg := ApplySubscriberOpts([]SubscriberOption{WithTopic("sensors/temperature")})
	if cfg.TopicName != "sensors/temperature" {
		t.Errorf("TopicName = %q, want %q", cfg.TopicName, "sensors/temperature")
	}
	// Default TopicName is empty.
	dflt := ApplySubscriberOpts(nil)
	if dflt.TopicName != "" {
		t.Errorf("default TopicName = %q, want empty", dflt.TopicName)
	}
}

//fusa:test REQ-RELAY-013
//fusa:test REQ-RELAY-014
func TestCallerEmbeddsNode(t *testing.T) {
	var c Caller = callerStub{}
	var n Node = c
	if n.Protocol() != CAN {
		t.Error("Caller must satisfy Node")
	}
}
