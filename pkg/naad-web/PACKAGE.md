# naad-web -- Web UI and API

This package provides an HTTP server with both a JSON API and an HTML UI for
viewing cached emergency alerts.

## Endpoints

### JSON API

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/history` | Returns all `AlertHistory` entries as JSON, sorted by most recent update |
| GET | `/api/history/{id}` | Returns a single alert's history by identifier (**not implemented**) |
| GET | `/api/summary` | Returns `AlertSummary` objects for all cached alerts |

### HTML UI

| Method | Path | Description |
|--------|------|-------------|
| GET | `/alertcache` | Table of all cached raw alerts with color-coded severity |
| GET | `/alertsummary` | Table of alert summaries (same template, fed from history) |
| GET | `/alertdetail/{id}` | Detailed view of a single alert with SOREM/EC layers, resources, references, and update history |

## HTML Templates

Templates are embedded via `go:embed` from `html/`:

### alerts.html
- Auto-refreshes every 10 seconds (`<meta http-equiv="refresh">`).
- Color codes rows by severity: red (Extreme), orange (Severe), yellow
  (Moderate), blue (Minor), green (Past urgency), grey (expired).
- Links each identifier to the detail view.
- Displays the `en-CA` language info block.

### alertdetail.html
- Shows full alert metadata in a table.
- Conditionally renders SOREM and EC layer details if present.
- Lists all resources with clickable links for URL-type resources.
- Shows the reference chain (what this alert references).
- Shows the update chain (what has updated this alert).

## Server Configuration

- Read/write timeouts: 10 seconds
- Max header size: 1 MB
- Runs in a background goroutine; `Shutdown()` performs graceful stop.

## Issues and Improvement Opportunities

- **`HandleGetAlert` is empty** (webserver.go:77-78). The `/api/history/{id}`
  endpoint is registered but the handler has no implementation -- it returns
  an empty 200 response.

- **No Content-Type headers.** JSON endpoints do not set
  `Content-Type: application/json`. Browsers and API clients may
  misinterpret the response.

- **WriteHeader after body write.** In error paths (webserver.go:93,
  ui.go:68, etc.), `w.WriteHeader(http.StatusInternalServerError)` is called
  after `encoder.Encode()` or `template.Execute()` has already started
  writing to the response. At that point the status code is already 200 and
  the `WriteHeader` call is a no-op (Go's http package logs a warning).

- **No static asset serving.** The HTML templates are bare HTML with no CSS
  or JavaScript. Adding a minimal stylesheet would improve readability
  significantly, especially for the color-coded alert tables.

- **No index/root handler.** There is no handler for `/` -- users must know
  to navigate to `/alertsummary` or `/alertcache`.

- **Template nil-safety.** The alert detail template accesses
  `.History.Current.Info` and `.History.UpdatedBy` directly. If an alert
  exists in the cache but has no history entry (e.g., it was an update-only
  message), accessing `.History` fields would panic with a nil pointer.

- **Shutdown uses `context.TODO()`** (webserver.go:74). A proper shutdown
  context with a timeout would be more robust.
