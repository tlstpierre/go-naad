# testreceiver -- NAAD Alert Receiver Application

This is the main application that wires together all the packages into a
working emergency alert receiver. It connects to NAAD streaming servers,
processes incoming alerts, displays them on the console, serves a web UI,
and optionally announces them via RTP multicast audio.

## Message Flow

```
  NAAD Servers              Application                    Outputs
  ============          ================                ============

  TCP stream 1 --\                                     /-- Console display
  TCP stream 2 ---+--> ReceiverGroup --> Processor --+--> Web UI (cache)
  UDP multicast -/     (dedup, hbeat)    (cache,     \-- Audio announcer(s)
                        |                 process)         |
                   Archive fetch                     Per-channel:
                   (heartbeat refs)                   - TTS or broadcast audio
                                                      - CAAS tone / chime
                                                      - G.722 or mu-law
                                                      - RTP multicast
```

## Files

### main.go

Entry point. Parses CLI flags, loads config, and orchestrates startup:

1. Initialize and load configuration.
2. Create the alert cache and set archive servers.
3. Create a `ReceiverGroup` from configured stream/multicast sources.
4. Initialize the geographic filter from config lat/lon.
5. Start the web server.
6. Initialize audio announcer channels.
7. Create the Processor (connects receiver output to cache + announcer).
8. Start the receiver group.
9. Enter the main loop: handle OS signals, periodic cache cleanup (hourly),
   and display of processed alert info.

**CLI flags:**
- `-loglevel` (default: `info`)
- `-logfile` (default: stderr)
- `-config` (default: use built-in defaults)

**Signal handling:** SIGINT, SIGTERM, SIGQUIT, SIGKILL trigger graceful
shutdown. SIGHUP is caught but ignored (could be used for config reload).

### config.go

YAML configuration with sensible defaults for the Canadian NAAD system.

Default stream servers:
- `tcp://streaming1.naad-adna.pelmorex.com:8080`
- `tcp://streaming2.naad-adna.pelmorex.com:8080`
- `udp://224.0.10.10:25555`

Default archive servers:
- `capcp1.naad-adna.pelmorex.com`
- `capcp2.naad-adna.pelmorex.com`

Default location: Scugog/Uxbridge, Ontario (CAP codes 3518020, 3518029).

### processor.go

Receives alerts from the receiver group channel, adds them to the cache,
calls `ProcessAlert()`, routes to the announcer, and (intended but currently
bypassed) sends info blocks to the display channel.

### announcer.go

Creates one `Announcer` per configured channel. Each announcer has:
- Its own `TransmitChannel` for audio output.
- A `UpdateSuppressor` with a 2-hour window to avoid re-announcing the same
  alert.
- CAP code filtering: only announces alerts matching the channel's codes.

### display.go

Console output for processed alerts. Checks both CAP code and lat/lon
matching and warns when an alert is local.

## Issues and Improvement Opportunities

- **SIGKILL cannot be caught.** The signal handler lists `syscall.SIGKILL`
  (main.go), but SIGKILL cannot be intercepted by any process. Go will
  accept it in the Notify call without error, but it will never be delivered
  to the channel.

- **`signal.Notify(sigs)` catches all signals** (main.go). Without an
  explicit signal list, every signal (including SIGCHLD, SIGWINCH, etc.)
  is delivered. This is noisy but harmless; being explicit about which
  signals to catch would be cleaner.

- **AnnouncerInit error return is ignored** (main.go). The function
  returns an error but it is not checked.

- **webServer error not checked** (main.go). `NewServer` returns an error
  that is not checked.

- **No config file validation.** If the YAML has unexpected types or missing
  required fields, the decoder will silently use zero values. A validation
  step after loading would catch misconfiguration early.

- **No health/liveness endpoint.** For deployment in a monitored environment,
  an endpoint that reports receiver connection status, last heartbeat time,
  and cache size would aid operations.
