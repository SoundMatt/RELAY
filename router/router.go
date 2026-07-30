// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

// Package router implements the RELAY crossbar: a central switch fabric that
// routes relay.Message between named protocol spokes. Each spoke is any
// relay.Node (an in-process Adapt()ed implementation, or a CLI-backed node).
// A route forwards messages from one source spoke to one or more destination
// spokes, optionally filtered and converted (the gateway/translation step).
// "Repeat" is a same-protocol route (identity converter); "bridge" is a
// cross-protocol route (a converter rewrites the message).
//
// The engine depends only on the relay.Node interface and relay.Message, so it
// is zero-dependency and works identically for in-process and CLI-backed spokes.
package router

import (
	"context"
	"fmt"
	"sort"
	"sync"

	relay "github.com/SoundMatt/RELAY"
)

// Converter rewrites a message as it crosses a route. A nil converter on a
// Route means identity (a same-protocol repeat).
//
//fusa:req REQ-RELAY-085
type Converter func(relay.Message) (relay.Message, error)

// Filter reports whether a message is eligible for a route. A nil filter on a
// Route matches every message.
//
//fusa:req REQ-RELAY-084
type Filter func(relay.Message) bool

// Route forwards messages from the source spoke From to each destination spoke
// in To, applying Filter (if set) then Convert (if set).
//
//fusa:req REQ-RELAY-084
type Route struct {
	From    string
	To      []string
	Filter  Filter
	Convert Converter
}

// Stats is a snapshot of router activity.
//
//fusa:req REQ-RELAY-084
type Stats struct {
	Forwarded uint64 `json:"forwarded"`
	Filtered  uint64 `json:"filtered"`
	Errors    uint64 `json:"errors"`
}

// Router is the crossbar switch fabric.
//
//fusa:req REQ-RELAY-084
type Router struct {
	mu     sync.Mutex
	spokes map[string]relay.Node
	routes []Route
	stats  Stats
}

// New returns an empty Router.
//
//fusa:req REQ-RELAY-084
func New() *Router {
	return &Router{spokes: make(map[string]relay.Node)}
}

// AddSpoke registers a named spoke. It errors on a duplicate name or nil node.
//
//fusa:req REQ-RELAY-084
func (r *Router) AddSpoke(name string, node relay.Node) error {
	if name == "" {
		return fmt.Errorf("router: spoke name must not be empty")
	}
	if node == nil {
		return fmt.Errorf("router: spoke %q node must not be nil", name)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, dup := r.spokes[name]; dup {
		return fmt.Errorf("router: duplicate spoke %q", name)
	}
	r.spokes[name] = node
	return nil
}

// AddRoute registers a route. It errors if the source or any destination spoke
// is unknown, or if To is empty.
//
//fusa:req REQ-RELAY-084
func (r *Router) AddRoute(rt Route) error {
	if len(rt.To) == 0 {
		return fmt.Errorf("router: route from %q has no destinations", rt.From)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.spokes[rt.From]; !ok {
		return fmt.Errorf("router: route source %q is not a registered spoke", rt.From)
	}
	for _, dst := range rt.To {
		if _, ok := r.spokes[dst]; !ok {
			return fmt.Errorf("router: route destination %q is not a registered spoke", dst)
		}
	}
	r.routes = append(r.routes, rt)
	return nil
}

// Stats returns a snapshot of forwarding counters.
//
//fusa:req REQ-RELAY-084
func (r *Router) Stats() Stats {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.stats
}

// Run subscribes to every source spoke and forwards messages along the routes
// until ctx is cancelled. Each source is drained in its own goroutine. Run
// returns ctx.Err() (typically context.Canceled) once all sources stop.
//
//fusa:req REQ-RELAY-084
func (r *Router) Run(ctx context.Context) error {
	sources := r.sourceSpokes()
	if len(sources) == 0 {
		return fmt.Errorf("router: no routes configured")
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var wg sync.WaitGroup
	for _, src := range sources {
		r.mu.Lock()
		node := r.spokes[src]
		r.mu.Unlock()
		ch, err := node.Subscribe()
		if err != nil {
			cancel()
			wg.Wait()
			return fmt.Errorf("router: subscribe to %q: %w", src, err)
		}
		wg.Add(1)
		go func(src string, ch <-chan relay.Message) {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case msg, ok := <-ch:
					if !ok {
						return
					}
					r.dispatch(ctx, src, msg)
				}
			}
		}(src, ch)
	}

	<-ctx.Done()
	wg.Wait()
	return ctx.Err()
}

// dispatch forwards one message from src along every matching route. It
// snapshots the routes and destination spokes it needs under r.mu, then does
// the (potentially blocking, I/O-bound) Send calls without holding the lock —
// so a concurrent AddSpoke/AddRoute doesn't race the map/slice reads, but
// dispatch also doesn't serialize on the mutex for the actual send.
func (r *Router) dispatch(ctx context.Context, src string, msg relay.Message) {
	r.mu.Lock()
	matching := make([]Route, 0, len(r.routes))
	for _, rt := range r.routes {
		if rt.From == src {
			matching = append(matching, rt)
		}
	}
	dstNode := make(map[string]relay.Node, len(r.spokes))
	for name, n := range r.spokes {
		dstNode[name] = n
	}
	r.mu.Unlock()

	for _, rt := range matching {
		if rt.Filter != nil && !rt.Filter(msg) {
			r.bump(&r.stats.Filtered)
			continue
		}
		out := msg
		if rt.Convert != nil {
			converted, err := rt.Convert(msg)
			if err != nil {
				r.bump(&r.stats.Errors)
				continue
			}
			out = converted
		}
		for _, dst := range rt.To {
			node, ok := dstNode[dst]
			if !ok {
				r.bump(&r.stats.Errors)
				continue
			}
			if err := node.Send(ctx, out); err != nil {
				r.bump(&r.stats.Errors)
			} else {
				r.bump(&r.stats.Forwarded)
			}
		}
	}
}

// sourceSpokes returns the sorted, de-duplicated set of spokes that are a
// source in at least one route.
func (r *Router) sourceSpokes() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	seen := make(map[string]bool)
	for _, rt := range r.routes {
		seen[rt.From] = true
	}
	out := make([]string, 0, len(seen))
	for s := range seen {
		out = append(out, s)
	}
	sort.Strings(out)
	return out
}

func (r *Router) bump(counter *uint64) {
	r.mu.Lock()
	*counter++
	r.mu.Unlock()
}
