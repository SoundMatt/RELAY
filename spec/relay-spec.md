# RELAY Specification — v0.1 (draft)

**Real-time Embedded Link Abstraction Yoke**

Networks: CAN · DDS · LIN · MQTT · RCP · SOME/IP  
Languages: Go · Rust · C++

Machine-readable version: [`spec/version.json`](version.json)  
Change history: [`spec/CHANGELOG.md`](CHANGELOG.md)

---

## 1. Scope

RELAY defines the complete shared contract that all protocol implementations in the
SoundMatt ecosystem build against, and the application-level interface that allows
code written against one protocol to be swapped for another with a single line change.

It covers:

1. A **universal message envelope** (`Message`) for cross-protocol tooling, tracing, and observability.
2. **Canonical frame types** per protocol — the authoritative type definitions.
3. **Constructor contract** — how every implementation is constructed and discovered.
4. **Interface contracts** — the exact method signatures each implementation MUST expose.
5. **Optional interfaces** — capability extensions (loaning, health, metrics, drain) available to all protocols.
6. **Application interface** — `relay.Node` and `relay.Caller`, the protocol-agnostic interfaces applications program against, with per-protocol `Adapt()` adapters.
7. **Common error sentinels** and their wrapping semantics.
8. **Lifecycle guarantees** including concurrency and subscription semantics.
9. **Subscriber defaults** and the standard options helpers.
10. **CLI contract** — the command surface `relay conform` uses to verify any binary.
11. **Capability discovery** — how implementations declare what they support.
12. **Implementation naming** — binary, package, and module names.
13. **Per-protocol defaults** — buffer sizes, address limits, timing constraints.
14. **Language bindings** — Go canonical types, C++ abstract classes, Rust async traits.

RELAY does **not** define:

- Wire formats. Each protocol retains its own native wire encoding.
- Transport selection. Implementations choose their transports.
- Application-level data models or schemas.

---

## 2. Terminology

| Term | Meaning |
|---|---|
| Implementation | A concrete library or binary speaking one protocol (e.g. go-CAN, cpp-RCP) |
| RELAY-conformant | Satisfies all MUST requirements in this spec for its declared protocol |
| Canonical type | The RELAY-defined struct/class for a protocol frame (e.g. `CANFrame`) |
| Envelope | `relay.Message` — the universal cross-protocol wrapper |
| Transport | A concrete network backend (socketcan, UDP, virtual, mock) |
| Protocol | One of: CAN, DDS, LIN, MQTT, RCP, SOMEIP |
| Node | A protocol endpoint wrapped at the `relay.Message` level |
| MUST / MUST NOT | RFC 2119 mandatory |
| SHOULD / SHOULD NOT | RFC 2119 recommended |
| MAY | RFC 2119 optional |

---

## 3. Protocol Identifiers

Each protocol is identified by a typed integer constant. Zero is reserved.

**Go (canonical):**
```go
type Protocol int

const (
    CAN    Protocol = 1 // Controller Area Network (classic and FD)
    DDS    Protocol = 2 // Data Distribution Service
    LIN    Protocol = 3 // Local Interconnect Network
    MQTT   Protocol = 4 // Message Queuing Telemetry Transport
    RCP    Protocol = 5 // Remote Control Protocol
    SOMEIP Protocol = 6 // Scalable service-Oriented MiddlewarE over IP
)

func (p Protocol) String() string {
    switch p {
    case CAN:    return "CAN"
    case DDS:    return "DDS"
    case LIN:    return "LIN"
    case MQTT:   return "MQTT"
    case RCP:    return "RCP"
    case SOMEIP: return "SOMEIP"
    default:     return "unknown"
    }
}
```

**C++:** `enum class Protocol : int { CAN=1, DDS=2, LIN=3, MQTT=4, RCP=5, SOMEIP=6 };`

**Rust:** `#[repr(i32)] pub enum Protocol { CAN=1, DDS=2, LIN=3, MQTT=4, RCP=5, SOMEIP=6 }`

In JSON, `Protocol` is serialised as its integer value.

### 3.1 Adding a new protocol

New protocols are added by opening a PR proposing the new constant and canonical
name, adding the canonical frame type (§14), interface contract (§8), CLI
contract (§10), and `Adapt()` implementation (§9). Values are assigned
sequentially and never reused. The spec MINOR version is bumped.

---

## 4. Universal Message Envelope

`relay.Message` is used by tooling and by `relay.Node` / `relay.Caller` (§9) to
represent any message from any protocol. It is **not** a wire format.

### 4.1 Go definition

```go
type Message struct {
    Protocol  Protocol          `json:"protocol"`  // integer per §3
    Version   Version           `json:"version"`
    ID        string            `json:"id"`        // §4.2
    Payload   []byte            `json:"payload"`   // base64 in JSON
    Timestamp time.Time         `json:"timestamp"` // RFC 3339 nanosecond
    Seq       uint64            `json:"seq,omitempty"`
    Meta      map[string]string `json:"meta,omitempty"` // §4.3
}

type Version struct {
    Major int `json:"major"`
    Minor int `json:"minor"`
    Patch int `json:"patch"`
}

func (v Version) String() string {
    return fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
}
```

### 4.2 ID field

The canonical routing key serialised as a string:

| Protocol | ID value | Example |
|---|---|---|
| CAN | Decimal frame ID | `"256"` |
| DDS | Topic name | `"vehicle/speed"` |
| LIN | Decimal frame ID (0–63) | `"42"` |
| MQTT | Topic string | `"sensors/temp"` |
| RCP | Zone name | `"FrontLeft"` |
| SOMEIP | `"serviceID/methodID"` decimal | `"4660/22136"` |

### 4.3 Meta field

