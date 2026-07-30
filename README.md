# RELAY

**Real-time Embedded Link Abstraction Yoke**

RELAY is the shared specification and library for the SoundMatt embedded network
protocol ecosystem. CAN, DDS, LIN, MQTT, RCP, and SOME/IP implementations in Go,
C, C++, and Rust build against RELAY to share canonical types, interface
contracts, error semantics, and a common application API.

[![CI](https://github.com/SoundMatt/RELAY/actions/workflows/ci.yml/badge.svg)](https://github.com/SoundMatt/RELAY/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/SoundMatt/RELAY.svg)](https://pkg.go.dev/github.com/SoundMatt/RELAY)

---

## Protocol coverage

| Protocol | Go | C | C++ | Rust |
|---|---|---|---|---|
| CAN | [go-CAN](https://github.com/SoundMatt/go-CAN) | — | [cpp-CAN](https://github.com/SoundMatt/cpp-CAN) | [rust-CAN](https://github.com/SoundMatt/rust-CAN) |
| DDS | [go-DDS](https://github.com/SoundMatt/go-DDS) | — | [cpp-DDS](https://github.com/SoundMatt/cpp-DDS) | [rust-DDS](https://github.com/SoundMatt/rust-DDS) |
| LIN | [go-LIN](https://github.com/SoundMatt/go-LIN) | — | [cpp-LIN](https://github.com/SoundMatt/cpp-LIN) | [rust-LIN](https://github.com/SoundMatt/rust-LIN) |
| MQTT | [go-MQTT](https://github.com/SoundMatt/go-mqtt) | — | [cpp-MQTT](https://github.com/SoundMatt/cpp-MQTT) | [rust-MQTT](https://github.com/SoundMatt/rust-MQTT) |
| RCP | [go-RCP](https://github.com/SoundMatt/go-RCP) | [c-RCP](https://github.com/SoundMatt/c-RCP) | [cpp-RCP](https://github.com/SoundMatt/cpp-RCP) | [rust-RCP](https://github.com/SoundMatt/rust-RCP) |
| SOME/IP | [go-SOMEIP](https://github.com/SoundMatt/go-SOMEIP) | — | — | — |

## Specification

Full specification: [`spec/relay-spec.md`](spec/relay-spec.md)  
Machine-readable version: [`spec/version.json`](spec/version.json)  
Change history: [`spec/CHANGELOG.md`](spec/CHANGELOG.md)

Current: **v2.0 (stable)**

## Install

```
go get github.com/SoundMatt/RELAY@latest
```

Install the CLI:

```
go install github.com/SoundMatt/RELAY/cmd/relay@latest
```

## Usage

```go
import relay "github.com/SoundMatt/RELAY"

// All protocol adapters satisfy relay.Node
var node relay.Node = can.Adapt(bus)

// Send — identical regardless of underlying protocol
err := node.Send(ctx, relay.Message{
    Protocol: relay.CAN,
    ID:       "256",      // CAN frame 0x100; DDS topic; MQTT topic; RCP ByteBusID…
    Payload:  data,
})

// Subscribe
ch, err := node.Subscribe(
    relay.WithChannelDepth(128),
    relay.WithBackPressure(relay.DropOldest),
)
for msg := range ch {
    fmt.Printf("%s %s %x\n", msg.Protocol, msg.ID, msg.Payload)
}

// Request/response (RCP, SOME/IP)
if caller, ok := node.(relay.Caller); ok {
    resp, err := caller.Call(ctx, relay.Message{
        Protocol: relay.RCP,
        ID:       "9", // decimal ByteBusID
        Payload:  data,
    })
}
```

## CLI

```
Usage: relay <command> [flags]
```

| Command | Description |
|---|---|
| `version [--format text\|json]` | Print tool and spec version |
| `capabilities` | Print RELAY tooling capabilities document |
| `status` | Print RELAY tooling status document |
| `conform <bin>` | Verify that `<bin>` conforms to the RELAY spec |
| `convert` | Reference canonical-value → `relay.Message` conversion (stdin→stdout) |
| `interop <bin>...` | Check implementations are behaviourally interchangeable |
| `crossbar` | Route `relay.Message`s between protocol spokes (`--config`) |
| `probe` | Discover RELAY-conformant binaries |
| `trace` | Capture or replay a `relay.Message` stream |
| `report` | Cross-implementation conformance report |
| `sbom` | Print the software bill of materials |
| `safety-case` | Summarise the safety evidence set |
| `audit-pack` | Bundle all safety evidence into a zip |
| `compare` | Compare two implementations for interchangeability |
| `versions` | List implementations and their spec alignment |
| `serve` | Serve a web dashboard, JSON API, and status badge |

## Conformance

```
relay conform <binary>
```

## Roadmap

See [`ROADMAP.md`](ROADMAP.md).

## Contributing

Sign-off required on every commit (DCO):

```
git commit -s -m "feat: description"
```

## License

Mozilla Public License 2.0 — see [LICENSE](LICENSE).
