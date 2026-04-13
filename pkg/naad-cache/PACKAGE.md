# naad-cache -- Alert Cache and History Tracking

This package provides an in-memory cache of alert messages with update-chain
correlation and expiration management.

## Purpose

The NAAD system sends alerts that may be updated or cancelled over time. The
cache tracks the full lifecycle of each alert: original publication, subsequent
updates, and expiration. This makes it suitable as a backend for the web UI and
for any component that needs to know the current state of active alerts.

## Key Types

### Cache (cache.go)

Thread-safe (via `sync.RWMutex`) map-based store with two internal maps:

- `alerts`: All raw `Alert` structs keyed by identifier.
- `history`: `AlertHistory` entries keyed by the *original* alert identifier,
  tracking the update chain.

Configuration:
- `ArchiveTime`: How long to keep expired alerts (default 72 hours).
- `archiveServers`: HTTP archive server hostnames for fetching missed originals.

### AlertHistory (cache.go)

Tracks the lifecycle of a single alert thread:

| Field | Purpose |
|-------|---------|
| `IsOriginal` | True if this was the first message (not an update) |
| `Updated` | True if at least one update has been applied |
| `IsUpdate` | True if the *current* version is an update |
| `LastUpdate` | Timestamp of the most recent update |
| `Expired` | Expiration time (zero value if still active) |
| `Identifier` | The original alert identifier |
| `Current` | Pointer to the most recent alert version |
| `Original` | Pointer to the first alert in the chain |
| `UpdatedBy` | List of update message identifiers |

## Operations

### Add(alert)

Two-phase operation to avoid holding the write lock during network I/O:

- **Phase 1 (under lock):** Stores the raw alert. For updates, identifies
  which referenced originals are missing from cache and applies updates for
  those already present. For new alerts, creates a fresh `AlertHistory` entry.
- **Phase 2 (outside lock):** Fetches missing originals via the shared
  `FetchTracker`, which deduplicates requests that may also come from the
  receiver group's heartbeat reference fetcher. Re-acquires the lock briefly
  to store each fetched result and link the update chain.

### Get(id) -> (Alert, AlertHistory)

Returns the raw alert and its history entry. Returns nil history if the
identifier is not tracked (e.g., it was an update message rather than an
original).

### Clean()

Iterates all history entries and checks expiration:
- Marks the `Expired` timestamp from the first expired info block.
- Deletes history entries that have been expired longer than `ArchiveTime`.
- Deletes raw alerts whose info blocks are all expired beyond `ArchiveTime`.

### DumpHistory() / DumpAlerts()

Returns sorted snapshots for rendering:
- `CacheHistory`: Sorted by `LastUpdate` descending, then `Original.Sent`,
  then identifier.
- `AlertList`: Sorted by `Sent` descending, then identifier.

## Issues and Improvement Opportunities

- **`ApplyUpdate` replaces Current entirely** (cache.go). The commented-out
  code suggests there was an intent to merge per-info-block updates, but
  currently the entire alert is replaced. If an update only changes one
  language's info block, the previous version is lost from `Current` (though
  `Original` is preserved).

- **No persistence.** The cache is purely in-memory. A process restart loses
  all alert history. For a reliability-critical system, periodic serialization
  to disk (or a lightweight embedded database) would allow recovery after
  restarts without re-fetching everything from archive servers.
