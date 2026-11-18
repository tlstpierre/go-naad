# naad-audio -- Audio Processing and Delivery

This package generates audio output for emergency alerts, including broadcast
audio retrieval, text-to-speech synthesis, alert tone generation, codec
encoding, and RTP multicast delivery.

## Overview

When an alert arrives, the audio pipeline works as follows:

1. **Audio source selection** (content.go): Look for a "Broadcast Audio"
   resource on the alert. If found, fetch and decompress the MP3. If not, fall
   back to SOREM broadcast text or generate TTS from the alert fields.
2. **Alert tone prepend** (content.go): If the alert is an emergency
   (SOREM Broadcast Immediate, or Extreme/Severe severity with non-Past
   urgency), prepend the CAAS 8-tone alert signal. Otherwise, prepend a
   3-note announcement chime. Cooldown timers prevent rapid re-triggering.
3. **Encoding and delivery** (delivery.go): Downsample to the target rate,
   encode to G.722 (64 kbps) or mu-law (G.711), and transmit via RTP
   multicast to configured IP addresses.

## Key Types

### TransmitChannel (delivery.go)

The main audio output channel. Created per configured channel name (e.g.,
"test", "paging", etc.). Each channel has its own:

- Language filter
- CAP code filter
- SOREM-only mode
- Per-destination multicast with independent codec selection
- Optional local audio output via mc-audio playback API
- TTS voice override and configurable pronunciation replacements
- Cooldown timers for alert/announce tones

### Text cleanup (clean.go)

Alert text is normalized once by `CleanText` (line endings, control
characters including stray BEL, whitespace runs, blank lines), then adapted
per output medium:

- `DisplayText` - one line per line, no blank lines (real-time text, signs)
- `SpeechText` - lines joined into sentences, repeated punctuation
  collapsed, pronunciation replacements applied on word boundaries (TTS)

`AlertText` (real-time text) and `TTSText` / SOREM broadcast text (speech)
both go through these.

### ChannelConfig (delivery.go)

YAML-configurable per-channel settings:

| Field | Purpose |
|-------|---------|
| `SpeakContent` | Include the full description text in TTS, not just the headline |
| `SoremOnly` | Only announce alerts that have SOREM broadcast text |
| `StripComments` | Remove `###`-delimited comments from description before TTS |
| `Multicast` | List of `MulticastDest` (address + payload type) for RTP output |
| `LocalAudio` | Enable local soundcard output via mc-audio |
| `LocalAudioDevice` | OS-specific device name/ID (empty = system default) |
| `LocalAudioRate` | Local playback sample rate (default 48000) |
| `Language` | Language code filter (e.g., `en-CA`) |
| `Voice` | Override the global Piper TTS voice for this channel |
| `CAPCodes` | Geographic filter: only announce alerts matching these codes |
| `TTSReplacements` | Custom pronunciation fixups (default set if omitted) |

### Oscillator (tones.go)

A simple sine-wave oscillator used to synthesize alert tones. Tracks phase
internally and produces int16 or float64 samples at the configured frequency
and sample rate.

## Audio Functions

### GetAudio (content.go)

Searches the alert's resources for broadcast audio. Handles both
`application/x-url` (external link) and `audio/mpeg` (embedded/fetched MP3)
resources. Falls back to TTS via Piper.

### GenerateCAAS (tones.go)

Synthesizes the Canadian Alert Attention Signal (CAAS), an 8-second sequence
of two alternating tone groups, each composed of three sine waves with
amplitude modulation. This is the distinctive emergency alert tone heard on
Canadian broadcasts.

### AnnounceChime (tones.go)

A lighter 3-note chime (660 Hz, 1100 Hz, 880 Hz) with logarithmic decay,
used for non-emergency alerts.

### TTSText (tts.go)

Generates spoken text from alert metadata. For Environment Canada alerts, uses
the format "Environment Canada [type] for [areas]. [headline]." For other
alerts, uses "[category] alert for [areas]. Urgency: [urgency], Severity:
[severity], [headline]." Applies pronunciation fixups for abbreviations and
domain names.

## Issues and Improvement Opportunities

No major issues remaining.