| Protocol | Key | Values |
|---|---|---|
| CAN | `can.ext` | `true` \| `false` |
| CAN | `can.fd` | `true` \| `false` |
| CAN | `can.rtr` | `true` \| `false` |
| CAN | `can.brs` | `true` \| `false` |
| DDS | `dds.writer_guid` | 32-char hex |
| DDS | `dds.reliability` | `best_effort` \| `reliable` |
| DDS | `dds.durability` | `volatile` \| `transient_local` |
| LIN | `lin.checksum_type` | `classic` \| `enhanced` |
| LIN | `lin.checksum` | Decimal uint8 |
| MQTT | `mqtt.qos` | `0` \| `1` \| `2` |
| MQTT | `mqtt.retained` | `true` \| `false` |
| RCP | `rcp.priority` | `normal` \| `high` \| `critical` |
| RCP | `rcp.cmd_type` | `noop` \| `set` \| `get` \| `reset` \| `watchdog` \| `sleep` \| `wake` |
| SOMEIP | `someip.msg_type` | `request` \| `request_no_return` \| `notification` \| `response` \| `error` |
| SOMEIP | `someip.return_code` | Decimal uint8 |
| SOMEIP | `someip.interface_version` | Decimal uint8 |

---

## 5. Common Error Sentinels

### 5.1 Mandatory sentinels

Every RELAY-conformant implementation MUST define these four errors:

| Error | MUST be returned when |
|---|---|
| `ErrClosed` | Operation on a closed connection or subscription |
| `ErrNotConnected` | Operation before connection is established |
| `ErrTimeout` | Operation did not complete within the permitted time |
| `ErrPayloadTooLarge` | Payload exceeds the protocol maximum |

**Go:**
```go
var (
    ErrClosed          = errors.New("relay: closed")
    ErrNotConnected    = errors.New("relay: not connected")
    ErrTimeout         = errors.New("relay: timeout")
    ErrPayloadTooLarge = errors.New("relay: payload too large")
)
```

**C++:** `enum class Errc { closed, not_connected, timeout, payload_too_large };`
registered as `std::error_category` named `"relay"`.

**Rust:**
```rust
#[derive(Debug, thiserror::Error)]
pub enum Error {
    #[error("relay: closed")]            Closed,
    #[error("relay: not connected")]     NotConnected,
    #[error("relay: timeout")]           Timeout,
    #[error("relay: payload too large")] PayloadTooLarge,
}
```

### 5.2 Error wrapping semantics

In Go, protocol-specific errors representing a common condition MUST be wrapped
so `errors.Is` returns true for the appropriate sentinel:

```go
return fmt.Errorf("rcp: zone gone: %w", relay.ErrClosed)  // correct
```

In C++, protocol-specific codes MUST map to the canonical `relay::Errc` via
`std::error_condition` equivalence.

In Rust, protocol-specific variants MUST implement `From<relay::Error>` or
expose `.kind() -> relay::Error`.

### 5.3 Protocol-specific errors

Implementations MAY define additional errors. These are not checked by
`relay conform` but are enumerated here for consistency across the ecosystem.

| Protocol | Error | Condition |
|---|---|---|
| CAN | `ErrInvalidFrame` | Frame fails `ValidateFrame` |
| DDS | `ErrTopicEmpty` | Topic string is empty |
| DDS | `ErrQoSMismatch` | Publisher and subscriber QoS incompatible |
| DDS | `ErrDeadlineMissed` | Sample not delivered before deadline |
| DDS | `ErrSampleRejected` | Sample rejected due to resource limits |
| DDS | `ErrResourceLimits` | Resource limit exceeded |
| DDS | `ErrLoanBuffer` | Loaned buffer cannot be acquired |
| LIN | `ErrNoResponse` | No slave responded within schedule window |
| MQTT | `ErrTopicEmpty` | Topic string is empty |
| MQTT | `ErrQoSUnsupported` | QoS level not supported by broker |
| RCP | `ErrNotFound` | Zone not in registry |
| RCP | `ErrAlreadyExists` | Zone already registered |
| RCP | `ErrBusy` | Zone controller busy |
| RCP | `ErrZoneMismatch` | Command zone ≠ controller zone |
| SOMEIP | `ErrUnknownService` | Service ID not registered |
| SOMEIP | `ErrUnknownMethod` | Method ID not registered |
| SOMEIP | `ErrMalformedMessage` | Header or payload malformed |

---

## 6. Lifecycle Requirements

Every RELAY-conformant implementation MUST satisfy all of the following.

1. **Idempotent close.** `Close()` MUST be safe to call multiple times; subsequent calls MUST be no-ops and MUST NOT return an error.
2. **Send after close.** Any send/publish/call after `Close()` MUST return `ErrClosed`.
3. **Receive after close.** Subscribe calls after `Close()` MUST return `ErrClosed`. Channels already returned MUST be closed by the implementation.
4. **Unsubscribe semantics.** After `Unsubscribe()` or `Subscription.Close()`: the channel MUST be closed; further sends to that subscription MUST be silently dropped; calling `Unsubscribe()` again MUST be a no-op.
5. **Context cancellation.** Blocking operations MUST return within a reasonable scheduling interval after context cancellation. Deadline expiry MUST return `ErrTimeout`.
6. **Concurrent close.** `Close()` MUST be safe to call concurrently with in-flight operations; in-flight operations MUST unblock and return `ErrClosed`.
7. **Concurrent sends.** `Send` / `Publish` / `Call` / `Write` MUST be safe to call from multiple goroutines or threads concurrently without external synchronisation.
8. **Multiple subscriptions.** `Subscribe()` MAY be called multiple times; each call MUST return an independent channel or `Subscription`. Closing one MUST NOT affect others.
9. **Zero-value safety.** A zero-value or uninitialised node MUST NOT panic; it MUST return `ErrNotConnected` for any operation.

---

## 7. Constructor Contract

Every RELAY-conformant transport sub-package MUST export a `New` function
returning the protocol's primary interface. Three forms are permitted:

**Form 1 — endpoint-addressed** (preferred for hardware transports):
```go
func New(ctx context.Context, endpoint string, opts ...Option) (Interface, error)
```

**Form 2 — no endpoint** (in-process, virtual, mock):
```go
func New(opts ...Option) Interface
// or: func New(opts ...Option) (Interface, error)
```

**Form 3 — config-struct** (many parameters):
```go
func New(cfg Config) (Interface, error)
```

Rules:

