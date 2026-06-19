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
	"strings"
	"sync"
	"syscall"
	"time"

	relay "github.com/SoundMatt/RELAY"
	"github.com/SoundMatt/RELAY/router"
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

	mu     sync.Mutex
	subCmd *exec.Cmd
	closed bool
}

func (n *cliNode) Protocol() relay.Protocol { return n.proto }

// Subscribe spawns `<binary> subscribe --format json` and streams the decoded
// relay.Message NDJSON on the returned channel until the node is closed.
func (n *cliNode) Subscribe(_ ...relay.SubscriberOption) (<-chan relay.Message, error) {
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
	n.mu.Unlock()

	ch := make(chan relay.Message, 64)
	go func() {
		defer close(ch)
		sc := bufio.NewScanner(stdout)
		sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
		for sc.Scan() {
			var m relay.Message
			if json.Unmarshal(sc.Bytes(), &m) == nil {
				ch <- m
			}
		}
		_ = cmd.Wait()
	}()
	return ch, nil
}

// Send writes msg as one NDJSON line to `<binary> send --format json`.
func (n *cliNode) Send(ctx context.Context, msg relay.Message) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	args := append([]string{"send", "--format", "json"}, n.sendArgs...)
	cmd := exec.CommandContext(ctx, n.binary, args...)
	cmd.Stdin = strings.NewReader(string(data) + "\n")
	return cmd.Run()
}

// Close terminates the subscribe process.
func (n *cliNode) Close() error {
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.closed {
		return nil
	}
	n.closed = true
	if n.subCmd != nil && n.subCmd.Process != nil {
		_ = n.subCmd.Process.Kill()
	}
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
		route := router.Route{From: rt.From, To: rt.To}
		if rt.Converter != "" {
			conv, err := router.Lookup(rt.Converter)
			if err != nil {
				fmt.Fprintf(stderr, "relay crossbar: %v\n", err)
				return exitCode(2)
			}
			route.Convert = conv
		} else if len(rt.To) > 0 {
			route.Convert = router.DefaultConverter(protoOf[rt.From], protoOf[rt.To[0]])
		}
		if err := r.AddRoute(route); err != nil {
			fmt.Fprintf(stderr, "relay crossbar: %v\n", err)
			return exitCode(2)
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
