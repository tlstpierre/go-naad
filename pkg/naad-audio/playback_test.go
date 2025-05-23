package naadaudio

import (
	"encoding/xml"
	"flag"
	"github.com/tlstpierre/go-naad/pkg/naad-xml"
	"time"
	//	"github.com/davecgh/go-spew/spew"
	"bytes"
	"context"
	"encoding/binary"
	"github.com/shenjinti/go722"
	log "github.com/sirupsen/logrus"
	"github.com/tlstpierre/mc-audio/pkg/g711"
	"github.com/tlstpierre/mc-audio/pkg/mc-transmit"
	"github.com/tlstpierre/mc-audio/pkg/piper-tts"
	"os"
	"sync"
	"testing"
)

var (
	testData    = flag.String("testdata", "../../testdata/", "Path to Pelmorex sample XML")
	testAddress = flag.String("testaddress", "[FF05:0:0:0:0:0:1:1010]:5006", "Multicast address for testing audio")
	LogLevel    = flag.String("loglevel", "info", "The logging verbosity")
	tx          *TransmitChannel
	alertChan   = make(chan *naadxml.Alert, 1)
)

func TestMain(m *testing.M) {
	flag.Parse()
	lvl, _ := log.ParseLevel(*LogLevel)
	log.SetLevel(lvl)

	var err error
	wg := new(sync.WaitGroup)
	config := ChannelConfig{
		SpeakContent:  true,
		StripComments: true,
		G722:          true,
		Addresses:     []string{*testAddress},
		Language:      "en-CA",
	}
	tx, err = NewTransmitter(alertChan, config, context.TODO(), wg)
	if err != nil {
		log.Fatal(err)
	}
	tx.AddTTS(pipertts.PiperConfig{
		Samplerate: 16000,
		Command:    "/home/tim/piper/piper",
		VoicePath:  "/home/tim/piper/",
		Voice:      "en_GB-alan-low",
	})
	tx.AddMulticast([]string{
		*testAddress,
	})

	exitVal := m.Run()
	log.Infof("Leaving TestMain() (Session closing) (Exit Status: %d)", exitVal)
}

func TestSample2(t *testing.T) {
	data, err := os.ReadFile(*testData + "Sample2_CAPCP_with_Embedded_Large_Audio_File.xml")
	if err != nil {
		log.Fatalf("Could not open test XML file - %v", err)
	}
	testAlert := new(naadxml.Alert)
	err = xml.Unmarshal(data, testAlert)
	if err != nil {
		t.Fail()
		t.Errorf("Problem decoding sample 2 - %v", err)
		return
	}
	testAlert.GetLayers()
	testAlert.ProcessAlert()
	alertChan <- testAlert
	time.Sleep(31 * time.Second)
	// spew.Dump(testAlert)
}

func TestSample4(t *testing.T) {
	data, err := os.ReadFile(*testData + "Sample4_CAPCP_with_External_Large_Audio_File.XML")
	if err != nil {
		log.Fatalf("Could not open test XML file - %v", err)
	}
	testAlert := new(naadxml.Alert)
	err = xml.Unmarshal(data, testAlert)
	if err != nil {
		t.Fail()
		t.Errorf("Problem decoding sample 2 - %v", err)
		return
	}
	testAlert.GetLayers()
	testAlert.ProcessAlert()
	alertChan <- testAlert
	time.Sleep(31 * time.Second)
	// spew.Dump(testAlert)
}

func TestSample10(t *testing.T) {
	data, err := os.ReadFile(*testData + "Sample10_CAPCP_with_TTS.XML")
	if err != nil {
		log.Fatalf("Could not open test XML file - %v", err)
	}
	testAlert := new(naadxml.Alert)
	err = xml.Unmarshal(data, testAlert)
	if err != nil {
		t.Fail()
		t.Errorf("Problem decoding sample 2 - %v", err)
		return
	}
	testAlert.GetLayers()
	testAlert.ProcessAlert()
	alertChan <- testAlert
	time.Sleep(31 * time.Second)
	// spew.Dump(testAlert)
}

func TestSample5(t *testing.T) {
	data, err := os.ReadFile(*testData + "Sample5_CAPCP_with_Multiple_External_Audio_File_links.XML")
	if err != nil {
		log.Fatalf("Could not open test XML file - %v", err)
	}
	testAlert := new(naadxml.Alert)
	err = xml.Unmarshal(data, testAlert)
	if err != nil {
		t.Fail()
		t.Errorf("Problem decoding sample 2 - %v", err)
		return
	}
	testAlert.GetLayers()
	testAlert.ProcessAlert()
	alertChan <- testAlert
	time.Sleep(31 * time.Second)
	// spew.Dump(testAlert)
}

func TestTone(t *testing.T) {
	mctx, txerr := rtptransmit.NewSender([]string{*testAddress}, 0, 8000, 20)
	if txerr != nil {
		t.Logf("problem creating multicast transmitter - %v", txerr)
		t.Fail()
		return
	}
	osc := NewOscillator(7271.96, 16000)
	lowrate := make([]int16, 8000)
	osc.WriteSamples(lowrate)
	encoded := g711.ULawEncode(lowrate)
	mctx.SendBuffer(bytes.NewBuffer(encoded), 160, context.TODO())
	mctx.Stop()
}

func TestChime(t *testing.T) {
	mctx, txerr := rtptransmit.NewSender([]string{*testAddress}, 0, 8000, 20)
	if txerr != nil {
		t.Logf("problem creating multicast transmitter - %v", txerr)
		t.Fail()
		return
	}
	lowrate := Chime(880, 5, 8000)
	encoded := g711.ULawEncode(lowrate)
	mctx.SendBuffer(bytes.NewBuffer(encoded), 160, context.TODO())
	mctx.Stop()
}

func TestAnnounceChimeG722(t *testing.T) {
	mctx, txerr := rtptransmit.NewSender([]string{*testAddress}, 9, 16000, 20)
	if txerr != nil {
		t.Logf("problem creating multicast transmitter - %v", txerr)
		t.Fail()
		return
	}
	lowrate := AnnounceChime(16000)
	encoder := go722.NewG722Encoder(go722.Rate64000, go722.G722_DEFAULT)
	g722Buf := new(bytes.Buffer)
	err := binary.Write(g722Buf, binary.LittleEndian, lowrate)
	if err != nil {
		log.Errorf("Problem converting audio to little-endian for g722 encoder - %v", err)
		t.Fail()
		t.Log(err)
	}
	encoded := encoder.Encode(g722Buf.Bytes())
	mctx.SendBuffer(bytes.NewBuffer(encoded), 160, context.TODO())
	mctx.Stop()
}

func TestAttention(t *testing.T) {
	mctx, txerr := rtptransmit.NewSender([]string{*testAddress}, 0, 8000, 20)
	if txerr != nil {
		t.Logf("problem creating multicast transmitter - %v", txerr)
		t.Fail()
		return
	}
	lowrate := GenerateCAAS(8000)
	encoded := g711.ULawEncode(lowrate)
	mctx.SendBuffer(bytes.NewBuffer(encoded), 160, context.TODO())
	mctx.Stop()
}