1. `New` MUST return an implementation of the protocol's mandatory interface (§8).
2. A failed `New` MUST return a nil interface and non-nil error; it MUST NOT return a non-nil interface in a broken state.
3. `New` MUST NOT block indefinitely; connection establishment MUST use Form 1 with `ctx`.
4. Every implementation MUST ship a `mock` sub-package with a Form 2 `New` returning a fully functional in-process implementation. The mock MUST implement all mandatory and optional interfaces the primary implementation supports.

---

## 8. Protocol Interface Contracts

These are the exact method signatures every RELAY-conformant implementation MUST
expose. Go definitions are canonical; C++ and Rust equivalents are in §18.

### 8.1 CAN

```go
type Bus interface {
    Send(ctx context.Context, f Frame) error
    Subscribe(filters ...Filter) (<-chan Frame, error)
    Close() error
}

// Optional zero-copy extension.
type LoaningBus interface {
    Bus
    Loan() (*LoanedFrame, error)
    SendLoaned(ctx context.Context, f *LoanedFrame) error
}

func ValidateFrame(f Frame) error
func MaxDataLen(fd bool) int // 64 if fd, else 8
```

### 8.2 DDS

```go
type Participant interface {
    NewPublisher(topic string, opts ...PublisherOption) (Publisher, error)
    NewSubscriber(topic string, opts ...SubscriberOption) (Subscriber, error)
    Domain() Domain
    Close() error
}

type Publisher interface {
    Write(payload []byte) error
    WriteCtx(ctx context.Context, payload []byte) error
    Close() error
}

type Subscriber interface {
    C() <-chan Sample
    TryRead() (Sample, bool)
    Unsubscribe()
    Close() error
}

// Optional zero-copy extension.
type LoaningPublisher interface {
    Publisher
    Loan(size int) ([]byte, error)
    Commit(buf []byte) error
}

type Domain int // MUST be 0–232 inclusive
```

### 8.3 LIN

```go
type Bus interface {
    Publish(id uint8, data []byte) error
    Subscribe(filters ...Filter) (<-chan Frame, error)
    Close() error
}

type MasterBus interface {
    Bus
    SendHeader(ctx context.Context, id uint8) (Frame, error)
}

func ValidateFrame(f Frame) error
func ProtectID(id uint8) uint8
func VerifyPID(pid uint8) (uint8, error)
func CalcChecksum(pid uint8, data []byte, ct ChecksumType) uint8
```

### 8.4 MQTT

```go
type Client interface {
    Publish(ctx context.Context, topic string, qos QoS, payload []byte) error
    Subscribe(topic string, qos QoS, opts ...SubscriberOption) (Subscription, error)
    Close() error
}

type Subscription interface {
    C() <-chan Message
    Unsubscribe() error
    Close() error
}

// MatchTopic MUST implement MQTT §4.7 wildcard semantics.
// Topics beginning with '$' MUST NOT match wildcard subscriptions.
func MatchTopic(filter, topic string) bool
```

### 8.5 RCP

```go
type Controller interface {
    Zone() Zone
    Send(ctx context.Context, cmd *Command) (*Response, error)
    Subscribe(ctx context.Context) (<-chan *Status, error)
    Close() error
}

type Registry interface {
    Register(ctrl Controller) error
    Deregister(zone Zone) error
    Lookup(zone Zone) (Controller, error)
    Controllers() []Controller
    Close() error
}

type LoaningController interface {
    Controller
    Loan(size int) (*Loan, error)
    SendLoaned(ctx context.Context, cmd *Command) (*Response, error)
}
```

### 8.6 SOME/IP

```go
type Service interface {
    Call(ctx context.Context, methodID MethodID, payload []byte) (Message, error)
    CallNoReturn(ctx context.Context, methodID MethodID, payload []byte) error
    Subscribe(eventID EventID, opts ...SubscriberOption) (Subscription, error)
    Close() error
}

type Server interface {
    RegisterMethod(methodID MethodID, handler MethodHandler) error
    Emit(eventID EventID, payload []byte) error
    Close() error
}

type Subscription interface {
    C() <-chan Message
    Unsubscribe() error
    Close() error
}

type MethodHandler func(ctx context.Context, req Message) ([]byte, error)
```

---

## 9. Optional Interfaces

Available to all protocols. Presence is declared in the capabilities document (§12.2).

```go
// HealthProvider exposes node health. Applicable to all protocols.
type HealthProvider interface {
    Health() Health
}

type HealthStatus int
const (
    HealthOK       HealthStatus = 0
    HealthDegraded HealthStatus = 1
    HealthDown     HealthStatus = 2
)

type Health struct {
    Status  HealthStatus `json:"status"`
    Details string       `json:"details,omitempty"`
}

// MetricsProvider exposes runtime counters. Applicable to all protocols.
type MetricsProvider interface {
    Metrics() Metrics
}

type Metrics struct {
    WriteCount     uint64 `json:"write_count"`
    DeliverCount   uint64 `json:"deliver_count"`
    DropCount      uint64 `json:"drop_count"`
    BytesWritten   uint64 `json:"bytes_written"`
    BytesDelivered uint64 `json:"bytes_delivered"`
    ErrorCount     uint64 `json:"error_count"`
}

// Drainer extends any node with graceful shutdown. Applicable to all protocols.
type Drainer interface {
    CloseWithDrain(ctx context.Context) error
}
```

---

## 10. Application Interface

This section defines the protocol-agnostic API that applications program against.
It lives in the RELAY Go package itself. Applications that code to `relay.Node` or
`relay.Caller` can swap the underlying protocol by changing a single constructor call.

### 10.1 relay.Node — pub/sub protocols

`Node` is the application-level interface for publish/subscribe protocols.
CAN, DDS, LIN, and MQTT adapt to `Node`. SOMEIP also adapts to `Node` for its
event subscription side.

