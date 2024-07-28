package naadaudio

import (
	//	log "github.com/sirupsen/logrus"
	"math"
)

type Oscillator struct {
	phase      float64
	sampleStep float64
}

func NewOscillator(frequency float64, rate uint32) *Oscillator {
	return &Oscillator{
		sampleStep: 2 * math.Pi / float64(rate) * frequency,
	}
}

func (o *Oscillator) Sample() int16 {
	output := int16(math.Sin(o.phase) * 0x7fff)
	o.phase += o.sampleStep
	return output
}

func (o *Oscillator) FloatSample() float64 {
	output := math.Sin(o.phase)
	o.phase += o.sampleStep
	return output
}

func (o *Oscillator) WriteSamples(buffer []int16) {
	for i, _ := range buffer {
		buffer[i] = o.Sample()
	}
}

func GenerateCAAS(rate uint32) []int16 {
	// 932.33 Hz, 1046.5 Hz and 3135.96, modulated at 7271.96 Hz
	osc1a := NewOscillator(932.33, rate)
	osc1b := NewOscillator(1046.5, rate)
	osc1c := NewOscillator(3135.96, rate)
	osc1m := NewOscillator(7271.96, rate)

	//440Hz, 659.26 Hz and 3135.96 Hz, modulated at 1099.26 Hz.
	osc2a := NewOscillator(440, rate)
	osc2b := NewOscillator(659.26, rate)
	osc2c := NewOscillator(3135.97, rate)
	osc2m := NewOscillator(1099.26, rate)

	group1 := make([]int16, rate/2)
	for i, _ := range group1 {
		var sample float64
		sample += osc1a.FloatSample()
		sample += osc1b.FloatSample()
		sample += osc1c.FloatSample()
		if rate >= 16000 {
			group1[i] = int16(sample / 6 * (osc1m.FloatSample() + 1) * 0x6fff)
		} else {
			group1[i] = int16(sample / 6 * 0x6fff) //(osc1m.FloatSample() / 3))
		}
	}
	group2 := make([]int16, rate/2)
	for i, _ := range group2 {
		var sample float64
		sample += osc2a.FloatSample()
		sample += osc2b.FloatSample()
		sample += osc2c.FloatSample()
		//		group2[i] = int16(sample / 4)
		group2[i] = int16(sample / 6 * (osc2m.FloatSample() + 1) * 0x6fff)
	}
	output := make([]int16, 0, rate*8)
	for i := 0; i < 8; i++ {
		output = append(output, group1...)
		output = append(output, group2...)
	}
	return output
}

func Chime(frequency float64, decay, rate uint32) []int16 {
	osc := NewOscillator(frequency, rate)
	buf := make([]int16, decay*rate)
	sampleCountFloat := float64(decay * rate)
	//	sampleCount := decay * rate
	for i, _ := range buf {
		var fade float64
		//		if i < int(sampleCount/10) {
		//		fade = 1
		//	} else {
		fade = math.Log10(2 * sampleCountFloat / float64(i))
		//		}

		buf[i] = int16(osc.FloatSample() * fade * 0x1fff)
		//}

	}
	return buf
}

func AnnounceChime(rate uint32) []int16 {
	output := make([]int16, rate*5)
	chime1 := Chime(660, 3, rate)
	chime2 := Chime(1100, 3, rate)
	chime3 := Chime(880, 3, rate)
	offset1 := 2 * int(rate) / 3
	offset2 := 4 * int(rate) / 3
	for i, v := range chime1 {
		output[i] += v
	}

	for i, v := range chime2 {
		output[i+offset1] += v
	}
	for i, v := range chime3 {
		output[i+offset2] += v
	}

	return output
}
