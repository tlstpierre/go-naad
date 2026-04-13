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
- Codec selection (G.722 or mu-law)
- Multicast address list
- TTS voice override
- Cooldown timers for alert/announce tones

### ChannelConfig (delivery.go)

YAML-configurable per-channel settings:

| Field | Purpose |
|-------|---------|
| `SpeakContent` | Include the full description text in TTS, not just the headline |
| `SoremOnly` | Only announce alerts that have SOREM broadcast text |
| `StripComments` | Remove `###`-delimited comments from description before TTS |
| `G722` | Use G.722 codec (true) or mu-law/G.711 (false) |
| `Addresses` | RTP multicast destination addresses |
| `Language` | Language code filter (e.g., `en-CA`) |
| `Voice` | Override the global Piper TTS voice for this channel |
| `CAPCodes` | Geographic filter: only announce alerts matching these codes |

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

- **Local audio output not implemented** (delivery.go). The `LocalAudio`
  flag is checked but the body is a TODO comment. No local speaker/soundcard
  output is available.

- **TTS pronunciation replacements are fragile** (tts.go). The replacements
  are applied globally and can cause unintended substitutions (e.g.,
  "St." -> "saint" would transform "St. Louis" correctly but also "St." at
  end of a sentence). The "Ontario" -> "ontaireo" replacement is a
  pronunciation hack that ideally belongs in the TTS engine's lexicon.

- **Unused global `speaker` variable** (tts.go). A package-level `speaker`
  variable is declared but never used; each `TransmitChannel` has its own
  `speaker` field.