```go
// Node is the protocol-agnostic pub/sub interface.
// Applications program against Node; the underlying protocol is an implementation detail.
type Node interface {
    // Protocol returns the network protocol this node speaks.
    Protocol() Protocol

    // Send transmits msg to the network.
    // msg.ID carries the routing key per §4.2.
    // msg.Meta carries protocol-specific fields per §4.3.
    // Returns ErrClosed, ErrNotConnected, ErrTimeout, or ErrPayloadTooLarge on failure.
    Send(ctx context.Context, msg Message) error

    // Subscribe returns a channel of inbound messages.
    // The channel is closed when the node closes (§6.3).
    Subscribe(opts ...SubscriberOption) (<-chan Message, error)

    // Close closes the node. Idempotent per §6.1.
    Close() error
}
```

### 10.2 relay.Caller — request/response protocols

`Caller` extends `Node` for protocols with request/response semantics.
RCP and SOMEIP adapt to `Caller`.

```go
// Caller extends Node with request/response semantics.
// Applications can probe: if c, ok := node.(relay.Caller); ok { ... }
type Caller interface {
    Node

    // Call sends req and blocks until a response arrives or ctx expires.
    // req.ID carries the routing key; resp.ID carries the responder's identity.
    // Returns ErrTimeout if ctx expires before a response.
    Call(ctx context.Context, req Message) (Message, error)
}
```

### 10.3 Adapt() — per-protocol adapters

Every RELAY-conformant protocol package MUST export an `Adapt()` function in its
root package that wraps the primary protocol interface as the appropriate
application interface. `Adapt()` uses `ToMessage()` / `FromMessage()` (§14) for
all conversions.

| Protocol | Signature | Returns |
|---|---|---|
| CAN | `func Adapt(bus Bus) relay.Node` | `relay.Node` |
| DDS | `func Adapt(p Participant) relay.Node` | `relay.Node` |
| LIN | `func Adapt(bus Bus) relay.Node` | `relay.Node` |
| MQTT | `func Adapt(c Client) relay.Node` | `relay.Node` |
| RCP | `func Adapt(c Controller) relay.Caller` | `relay.Caller` (also satisfies `relay.Node`) |
| SOMEIP | `func Adapt(s Service) relay.Caller` | `relay.Caller` (also satisfies `relay.Node`) |

`Adapt()` MUST NOT block. It wraps the given implementation; it does not connect.

### 10.4 Routing rules

`Adapt()` uses `Message.ID` and `Message.Meta` to route sends to the underlying
protocol. These rules define the mapping:

| Protocol | Send: ID → native | Send: Meta → native | Subscribe: native → Message |
|---|---|---|---|
| CAN | Parse decimal uint32 → `Frame.ID` | `can.ext/fd/rtr/brs` → frame flags | `Frame.ToMessage()` per §15.1 |
| DDS | String → topic name for `Publisher.Write()` | `dds.reliability` etc. ignored (set at Participant level) | `Sample.ToMessage()` per §15.2 |
| LIN | Parse decimal uint8 → frame ID for `Bus.Publish()` | — | `Frame.ToMessage()` per §15.3 |
| MQTT | String → topic for `Client.Publish()` | `mqtt.qos` → QoS level; `mqtt.retained` ignored on send | `Message.ToMessage()` per §15.4 |
| RCP | Zone name → `Zone` enum for `Controller.Send()` | `rcp.priority` → `Priority`; `rcp.cmd_type` → `CommandType` | `Status.ToMessage()` per §15.5 |
| SOMEIP | `"svcID/methodID"` → parse to `ServiceID`/`MethodID` | `someip.msg_type` → selects `Call()` vs `CallNoReturn()` | `Message.ToMessage()` per §15.6 |

### 10.5 What is intentionally not preserved at the Node level

Applications that need these features MUST use the protocol-specific interface directly.

| Feature | Protocol | Lost at Node level |
|---|---|---|
| Typed samples (`TypedPublisher[T]`) | DDS | Node operates on raw `[]byte` |
| QoS per-write override | DDS | QoS is set at `Participant` level |
| Response payload from `Controller.Send()` | RCP | `Node.Send()` discards the response; use `Caller.Call()` |
| Per-publish QoS | MQTT | Reads `mqtt.qos` from Meta; broker-enforced floor applies |
| DDS `TryRead()` / `WaitSet` | DDS | Only the channel interface is exposed |
| LIN `MasterBus.SendHeader()` | LIN | Master scheduling not available via Node |

### 10.6 Example

```go
// Application code — protocol-agnostic:
func publish(ctx context.Context, node relay.Node, payload []byte) error {
    return node.Send(ctx, relay.Message{
        Protocol: node.Protocol(),
        ID:       "291",     // CAN: frame 0x123 / DDS: topic "291" / MQTT: topic "291"
        Payload:  payload,
    })
}

// Caller code — identical for RCP and SOMEIP:
func request(ctx context.Context, c relay.Caller, payload []byte) (relay.Message, error) {
    return c.Call(ctx, relay.Message{
        Protocol: c.Protocol(),
        ID:       "FrontLeft",  // RCP zone / SOMEIP: "svcID/methodID"
        Payload:  payload,
    })
}

// Wire up with CAN:
bus, _ := socketcan.New(ctx, "vcan0")
publish(ctx, can.Adapt(bus), data)

// Wire up with DDS — zero changes to publish():
p, _ := cyclone.New(0)
publish(ctx, dds.Adapt(p), data)
```

---

## 11. CLI Contract

Every RELAY-conformant binary MUST implement the mandatory commands below.
Binary naming follows §13.

### 11.1 Mandatory commands

#### `version [--format text|json]`

Reports tool and spec version. JSON schema: §12.1. Exit: `0` success, `2` invalid args.

#### `capabilities`

Emits the capabilities document (§12.2) as JSON. Always JSON; no `--format` flag.
Exit: `0` success.

#### `status [--format text|json]`

Reports self-assessed health without a live network connection. JSON schema: §12.3.
Exit: `0` healthy, `1` degraded.

### 11.2 Recommended commands

#### `connect <endpoint> [--timeout duration]`

Connects and reports success or failure. Exit: `0` connected, `1` failed.

#### `send [protocol-flags] [--format text|json]`

Sends a single message. Protocol-specific required flags:

