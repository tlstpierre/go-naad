module github.com/tlstpierre/go-naad

go 1.25.0

replace gitlab.com/timstpierre/mc-audio => ../mc-audio

require (
	github.com/davecgh/go-spew v1.1.1
	github.com/golang/geo v0.0.0-20230421003525-6adc56603217
	github.com/gorilla/mux v1.8.1
	github.com/hajimehoshi/go-mp3 v0.3.4
	github.com/shenjinti/go722 v0.0.0-20241018003611-642cc8091058
	github.com/sirupsen/logrus v1.9.3
	gitlab.com/timstpierre/mc-audio v0.0.0-20260412202416-b5ae612a39f4
	gopkg.in/yaml.v2 v2.4.0
)

require (
	github.com/gen2brain/malgo v0.11.24 // indirect
	github.com/go-audio/audio v1.0.0 // indirect
	github.com/go-audio/riff v1.0.0 // indirect
	github.com/go-audio/wav v1.1.0 // indirect
	golang.org/x/sys v0.33.0 // indirect
)
