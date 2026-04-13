## Go Tools for NAAD and CAP-CP Alert System

This project implements a modular receiver and processor for the Canadian
National Alert Aggregation and Dissemination (NAAD) system. Alerts are
published in the Common Alerting Protocol Canadian Profile (CAP-CP) XML
format over redundant TCP streams and a UDP multicast group. This project
provides Go packages to receive, parse, filter, cache, display, and announce
those alerts.

By composing the packages into a pipeline, a customizable Emergency Alert
System (EAS) receiver can be built -- the included `testreceiver` application
demonstrates this.

### System Architecture

```
  NAAD Infrastructure               go-naad Pipeline                  Outputs
  ====================         ========================          ==============

  TCP streaming servers  \                                     /  Console display
  (streaming1, streaming2) --> naad-socket (ReceiverGroup) --> |  naad-web (HTTP)
  UDP multicast (224.0.10.10) /   |  dedup, heartbeat,    |   |  naad-audio (RTP
                                  |  archive fetch         |    \   multicast)
  Archive servers  <--------------+                        |
  (capcp1, capcp2)                                         |
                                  naad-xml (decode, process,
                                   layers, resources)
                                        |
                                  naad-filter (CAP codes,
                                   lat/lon, dedup, update
                                   suppression)
                                        |
                                  naad-cache (history,
                                   update chain tracking,
                                   expiration)
```

### Packages

Each package has a `PACKAGE.md` file with detailed documentation, key types,
and notes on issues and improvement opportunities.

| Package | Import Path | Purpose | Docs |
|---------|-------------|---------|------|
| **naad-xml** | `pkg/naad-xml` | CAP-CP XML parsing, layer processing, resource handling, alert summaries | [PACKAGE.md](pkg/naad-xml/PACKAGE.md) |
| **naad-socket** | `pkg/naad-socket` | TCP and UDP multicast receivers, receiver multiplexing, heartbeat handling, archive fetch | [PACKAGE.md](pkg/naad-socket/PACKAGE.md) |
| **naad-audio** | `pkg/naad-audio` | Audio pipeline: broadcast audio retrieval, MP3 decode, TTS, CAAS tone synthesis, G.722/mu-law encoding, RTP multicast delivery | [PACKAGE.md](pkg/naad-audio/PACKAGE.md) |
| **naad-filter** | `pkg/naad-filter` | Geographic filtering (CAP codes and s2 geometry), message deduplication, update suppression | [PACKAGE.md](pkg/naad-filter/PACKAGE.md) |
| **naad-cache** | `pkg/naad-cache` | In-memory alert cache with update-chain tracking and expiration management | [PACKAGE.md](pkg/naad-cache/PACKAGE.md) |
| **naad-web** | `pkg/naad-web` | HTTP server with JSON API and HTML UI for viewing cached alerts | [PACKAGE.md](pkg/naad-web/PACKAGE.md) |

### Application

| Application | Path | Purpose | Docs |
|-------------|------|---------|------|
| **testreceiver** | `cmd/testreceiver` | Complete NAAD receiver application wiring all packages together | [APPLICATION.md](cmd/testreceiver/APPLICATION.md) |

### External Dependencies

| Dependency | Purpose |
|------------|---------|
| `github.com/tlstpierre/mc-audio` | Custom library for RTP multicast transmission, G.711 encoding, and Piper TTS integration |
| `github.com/golang/geo` | S2 spherical geometry for geographic area matching |
| `github.com/gorilla/mux` | HTTP request routing |
| `github.com/hajimehoshi/go-mp3` | MP3 audio decoding |
| `github.com/shenjinti/go722` | G.722 audio codec |
| `github.com/sirupsen/logrus` | Structured logging |
| `gopkg.in/yaml.v2` | YAML configuration parsing |

### Configuration

The application is configured via a YAML file (`-config` flag). If no config
is specified, built-in defaults are used (see `cmd/testreceiver/config.go`).

Key configuration areas:
- **Stream servers**: TCP and UDP sources to receive alerts from.
- **Archive servers**: HTTP servers to fetch referenced/missed alerts.
- **CAP codes**: Geographic area codes to filter for local alerts.
- **Lat/Lon**: Receiver location for geometric area matching.
- **Web listen**: Address and port for the HTTP server.
- **TTS config**: Piper TTS engine path, voice, and sample rate.
- **Channels**: Named audio output channels, each with language, CAP code
  filters, codec selection, and multicast addresses.

### Running

```sh
# With default configuration
go run ./cmd/testreceiver/

# With a config file and debug logging
go run ./cmd/testreceiver/ -config myconfig.yaml -loglevel debug

# With log output to file
go run ./cmd/testreceiver/ -config myconfig.yaml -logfile /var/log/naad.log
```

### Test Data

The `testdata/` directory contains sample CAP-CP XML messages covering various
scenarios: alerts with no attachments, embedded audio, external audio links,
geographic polygons and circles, SOREM layers, Environment Canada layers,
minor updates, and WPAS profiles.

### Remaining Issues and Cleanup Priorities

1. **Error propagation.** `naad-xml/process.go` `ProcessAlert()` logs errors
   but always returns nil. Callers cannot tell when processing was degraded.

2. **No persistence.** The in-memory cache is lost on restart. Periodic
   state serialization would improve resilience across process restarts.

3. **Incomplete Place geometry.** `naad-filter/place.go` has two unimplemented
   methods (`Boundary.InCircle`, `Circle.InPolygon`). These only matter for
   non-point receiver locations but should be completed or explicitly
   documented as unsupported.

### License

MIT License. See [LICENSE](LICENSE).