| Protocol | Required flags |
|---|---|
| CAN | `--id <uint>` `--data <hex>` `[--fd]` `[--ext]` |
| DDS | `--topic <string>` `--payload <bytes>` |
| LIN | `--id <uint>` `--data <hex>` |
| MQTT | `--topic <string>` `--payload <bytes>` `[--qos 0\|1\|2]` |
| RCP | `--zone <name>` `--type <cmdtype>` `[--payload <hex>]` |
| SOMEIP | `--service <uint>` `--method <uint>` `[--payload <bytes>]` |

Exit: `0` sent, `1` error.

#### `subscribe [protocol-flags] [--format text|json] [--count N]`

Subscribes and prints received messages as `relay.Message` NDJSON on stdout.
`--count N` exits after N messages; omitting runs until SIGINT.

Exit: `0` clean, `1` error.

### 11.3 Exit codes

| Code | Meaning |
|---|---|
| `0` | Success |
| `1` | Protocol or operational error |
| `2` | Invalid arguments |

---

## 12. Capability Discovery

### 12.1 Version document

`<binary> version --format json`:

```json
{
    "tool":         "go-can",
    "protocol":     "CAN",
    "protocol_int": 1,
    "version":      "1.2.3",
    "spec_version": "0.1",
    "language":     "go",
    "runtime":      "go1.25.0"
}
```

`language` MUST be one of: `"go"`, `"cpp"`, `"rust"`.

### 12.2 Capabilities document

`<binary> capabilities`:

```json
{
    "kind":                "capabilities",
    "tool":                "go-can",
    "protocol":            "CAN",
    "protocol_int":        1,
    "version":             "1.2.3",
    "spec_version":        "0.1",
    "commands":            ["version", "capabilities", "status", "connect", "send", "subscribe"],
    "transports":          ["socketcan", "virtual"],
    "features":            ["fd", "isotp", "j1939"],
    "interfaces":          ["Bus"],
    "optional_interfaces": ["LoaningBus", "HealthProvider", "MetricsProvider"],
    "adapt":               true
}
```

`adapt` MUST be `true` if the package exports `Adapt()` per §10.3.

### 12.3 Status document

`<binary> status --format json`:

```json
{
    "protocol":  "CAN",
    "tool":      "go-can",
    "version":   "1.2.3",
    "healthy":   true,
    "connected": false,
    "endpoint":  "",
    "details":   {}
}
```

---

## 13. Implementation Naming

### 13.1 Repository names

`go-<PROTOCOL>`, `cpp-<PROTOCOL>`, `rust-<PROTOCOL>` where `<PROTOCOL>` is the
canonical uppercase name from §3 (e.g. `go-CAN`, `cpp-RCP`).

### 13.2 CLI binary names

Lowercase: `go-can`, `cpp-rcp`, `rust-someip`.

### 13.3 Package / namespace names

| Language | Pattern | Examples |
|---|---|---|
| Go | `package <protocol>` | `package can`, `package dds` |
| C++ | `namespace <protocol>` | `namespace can`, `namespace rcp` |
| Rust | `mod <protocol>` | `mod can`, `mod rcp` |

### 13.4 Go module paths

Protocol implementations: `github.com/SoundMatt/<RepoName>`  
RELAY package: `github.com/SoundMatt/RELAY`

Implementations import:
```go
import relay "github.com/SoundMatt/RELAY"
```

---

## 14. Subscriber Defaults and Helpers

| Parameter | Default | Notes |
|---|---|---|
| Channel depth | 64 | Messages buffered before back-pressure |
| Back-pressure | `DropNewest` | Drop arriving sample when channel full |

### 14.1 Standard helpers

Every implementation accepting subscription options MUST export these:

```go
type SubscriberConfig struct {
    ChannelDepth int                // 0 = use default (64)
    BackPressure BackPressurePolicy // 0 = DropNewest
}

type SubscriberOption func(*SubscriberConfig)

func WithChannelDepth(n int) SubscriberOption {
    return func(c *SubscriberConfig) { c.ChannelDepth = n }
}

func WithBackPressure(p BackPressurePolicy) SubscriberOption {
    return func(c *SubscriberConfig) { c.BackPressure = p }
}

func ApplySubscriberOpts(opts []SubscriberOption) SubscriberConfig {
    var c SubscriberConfig
    for _, o := range opts { o(&c) }
    return c
}

func (c SubscriberConfig) ChanDepth(defaultDepth int) int {
    if c.ChannelDepth > 0 { return c.ChannelDepth }
    return defaultDepth
}
```

The names `SubscriberConfig` and `SubscriberOption` MUST be used consistently across all protocols. (go-SOMEIP currently uses `SubscribeConfig`/`SubscribeOption` — tracked gap in Appendix A.)

---

## 15. Canonical Frame Types

The Go definitions below are authoritative. C++ and Rust equivalents MUST map
field-for-field (§18).

Every canonical type MUST implement:
```go
func (x T) ToMessage() relay.Message
func FromMessage(m relay.Message) (T, error)
```

These MUST be lossless for all mandatory fields. `Adapt()` (§10.3) uses them.

### 15.1 CAN — `Frame`, `Filter`, `LoanedFrame`

```go
type Frame struct {
    ID   uint32 `json:"id"`
    Ext  bool   `json:"ext,omitempty"`
    RTR  bool   `json:"rtr,omitempty"`
    FD   bool   `json:"fd,omitempty"`
    BRS  bool   `json:"brs,omitempty"`
    Data []byte `json:"data"`
}

type Filter struct {
    ID   uint32 `json:"id"`
    Mask uint32 `json:"mask"`
}

func (f Filter) Matches(fr Frame) bool { return fr.ID&f.Mask == f.ID&f.Mask }

type LoanedFrame struct {
    Frame
    release func()
}

func (f *LoanedFrame) Return() {
    if f.release != nil { f.release() }
}

const (
    CANMaxDataLen   = 8
    CANFDMaxDataLen = 64
    CANMaxStdID     = 0x7FF
    CANMaxExtID     = 0x1FFFFFFF
)

func MaxDataLen(fd bool) int {
    if fd { return CANFDMaxDataLen }
    return CANMaxDataLen
}
```

