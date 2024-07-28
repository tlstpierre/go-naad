module github.com/tlstpierre/go-naad

go 1.22.5

replace github.com/tlstpierre/mc-audio => ../mc-audio

require (
	github.com/davecgh/go-spew v1.1.1
	github.com/golang/geo v0.0.0-20230421003525-6adc56603217
	github.com/gorilla/mux v1.8.1
	github.com/hajimehoshi/go-mp3 v0.3.4
	github.com/sirupsen/logrus v1.9.3
	github.com/tlstpierre/mc-audio v0.0.0-00010101000000-000000000000
	gopkg.in/yaml.v2 v2.4.0
)

require (
	github.com/go-audio/audio v1.0.0 // indirect
	github.com/go-audio/riff v1.0.0 // indirect
	github.com/go-audio/wav v1.1.0 // indirect
	golang.org/x/sys v0.0.0-20220715151400-c0bba94af5f8 // indirect
)
