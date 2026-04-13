# naad-xml -- CAP-CP XML Message Parsing

This package implements Go types and methods for parsing, processing, and
summarizing messages in the Common Alerting Protocol Canadian Profile (CAP-CP)
XML format, which is the wire format used by the National Alert Aggregation and
Dissemination (NAAD) system.

## Scope

The package handles:

- XML deserialization of CAP-CP alerts into Go structs.
- Parsing of CAP-CP extensions: SOREM, Environment Canada, and CAP-CP profile
  layers embedded as alert parameters.
- Custom XML unmarshaling for geographic shapes (polygons, circles) and
  reference lists (the `Sender,Identifier,Timestamp` format).
- Fetching of external resources (broadcast audio URLs) and decoding of base64
  embedded resources.
- Construction of archive-server URLs for retrieving previously published
  alerts by reference.
- Generation of lightweight `AlertSummary` / `InfoSummary` structs suitable
  for web UI rendering.

## Key Types

| Type | File | Purpose |
|------|------|---------|
| `Alert` | format.go | Top-level CAP message: identifier, sender, timestamps, status, type, scope, references, and a slice of `AlertInfo` entries (one per language). |
| `AlertInfo` | format.go | Per-language body of the alert: event, urgency, severity, certainty, headline, description, areas, resources, and parsed layer data. |
| `AlertArea` | format.go | Geographic target area with description, optional polygon/circle, and geocode parameters. |
| `Resource` | resource.go | Attached or linked resource (audio, images, etc.) with MIME type, optional base64 content, and a `Fetch()` method for URL-type resources. |
| `References` / `Reference` | references.go | Parsed update-chain references with sender, identifier, timestamp, and methods to construct archive URLs and fetch the referenced alert. `FetchWithContext` supports context-based cancellation and timeout. |
| `FetchTracker` | fetchtracker.go | Deduplicates concurrent archive fetches for the same alert identifier. The first caller performs the HTTP request; subsequent callers block until the result is ready. Uses a per-fetch timeout (30s) independent of caller context and caches results (including errors) for a configurable TTL (2min). Used by both `naad-cache` and `naad-socket` to avoid redundant archive requests. |
| `GeoPolygon` / `GeoCircle` | geopoint.go | Geometric area definitions with custom XML unmarshalers for the CAP `lat,lon` pair format. |
| `Sorem` | sorem.go | SOREM layer data: broadcast/wireless immediacy flags and override text. |
| `EC` | environmentcanada.go | Environment Canada layer: event metadata, intrusive-broadcast flag, alert coverage, designation codes, CAP count. |
| `CAP` | cap-cp.go | CAP-CP profile layer: minor-change flag and change type. |
| `AlertSummary` / `InfoSummary` | summary.go | Flattened views of an alert for display, keyed by language. |
| `Layer` / `Profile` | layers.go | Parsed format+version pairs from the `Code` field. |

## Enumerations (format.go)

`AlertStatus`, `MessageType`, `AlertScope`, `Urgency`, `Severity`,
`Certainty`, `Category`, `ResponseType` -- string-typed constants mapping
directly to the CAP-CP enumeration values.

## Processing Pipeline

1. `xml.Decoder.Decode(&alert)` deserializes the raw XML.
2. `alert.ProcessAlert()` iterates each `AlertInfo` and calls `ProcessInfo()`.
3. `ProcessInfo()` calls `ProcessParams()` to parse layer/profile parameters
   into `SoremLayer`, `ECLayer`, and `CAPLayer` structs.
4. `ProcessInfo()` also fetches any `application/x-url` broadcast audio
   resource so it is available for later playback.

## Issues and Improvement Opportunities

- **`ProcessAlert()` returns only the last error.** Individual `AlertInfo`
  processing errors are now logged with the error detail and the last error
  is returned to the caller. A multi-error collector would give callers
  full visibility into partial failures.

- **Resource fetch failures are silent to the caller.** `ProcessInfo()` logs
  resource-fetch errors but does not propagate them. If the broadcast audio
  cannot be retrieved, downstream audio processing will silently fall back to
  TTS without the caller being aware of the degradation.

- **No validation of required fields.** CAP-CP mandates certain fields be
  present (e.g., at least one `info` block, area descriptions, etc.). The
  package trusts the input entirely. Adding a `Validate()` method would help
  catch malformed messages early rather than letting them cause nil-pointer
  panics later in the pipeline.

- **`GetLayers()` is called implicitly** during `ProcessParams()` but the
  parsed `Layers` and `Profiles` are stored on the `Alert` struct, which
  `AlertInfo` does not have a back-reference to. The layer/profile data is
  parsed from `Code` strings, but the coupling between `Alert.Code` and
  `AlertInfo.Parameters` is indirect and could be clearer.