**Constraints enforced by `ValidateFrame`:**
- Standard ID (Ext=false): 0x000–0x7FF
- Extended ID (Ext=true): 0x00000000–0x1FFFFFFF
- BRS MUST be false when FD is false
- RTR MUST be false when FD is true
- len(Data) ≤ 8 when FD is false; ≤ 64 when FD is true

---

### 15.2 DDS — `Sample`, `QoS`, `GUID`, `BackPressurePolicy`

```go
type GUID [16]byte

type Sample struct {
    Topic          string    `json:"topic"`
    Payload        []byte    `json:"payload"`
    Timestamp      time.Time `json:"timestamp"`
    SequenceNumber uint64    `json:"seq"`
    WriterGUID     GUID      `json:"writer_guid"`
}

type QoS struct {
    Reliability       ReliabilityKind `json:"reliability"`
    Durability        DurabilityKind  `json:"durability"`
    HistoryDepth      int             `json:"history_depth"`
    Deadline          time.Duration   `json:"deadline"`
    MaxSampleSize     int             `json:"max_sample_size"`
    TransportPriority int             `json:"transport_priority"`
    LatencyBudget     time.Duration   `json:"latency_budget"`
    Lifespan          time.Duration   `json:"lifespan"`
    PublishPeriod     time.Duration   `json:"publish_period"`
}

type ReliabilityKind int
const ( BestEffort ReliabilityKind = 0; Reliable ReliabilityKind = 1 )

type DurabilityKind int
const ( Volatile DurabilityKind = 0; TransientLocal DurabilityKind = 1 )

// BackPressurePolicy is the canonical back-pressure type for all protocols (§14).
type BackPressurePolicy int
const (
    DropNewest BackPressurePolicy = iota
    DropOldest
    Block
)
```

**Defaults:** Reliability=BestEffort, Durability=Volatile, HistoryDepth=1,
Deadline=0 (disabled), MaxSampleSize=0 (unlimited), TSN fields=0.

---

### 15.3 LIN — `Frame`, `Filter`, `ScheduleEntry`

```go
type Frame struct {
    ID           uint8           `json:"id"`
    Data         []byte          `json:"data"`
    Checksum     uint8           `json:"checksum"`
    ChecksumType LINChecksumType `json:"checksum_type"`
}

type Filter struct {
    ID  uint8 `json:"id"`
    All bool  `json:"all"`
}

func (f Filter) Matches(fr Frame) bool { return f.All || fr.ID == f.ID }

type ScheduleEntry struct {
    ID      uint8  `json:"id"`
    DelayMs uint32 `json:"delay_ms"`
}

type LINChecksumType int
const ( ClassicChecksum LINChecksumType = 0; EnhancedChecksum LINChecksumType = 1 )

const (
    LINMaxDataLen     = 8
    LINMaxID          = 0x3F
    LINDiagRequestID  = 0x3C
    LINDiagResponseID = 0x3D
)
```

**Constraints enforced by `ValidateFrame`:** ID ≤ 0x3F; 1 ≤ len(Data) ≤ 8;
diagnostic frames (0x3C, 0x3D) MUST use `ClassicChecksum`.

---

### 15.4 MQTT — `Message`, `UserProperty`, `QoS`

```go
type Message struct {
    Topic           string         `json:"topic"`
    Payload         []byte         `json:"payload"`
    QoS             QoS            `json:"qos"`
    Retained        bool           `json:"retained,omitempty"`
    PacketID        uint16         `json:"packet_id,omitempty"`
    ResponseTopic   string         `json:"response_topic,omitempty"`
    CorrelationData []byte         `json:"correlation_data,omitempty"`
    UserProperties  []UserProperty `json:"user_properties,omitempty"`
    ContentType     string         `json:"content_type,omitempty"`
    ExpiryInterval  uint32         `json:"expiry_interval,omitempty"`
}

type UserProperty struct {
    Key   string `json:"key"`
    Value string `json:"value"`
}

type QoS int
const ( AtMostOnce QoS = 0; AtLeastOnce QoS = 1; ExactlyOnce QoS = 2 )
```

---

### 15.5 RCP — `Command`, `Response`, `Status`, `Loan`

Underlying types match go-RCP for zero-copy casting. JSON tags shown are the
canonical names; go-RCP currently lacks JSON tags (tracked gap in Appendix A).

```go
type Command struct {
    ID       uint32      `json:"id"`
    Zone     Zone        `json:"zone"`
    Type     CommandType `json:"type"`
    Priority Priority    `json:"priority"`
    Payload  []byte      `json:"payload,omitempty"`
}

type Response struct {
    CommandID uint32         `json:"command_id"`
    Zone      Zone           `json:"zone"`
    Status    ResponseStatus `json:"status"`
    Payload   []byte         `json:"payload,omitempty"`
}

type Status struct {
    Zone    Zone   `json:"zone"`
    Seq     uint32 `json:"seq"`
    Healthy bool   `json:"healthy"`
    Payload []byte `json:"payload,omitempty"`
}

type Loan struct {
    Payload []byte
    release func()
}

func (l *Loan) Return() {
    if l.release != nil { l.release() }
}

type Zone uint8
const (
    ZoneUnknown    Zone = 0; ZoneFrontLeft  Zone = 1; ZoneFrontRight Zone = 2
    ZoneRearLeft   Zone = 3; ZoneRearRight  Zone = 4; ZoneCentral    Zone = 5
)

type Priority uint8
const ( PriorityNormal Priority = 0; PriorityHigh Priority = 1; PriorityCritical Priority = 2 )

type CommandType uint16
const (
    CmdNoop CommandType = 0; CmdSet CommandType = 1; CmdGet     CommandType = 2
    CmdReset CommandType = 3; CmdWatchdog CommandType = 4
    CmdSleep CommandType = 5; CmdWake CommandType = 6
)

type ResponseStatus uint8
const (
    StatusOK      ResponseStatus = 0; StatusError   ResponseStatus = 1
    StatusTimeout ResponseStatus = 2; StatusBusy    ResponseStatus = 3
    StatusUnknown ResponseStatus = 4
)
```

---

### 15.6 SOME/IP — `Message`

