package naadaudio

// LocalPlayback is the interface for local audio output devices.
// This mirrors the mc-audio PlaybackDevice interface but is defined
// locally to avoid a hard dependency on the mc-audio/audio package
// (which requires CGO via miniaudio).
type LocalPlayback interface {
	Write(samples []int16) error
	Start() error
	Stop() error
	Close() error
}

// AudioBackend is the interface for creating local playback devices.
type AudioBackend interface {
	OpenPlayback(deviceID string, sampleRate, channels, frameSize int) (LocalPlayback, error)
	Close() error
}
