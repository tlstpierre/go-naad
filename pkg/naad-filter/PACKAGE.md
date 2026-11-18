# naad-filter -- Message Filtering and Deduplication

This package provides geographic filtering, message deduplication, and update
suppression for the NAAD alert pipeline.

## Components

### Geographic Matching

Two complementary approaches are provided:

**CAP Code matching** (location.go): Checks whether an alert's geocode
parameters contain a specific CAP-CP location code prefix. CAP codes are
hierarchical (e.g., `3518020` for Scugog), so a prefix match can capture
broader regions. `IsCAPArea()` checks a single area, `AlertIsCAPArea()` checks
all areas across all info blocks.

**Geometric matching** (area.go, place.go): Uses the `golang/geo` s2 library
for spherical geometry. Alert areas can contain any number of polygons and
circles; the `IsPlaceInArea()` function tests whether a `Place` (point,
boundary, or circle) overlaps any of them.

CAP polygons have no defined winding order. `LoopFromPolygon()` drops the
repeated closing vertex and normalizes the loop, so clockwise polygons are not
treated as "everywhere except this area". Degenerate or malformed geometry is
skipped (logged), never fatal, and a panic in s2 is recovered.

**Geofences** (geofence.go): `GeoFence{Lat, Lon, RadiusKm}` is the per-channel
config type (`geofences:` in YAML). `AlertInGeoFence()` matches when any alert
polygon or circle overlaps the fence: centre inside the polygon, a vertex
inside the fence, or an edge passing within the radius.

**Only areas in effect match.** `AlertInGeoFence`, `IsPlaceInAlert` and
`AlertIsCAPArea` skip stand-down blocks (an ECCC `Alert_Location_Status`
other than `active`, or any issuer's `responseType` `AllClear`) and ECCC's
drawn threat areas whose `DLC` geocode is `ended` or `cancelled` (see
`naadxml.AlertInfo.InEffect` and `AlertArea.InEffect`). Those areas stay in
the parsed alert, for maps, but never start an alert. A stand-down reaches
the channels that had the alert through its `<references>` instead (see
`matchChannel` in `cmd/testreceiver`).

### Place Types (place.go)

The `Place` interface abstracts different geographic shapes:

| Type | Description | InPolygon | InCircle |
|------|-------------|-----------|----------|
| `Point` | Single lat/lon coordinate | s2 containment check | s2 cap containment check |
| `Boundary` | Polygon defined by vertices | s2 loop intersection | cap-loop intersection via `capIntersectsLoop` |
| `Circle` | Center point + radius in km | cap-loop intersection via `capIntersectsLoop` | s2 cap intersection |

Helper functions:
- `PointFromLatLon(lat, lon)` -- creates a Point
- `KmToAngle(km)` / `AngleToKm(angle)` -- unit conversions
- `CircleFromLLRadius(lat, lon, radius)` -- creates a Circle
- `BoundaryFromPoints([]Point)` -- creates a polygon Boundary

### Deduplication (deduplicate.go)

`DeDuplicate` tracks two categories of seen identifiers:

- **Messages**: Prevents the same alert from being processed twice when
  received from multiple TCP/UDP sources.
- **References**: Prevents re-fetching alerts from archive servers that have
  already been retrieved via heartbeat references.

Both use `map[string]time.Time` with `sync.RWMutex` for thread safety.
`CleanOld(age)` purges message entries older than the given duration and
reference entries older than 24 hours.

### Update Suppression (updatesuppress.go)

`UpdateSuppressor` prevents repeated audio announcements of the same alert
within a configurable time window. It uses a composite key of
`(Identifier, Urgency, Severity)` from the first info block -- so if an
update changes only the description but keeps the same urgency/severity, it
is suppressed. If the severity escalates, it will be announced again.

Methods:
- `Insert(alert)` -- record an alert
- `IsDuplicate(alert)` -- true if seen within the repeat window
- `Delete(alert)` -- remove a specific entry
- `Clean()` -- purge expired entries

## Issues and Improvement Opportunities


- **DeDuplicate.CleanOld uses a hardcoded 24-hour window for references**
  (deduplicate.go) regardless of the `age` parameter passed in for messages.
  This is intentional but could be clearer if made a separate parameter.