```go
type Message struct {
    ServiceID        uint16      `json:"service_id"`
    MethodID         uint16      `json:"method_id"`
    ClientID         uint16      `json:"client_id"`
    SessionID        uint16      `json:"session_id"`
    ProtocolVersion  uint8       `json:"protocol_version"`  // MUST equal SOMEIPProtocolVersion
    InterfaceVersion uint8       `json:"interface_version"`
    MessageType      MessageType `json:"message_type"`
    ReturnCode       ReturnCode  `json:"return_code"`
    Payload          []byte      `json:"payload,omitempty"`
}

type ServiceID  = uint16; type MethodID = uint16
type ClientID   = uint16; type SessionID = uint16
type InstanceID = uint16; type EventID   = MethodID

type MessageType uint8
const (
    MsgTypeRequest           MessageType = 0x00
    MsgTypeRequestNoReturn   MessageType = 0x01
    MsgTypeNotification      MessageType = 0x02
    MsgTypeResponse          MessageType = 0x80
    MsgTypeError             MessageType = 0x81
    MsgTypeTPRequest         MessageType = 0x20
    MsgTypeTPRequestNoReturn MessageType = 0x21
    MsgTypeTPNotification    MessageType = 0x22
    MsgTypeTPResponse        MessageType = 0xA0
    MsgTypeTPError           MessageType = 0xA1
)

type ReturnCode uint8
const (
    RetOK ReturnCode = 0x00; RetNotOK ReturnCode = 0x01
    RetUnknownService ReturnCode = 0x02; RetUnknownMethod ReturnCode = 0x03
    RetNotReady ReturnCode = 0x04; RetNotReachable ReturnCode = 0x05
    RetTimeout ReturnCode = 0x06; RetWrongProtocolVersion ReturnCode = 0x07
    RetWrongInterfaceVersion ReturnCode = 0x08; RetMalformedMessage ReturnCode = 0x09
    RetWrongMessageType ReturnCode = 0x0A
)

const SOMEIPProtocolVersion uint8 = 0x01
```

**Constraint:** `ProtocolVersion` MUST equal `SOMEIPProtocolVersion` (0x01).
Implementations MUST reject inbound messages where this field is not 0x01.

---

## 16. Per-Protocol Defaults

| Protocol | Max payload | Channel depth | Default timeout |
|---|---|---|---|
| CAN classic | 8 bytes | 64 | — (bus-driven) |
| CAN FD | 64 bytes | 64 | — (bus-driven) |
| DDS | QoS.MaxSampleSize (0=unlimited) | 64 | QoS.Deadline (0=none) |
| LIN | 8 bytes | 64 | — (schedule-driven) |
| MQTT | 268,435,455 bytes (v5) | 64 | — (broker-driven) |
| RCP | unlimited | 64 | 5 s |
| SOME/IP | 2³²−16 bytes | 64 | 5 s |

---

## 17. Conformance Requirements

An implementation is **RELAY-conformant** if and only if:

1. **Protocol declaration.** Capabilities document (§12.2) declares a protocol from §3 and a `spec_version`.
2. **Protocol interfaces.** All mandatory interfaces from §8 are implemented with exact method signatures.
3. **Error sentinels.** All four sentinels in §5.1 are defined; protocol-specific errors wrap them per §5.2.
4. **Lifecycle invariants.** All nine requirements in §6 are satisfied.
5. **Constructor contract.** Each transport sub-package exports `New` per §7; a `mock` sub-package is present.
6. **Application interface.** The root package exports `Adapt()` per §10.3; the capabilities document declares `"adapt": true`.
7. **CLI mandatory commands.** `version`, `capabilities`, `status` per §11.1 with JSON schemas matching §12.
8. **Frame constraints.** `ValidateFrame` rejects all frames violating §15 constraints.
9. **Envelope conversion.** `ToMessage()` and `FromMessage()` are lossless for mandatory fields.
10. **Subscriber helpers.** `SubscriberConfig`, `SubscriberOption`, `ApplySubscriberOpts`, `ChanDepth` exported per §14.1; default depth is 64.
11. **Protocol-specific constraints.**
    - CAN: `ValidateFrame` enforces BRS/FD and RTR/FD rules.
    - DDS: `Domain` MUST be validated as 0–232.
    - LIN: `ValidateFrame` enforces diagnostic checksum rule.
    - SOMEIP: `ProtocolVersion` MUST be validated as 0x01 on send and receive.
12. **SpecVersion constant.** Package exports `SpecVersion = "0.1"`.

`relay conform <binary>` verifies requirements 1, 3, 5 (mock presence), 6, 7, 8, and 12.
Requirements 2, 4, 9, 10, and 11 are verified by the implementation's own test suite.

---

## 18. Language Bindings

### 18.1 Go (canonical)

The RELAY Go package (`github.com/SoundMatt/RELAY`) exports all types from §3,
§4, §5, §8, §9, §10, §14, and §15.

### 18.2 C++

#### relay::Context

```cpp
namespace relay {
class Context {
public:
    static Context background() noexcept;
    static Context with_deadline(std::chrono::steady_clock::time_point) noexcept;
    static Context with_timeout(std::chrono::steady_clock::duration) noexcept;
    bool done() const noexcept;
    std::optional<std::chrono::steady_clock::time_point> deadline() const noexcept;
};
} // namespace relay
```

Already implemented as `rcp::Context` in cpp-RCP; cpp-RCP SHOULD alias
`rcp::Context = relay::Context`.

#### relay::Channel<T>

```cpp
namespace relay {
template<typename T>
class Channel {
public:
    explicit Channel(std::size_t capacity);
    bool push(T value);
    std::optional<T> recv();
    std::optional<T> try_recv();
    void close() noexcept;
    bool is_closed() const noexcept;
};
} // namespace relay
```

Already implemented as `rcp::StatusChannel`; cpp-RCP SHOULD alias
`StatusChannel = relay::Channel<Status>`.

#### relay::Node and relay::Caller (C++)

