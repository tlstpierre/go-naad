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

- **No static asset serving.** The HTML templates are bare HTML with no CSS
  or JavaScript. Adding a minimal stylesheet would improve readability
  significantly, especially for the color-coded alert tables.
