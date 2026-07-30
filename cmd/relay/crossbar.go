// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"sync"
	"syscall"
	"time"

	relay "github.com/SoundMatt/RELAY/v2"
	"github.com/SoundMatt/RELAY/v2/router"
)

// cliNode is a relay.Node backed by an x-Net binary's CLI: it sources messages
// from `<binary> subscribe --format json` and sinks them to
// `<binary> send --format json` (reading relay.Message NDJSON on stdin). This
// keeps the crossbar zero-dependency and cross-language — it conducts each
// implementation's own I/O rather than linking it.
//
//fusa:req REQ-RELAY-086
type cliNode struct {
	binary   string
	proto    relay.Protocol
	subArgs  []string
	sendArgs []string

	mu        sync.Mutex
	subCmd    *exec.Cmd
	closed    bool
	done      chan struct{}
	sendCmd   *exec.Cmd
	sendStdin io.WriteCloser
}

func (n *cliNode) Protocol() relay.Protocol { return n.proto }

// Subscribe spawns `<binary> subscribe --format json` and streams the decoded
// relay.Message NDJSON on the returned channel until the node is closed. It
// honors the caller's SubscriberOptions (channel depth and back-pressure
// policy per §10.5) instead of hardcoding an always-blocking send — a
// Block-policy subscriber that stops draining the channel no longer leaks the
// forwarding goroutine forever, since it also selects on Close().
func (n *cliNode) Subscribe(opts ...relay.SubscriberOption) (<-chan relay.Message, error) {
	cfg := relay.ApplySubscriberOpts(opts)
	depth := cfg.ChanDepth(64)

	args := append([]string{"subscribe", "--format", "json"}, n.subArgs...)
	cmd := exec.Command(n.binary, args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	n.mu.Lock()
	n.subCmd = cmd
	if n.done == nil {
		n.done = make(chan struct{})
	}
	done := n.done
	n.mu.Unlock()

	ch := make(chan relay.Message, depth)
	go func() {
		defer close(ch)
		sc := bufio.NewScanner(stdout)
		sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
		for sc.Scan() {
			var m relay.Message
			if json.Unmarshal(sc.Bytes(), &m) != nil {
				continue
			}
			switch cfg.BackPressure {
			case relay.DropNewest:
				select {
				case ch <- m:
				default:
					// channel full: drop the arriving sample
				}
			case relay.DropOldest:
				select {
				case ch <- m:
				default:
					select {
					case <-ch: // evict the oldest buffered sample
					default:
					}
					select {
					case ch <- m:
					default:
					}
				}
			default: // relay.Block
				select {
				case ch <- m:
				case <-done:
					return
				}
			}
		}
		_ = cmd.Wait()
	}()
	return ch, nil
}

// Send writes msg as one NDJSON line to a single persistent
// `<binary> send --format json` process's stdin — spec §11.2's "streaming
// JSON sink" is explicitly "the egress dual of subscribe --format json"
// (a persistent process reading a stream until EOF), not one process spawned
// per message. The process is started lazily on first use and kept alive
// for the node's lifetime; a write failure (e.g. the sink process died) tears
// it down so the next Send call restarts it.
func (n *cliNode) Send(ctx context.Context, msg relay.Message) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	stdin, startErr := n.sendPipe()
	if startErr != nil {
		return startErr
	}

	writeErrCh := make(chan error, 1)
	go func() {
		_, err := stdin.Write(append(data, '\n'))
		writeErrCh <- err
	}()
	select {
	case err := <-writeErrCh:
		if err != nil {
			n.teardownSend()
		}
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

// sendPipe returns the stdin pipe of the persistent send process, starting it
// if it isn't already running.
func (n *cliNode) sendPipe() (io.WriteCloser, error) {
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.closed {
		return nil, fmt.Errorf("cliNode: send on closed node")
	}
	if n.sendStdin != nil {
		return n.sendStdin, nil
	}
	args := append([]string{"send", "--format", "json"}, n.sendArgs...)
	cmd := exec.Command(n.binary, args...)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	n.sendCmd = cmd
	n.sendStdin = stdin
	return stdin, nil
}

// teardownSend closes and forgets the persistent send process so the next
// Send call restarts it, e.g. after a write to a dead process's pipe fails.
func (n *cliNode) teardownSend() {
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.sendStdin != nil {
		_ = n.sendStdin.Close()
		n.sendStdin = nil
	}
	if n.sendCmd != nil && n.sendCmd.Process != nil {
		_ = n.sendCmd.Process.Kill()
		_ = n.sendCmd.Wait()
	}
	n.sendCmd = nil
}

// Close terminates the subscribe process.
func (n *cliNode) Close() error {
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.closed {
		return nil
	}
	n.closed = true
	if n.done != nil {
		close(n.done)
	}
	if n.subCmd != nil && n.subCmd.Process != nil {
		_ = n.subCmd.Process.Kill()
	}
	if n.sendStdin != nil {
		_ = n.sendStdin.Close()
		n.sendStdin = nil
	}
	if n.sendCmd != nil && n.sendCmd.Process != nil {
		_ = n.sendCmd.Process.Kill()
		_ = n.sendCmd.Wait()
	}
	n.sendCmd = nil
	return nil
}

// crossbarConfig is the JSON configuration for `relay crossbar`.
type crossbarConfig struct {
	Spokes []struct {
		Name        string   `json:"name"`
		Binary      string   `json:"binary"`
		Protocol    string   `json:"protocol"`
		SubscribeAr []string `json:"subscribe_args,omitempty"`
		SendArgs    []string `json:"send_args,omitempty"`
	} `json:"spokes"`
	Routes []struct {
		From      string   `json:"from"`
		To        []string `json:"to"`
		Converter string   `json:"converter,omitempty"`
	} `json:"routes"`
}

// runCrossbar implements `relay crossbar --config FILE [--duration D]`.
//
//fusa:req REQ-RELAY-086
func runCrossbar(stdout, stderr io.Writer, args []string) error {
	fs := flag.NewFlagSet("crossbar", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	configPath := fs.String("config", "", "Path to the crossbar JSON config")
	duration := fs.Duration("duration", 0, "Run for this long then stop (0 = until interrupted)")
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("relay crossbar: %w", err)
	}
	if *configPath == "" {
		fmt.Fprintln(stderr, "relay crossbar: --config is required")
		return exitCode(2)
	}

	cfg, err := loadCrossbarConfig(*configPath)
	if err != nil {
		fmt.Fprintf(stderr, "relay crossbar: %v\n", err)
		return exitCode(2)
	}

	r := router.New()
	protoOf := map[string]relay.Protocol{}
	nodes := map[string]*cliNode{}
	for _, s := range cfg.Spokes {
		p, ok := relay.ParseProtocol(s.Protocol)
		if !ok {
			fmt.Fprintf(stderr, "relay crossbar: spoke %q: unknown protocol %q\n", s.Name, s.Protocol)
			return exitCode(2)
		}
		node := &cliNode{binary: s.Binary, proto: p, subArgs: s.SubscribeAr, sendArgs: s.SendArgs}
		if err := r.AddSpoke(s.Name, node); err != nil {
			fmt.Fprintf(stderr, "relay crossbar: %v\n", err)
			return exitCode(2)
		}
		protoOf[s.Name] = p
		nodes[s.Name] = node
	}
	for _, rt := range cfg.Routes {
		if rt.Converter != "" {
			// Explicit converter: the user asked for one shared conversion
			// across the whole fan-out — respect that as a single route.
			conv, err := router.Lookup(rt.Converter)
			if err != nil {
				fmt.Fprintf(stderr, "relay crossbar: %v\n", err)
				return exitCode(2)
			}
			if err := r.AddRoute(router.Route{From: rt.From, To: rt.To, Convert: conv}); err != nil {
				fmt.Fprintf(stderr, "relay crossbar: %v\n", err)
				return exitCode(2)
			}
			continue
		}
		// No explicit converter: router.Route applies exactly one Convert to
		// its entire To fan-out, so a single route picking a converter from
		// only the first destination would silently mis-tag every other
		// destination with a different protocol (e.g. a CAN source fanning
		// out to [DDS, LIN] would re-tag the LIN copy as DDS too). Split into
		// one single-destination route per target instead, each with its own
		// correctly-inferred converter — same fan-out behavior, correct per
		// destination, no change to router.Route's public shape.
		for _, dst := range rt.To {
			route := router.Route{From: rt.From, To: []string{dst}, Filter: nil}
			if protoOf[rt.From] != protoOf[dst] {
				route.Convert = router.DefaultConverter(protoOf[rt.From], protoOf[dst])
			}
			if err := r.AddRoute(route); err != nil {
				fmt.Fprintf(stderr, "relay crossbar: %v\n", err)
				return exitCode(2)
			}
		}
	}

	ctx, cancel := signalContext(*duration)
	defer cancel()

	fmt.Fprintf(stdout, "relay crossbar: %d spoke(s), %d route(s) — running\n", len(cfg.Spokes), len(cfg.Routes))
	runErr := r.Run(ctx)
	st := r.Stats()
	fmt.Fprintf(stdout, "relay crossbar: stopped — forwarded=%d filtered=%d errors=%d\n", st.Forwarded, st.Filtered, st.Errors)
	for _, n := range nodes {
		_ = n.Close() // best-effort
	}
	if runErr != nil && runErr != context.Canceled && runErr != context.DeadlineExceeded {
		return fmt.Errorf("relay crossbar: %w", runErr)
	}
	return nil
}

// loadCrossbarConfig reads and validates the JSON config.
func loadCrossbarConfig(path string) (crossbarConfig, error) {
	var cfg crossbarConfig
	b, err := os.ReadFile(path)
	if err != nil {
		return cfg, fmt.Errorf("read config: %w", err)
	}
	if err := json.Unmarshal(b, &cfg); err != nil {
		return cfg, fmt.Errorf("parse config: %w", err)
	}
	if len(cfg.Spokes) == 0 || len(cfg.Routes) == 0 {
		return cfg, fmt.Errorf("config must define at least one spoke and one route")
	}
	return cfg, nil
}

// signalContext returns a context cancelled on SIGINT/SIGTERM, or after d if d>0.
func signalContext(d time.Duration) (context.Context, context.CancelFunc) {
	if d > 0 {
		return context.WithTimeout(context.Background(), d)
	}
	ctx, cancel := context.WithCancel(context.Background())
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sig
		cancel()
	}()
	return ctx, cancel
}