```cpp
namespace relay {

class Node {
public:
    virtual Protocol protocol() const noexcept = 0;
    virtual std::error_code send(Context ctx, const Message& msg) = 0;
    virtual std::pair<Channel<Message>, std::error_code>
        subscribe(SubscriberOptions opts = {}) = 0;
    virtual std::error_code close() noexcept = 0;
    virtual ~Node() = default;
};

class Caller : public Node {
public:
    virtual std::pair<Message, std::error_code>
        call(Context ctx, const Message& req) = 0;
};

} // namespace relay
```

#### Error codes

```cpp
namespace relay {
enum class Errc { closed, not_connected, timeout, payload_too_large };
} // namespace relay
// Registered as std::error_category "relay".
```

Protocol-specific codes map to `relay::Errc` via `std::error_condition` equivalence.

### 18.3 Rust

#### Async-primary model

All blocking operations MUST be `async fn`. Required runtime: `tokio` or compatible.
Sync wrappers MAY be provided in a `blocking` sub-module.

#### relay::Node and relay::Caller (Rust)

```rust
use async_trait::async_trait;

#[async_trait]
pub trait Node: Send + Sync {
    fn protocol(&self) -> Protocol;
    async fn send(&self, ctx: Context, msg: Message) -> Result<(), Error>;
    async fn subscribe(&self, opts: SubscriberOptions)
        -> Result<tokio::sync::mpsc::Receiver<Message>, Error>;
    async fn close(&self) -> Result<(), Error>;
}

#[async_trait]
pub trait Caller: Node {
    async fn call(&self, ctx: Context, req: Message) -> Result<Message, Error>;
}
```

#### Context

```rust
pub struct Context {
    deadline: Option<std::time::Instant>,
}

impl Context {
    pub fn background() -> Self { Self { deadline: None } }
    pub fn with_timeout(d: std::time::Duration) -> Self {
        Self { deadline: Some(std::time::Instant::now() + d) }
    }
    pub fn done(&self) -> bool {
        self.deadline.map_or(false, |d| std::time::Instant::now() >= d)
    }
}
```

---

## 19. Versioning

### 19.1 Scheme

`RELAY spec vMAJOR.MINOR`

| Change | Increment |
|---|---|
| Breaking change to canonical types, interface signatures, or lifecycle requirements | MAJOR |
| New optional interface, protocol, CLI command, canonical field, or `Adapt()` target | MINOR |
| Clarification, editorial, Appendix A update | No change |

### 19.2 Deprecation policy

Before a MUST requirement is removed or inverted:
1. Mark deprecated in a MINOR release with documented replacement.
2. At least one further MINOR release MUST pass before removal.
3. CHANGELOG.md MUST record the deprecation, replacement, and planned removal version.

### 19.3 Compatibility

An implementation declares `"spec_version": "0.1"` in its capabilities document.
`relay conform` MUST accept implementations targeting any MINOR version within
the current MAJOR.

### 19.4 Machine-readable version

`spec/version.json` is authoritative. The spec document title is informational.

Current version: **v0.1**

**Go:** `const SpecVersion = "0.1"`  
**C++:** `constexpr std::string_view kRelaySpecVersion = "0.1";`  
**Rust:** `pub const RELAY_SPEC_VERSION: &str = "0.1";`

---

## Appendix A — Current project alignment

| Requirement | go-CAN | go-DDS | go-LIN | go-mqtt | go-RCP | go-SOMEIP | cpp-RCP |
|---|---|---|---|---|---|---|---|
| Canonical frame type | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| JSON tags on structs | ✗ | ✅ | ✗ | ✅ | ✗ | ✅ | n/a |
| ErrClosed | ✗ | ✅ | ✗ | ✅ | ✅ | ✅ | ✅ |
| ErrNotConnected | ✗ | n/a | ✗ | ✅ | ✗ | ⚠ rename | ✗ |
| ErrTimeout | ✗ | ✅ ctx | ✗ | ✗ | ✅ | ✅ | ✅ |
| ErrPayloadTooLarge | ✗ | ✅ | ✗ | ✅ | ✗ | ✗ | ✗ |
| Error wrapping (errors.Is) | ✗ | ✅ | ✗ | ✅ | partial | partial | n/a |
| ValidateFrame | ✅ | n/a | ✅ | n/a | n/a | n/a | n/a |
| MaxDataLen | ✅ | n/a | n/a | n/a | n/a | n/a | n/a |
| SubscriberConfig / SubscriberOption | ✗ | ✅ | ✗ | ✅ | ✗ | ⚠ rename | n/a |
| ApplySubscriberOpts / ChanDepth | ✗ | ✅ | ✗ | ✅ | ✗ | ⚠ rename | n/a |
| BackPressurePolicy | ✗ | ✅ | ✗ | ✗ | ✗ | ✗ | n/a |
| mock / virtual sub-package | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| Adapt() → relay.Node / relay.Caller | ✗ | ✗ | ✗ | ✗ | ✗ | ✗ | ✗ |
| ToMessage / FromMessage | ✗ | ✗ | ✗ | ✗ | ✗ | ✗ | ✗ |
| SpecVersion constant | ✗ | ✗ | ✗ | ✗ | ✗ | ✗ | ✗ |
| CLI: version --format json | ? | ? | ? | ? | ? | ? | ? |
| CLI: capabilities | ✗ | ✗ | ✗ | ✗ | ✗ | ✗ | ✗ |
| CLI binary name per §13.2 | ✗ cantool | ? | ✗ lintool | ? | ✗ rcptool | ? | ? |
| HealthProvider | ✗ | ✅ | ✗ | ✗ | ✗ | ✗ | ✗ |
| MetricsProvider | ✗ | ✅ | ✗ | ✗ | ✗ | ✗ | ✗ |
| Drainer | ✗ | ✅ | ✗ | ✗ | ✗ | ✗ | ✗ |
| LoaningBus / LoaningPublisher / LoaningController | ? | ✅ | n/a | n/a | ✅ | n/a | ✅ |
| relay::Context (C++) | n/a | n/a | n/a | n/a | n/a | n/a | ✅ as rcp:: |

**Legend:** ✅ conforms · ✗ missing · ⚠ partial/rename · ? unknown · n/a not applicable
