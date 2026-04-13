# naad-socket -- NAAD Stream Receivers

This package provides TCP and UDP multicast receivers for the NAAD alert
streaming infrastructure, plus a `ReceiverGroup` manager that multiplexes
multiple sources, deduplicates messages, handles heartbeats, and fetches
referenced alerts from archive servers.

## Architecture

```
                   +-----------+     +-----------+     +----------+
                   | TCP Recv  |---->|           |     |          |
                   | (stream1) |     |           |     | Archive  |
                   +-----------+     | Receiver  |---->| Fetcher  |
                   +-----------+     | Group     |     |          |
                   | TCP Recv  |---->| (dedup +  |     +-----+----+
                   | (stream2) |     |  heartbeat|           |
                   +-----------+     |  handler) |     outChannel
                   +-----------+     |           |---------->
                   | UDP Listen|---->|           |
                   | (multicast)     +-----------+
                   +-----------+
```

All receivers implement the `Receiver` interface (Connect, Disconnect,
SetHandler) and push decoded `Alert` structs into a shared channel. The
`ReceiverGroup` consumes that channel, filters heartbeats, deduplicates, and
forwards to the application via `outChannel`.

## Key Types

### Receiver (interface.go)

```go
type Receiver interface {
    Connect() error
    Disconnect()
    SetHandler(func(*naadxml.Alert) error)
}
```

### TCPReceiver (stream.go)

Connects to a NAAD TCP streaming server (e.g.,
`streaming1.naad-adna.pelmorex.com:8080`). Runs an `xml.Decoder` in a
goroutine (`listen()`). On connection loss, the single goroutine manages its
own reconnection with exponential backoff (5 seconds to 2 minutes). An
`isConnectionError()` helper distinguishes network-level errors (EOF, network
reset) from XML parsing errors for logging purposes. The `dial()` method
handles TCP connection establishment with a 1-minute timeout.

### Listener (udplisten.go)

Joins a UDP multicast group (e.g., `224.0.10.10:25555`) and decodes XML
messages from the datagram stream. Supports IPv6 multicast addresses.

### ReceiverGroup (receivermanager.go)

Manages a set of receivers and provides:

- **Source multiplexing**: Accepts `tcp://` and `udp://` source URIs, creates
  the appropriate receiver type for each.
- **Heartbeat handling**: Messages from `NAADS-Heartbeat` sender are not
  forwarded; instead their references are queued for archive fetch.
- **Deduplication**: Messages seen from multiple sources are only forwarded
  once.
- **Reference fetching**: A background goroutine retrieves referenced messages
  from archive servers (via a shared `FetchTracker`) when they haven't been
  seen before. The tracker deduplicates concurrent fetch requests that may
  also originate from the cache's update-chain resolution.
- **Periodic cleanup**: Every 30 minutes, dedup entries older than 24 hours
  are purged.

## Issues and Improvement Opportunities

- **Connection timeout hardcoded** (stream.go). The 1-minute dial timeout
  is not configurable. In a poor-network environment this may be too short;
  in a fast-failover scenario it may be too long.
