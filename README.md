# RELAY

**Real-time Embedded Link Abstraction Yoke**

RELAY is the shared specification and library for the SoundMatt embedded network
protocol ecosystem. CAN, DDS, LIN, MQTT, RCP, and SOME/IP implementations in Go,
Rust, and C++ build against RELAY to share canonical types, interface contracts,
error semantics, and a common application API.

[![CI](https://github.com/SoundMatt/RELAY/actions/workflows/ci.yml/badge.svg)](https://github.com/SoundMatt/RELAY/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/SoundMatt/RELAY.svg)](https://pkg.go.dev/github.com/SoundMatt/RELAY)

---

## Protocol coverage

| Protocol | Go | C++ | Rust |
|---|---|---|---|
| CAN | [go-CAN](https://github.com/SoundMatt/go-CAN) | — | — |
| DDS | [go-DDS](https://github.com/SoundMatt/go-DDS) | — | — |
| LIN | [go-LIN](https://github.com/SoundMatt/go-LIN) | — | — |
| MQTT | [go-MQTT](https://github.com/SoundMatt/go-mqtt) | — | — |
| RCP | [go-RCP](https://github.com/SoundMatt/go-RCP) | [cpp-RCP](https://github.com/SoundMatt/cpp-RCP) | — |
| SOME/IP | [go-SOMEIP](https://github.com/SoundMatt/go-SOMEIP) | — | — |

## Specification

Full specification: [`spec/relay-spec.md`](spec/relay-spec.md)  
Machine-readable version: [`spec/version.json`](spec/version.json)  
Change history: [`spec/CHANGELOG.md`](spec/CHANGELOG.md)

Current: **v0.1 (draft)**

## Install

```
go get github.com/SoundMatt/RELAY@latest
```

## Usage

```go
import relay "github.com/SoundMatt/RELAY"

// All protocol adapters satisfy relay.Node
var node relay.Node = can.Adapt(bus)

// Send — identical regardless of underlying protocol
err := node.Send(ctx, relay.Message{
    Protocol: relay.CAN,
    ID:       "256",      // CAN frame 0x100; DDS topic; MQTT topic; RCP zone…
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
        ID:       "FrontLeft",
        Payload:  data,
    })
}
```

## CLI

```
relay version [--format text|json]
```

## Conformance

```
relay conform <binary>
```

(available from v0.5)

## Roadmap

See [`ROADMAP.md`](ROADMAP.md).

## Contributing

Sign-off required on every commit (DCO):

```
git commit -s -m "feat: description"
```

## License

Mozilla Public License 2.0 — see [LICENSE](LICENSE).
