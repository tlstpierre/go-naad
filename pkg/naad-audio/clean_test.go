package naadaudio

import (
	"strings"
	"testing"

	naadxml "github.com/tlstpierre/go-naad/pkg/naad-xml"
	"gitlab.com/timstpierre/mc-audio/pkg/t140"
)

func TestCleanText(t *testing.T) {
	for _, c := range []struct{ in, want string }{
		// Environment Canada description as received live on 2026-11-18
		{"\nNear-zero visibility in fog continues.\n\nFog will dissipate early this afternoon.\n\n",
			"Near-zero visibility in fog continues.\nFog will dissipate early this afternoon."},
		{"a\r\nb\rc d e", "a\nb\nc\nd\ne"},
		{"  lots   of\tspace  here  ", "lots of space here"},
		{"ring\x07 the\x1b bell\xef\xbb\xbf", "ring the bell"},
		{"\n \n\t\n", ""},
		{"Électricité – coupure à Montréal", "Électricité – coupure à Montréal"},
	} {
		if got := CleanText(c.in); got != c.want {
			t.Errorf("CleanText(%q)\n got %q\nwant %q", c.in, got, c.want)
		}
	}
}

func TestReplaceWords(t *testing.T) {
	for _, c := range []struct{ in, from, to, want string }{
		{"25 mm of rain in the community", "mm", "milli meters", "25 milli meters of rain in the community"},
		{"25mm", "mm", "milli meters", "25milli meters"},
		{"winds of 90 km/h", "km/h", "kilometres per hour", "winds of 90 kilometres per hour"},
		{"see weather.gc.ca or canada.ca.", ".ca", " dot see aay ", "see weather.gc dot see aay  or canada dot see aay ."},
		{"www.canada.ca", ".ca", " dot ca", "www.canada dot ca"},
		{"Ontario and Ontarians", "Ontario", "ontaireo", "ontaireo and Ontarians"},
		{"St. Catharines, East.", "St.", "saint", "saint Catharines, East."},
		{"no match", "", "x", "no match"},
	} {
		if got := replaceWords(c.in, c.from, c.to); got != c.want {
			t.Errorf("replaceWords(%q, %q)\n got %q\nwant %q", c.in, c.from, got, c.want)
		}
	}
}

func TestSpeechText(t *testing.T) {
	for _, c := range []struct{ in, want string }{
		{"yellow advisory - fog - in effect\nNear-zero visibility.\n\nFog will dissipate", "yellow advisory - fog - in effect. Near-zero visibility. Fog will dissipate."},
		{"No Action is required..This is a test", "No Action is required. This is a test."},
		{"first ### comment", "first. comment."},
		{"Wait , what ?", "Wait, what?"},
		{"", ""},
	} {
		if got := SpeechText(c.in, DefaultTTSReplacements); got != c.want {
			t.Errorf("SpeechText(%q)\n got %q\nwant %q", c.in, got, c.want)
		}
	}
}

func TestTTSText(t *testing.T) {
	info := &naadxml.AlertInfo{
		Headline:    "yellow advisory - fog - in effect",
		Description: "\nNear-zero visibility in fog continues.\n\nFog will dissipate early this afternoon.\n\n###Issued by the community office.",
		ECLayer:     &naadxml.EC{AlertType: "advisory"},
		Area:        []naadxml.AlertArea{{Description: "Yellowknife Region"}},
	}
	want := "Environment Canada advisory for Yellowknife Region. yellow advisory - fog - in effect. Near-zero visibility in fog continues. Fog will dissipate early this afternoon."
	if got := TTSText(info, true, true, nil); got != want {
		t.Errorf("\n got %q\nwant %q", got, want)
	}
	// No category must not panic.
	if got := TTSText(&naadxml.AlertInfo{Headline: "test"}, false, true, nil); !strings.HasPrefix(got, "Emergency alert for") {
		t.Errorf("no-category alert: %q", got)
	}
}

func TestAlertTextAreas(t *testing.T) {
	info := &naadxml.AlertInfo{
		Headline:    "yellow advisory - fog - in effect",
		Description: "\nNear-zero visibility in fog continues.\n\nFog will dissipate early this afternoon.\n\n",
		Area:        []naadxml.AlertArea{{Description: "Yellowknife Region"}},
	}
	ls := t140.LineSeparator
	want := "yellow advisory - fog - in effect" + ls + "Yellowknife Region" + ls + "Near-zero visibility in fog continues." + ls + "Fog will dissipate early this afternoon." + ls
	if got := AlertText(info, true); got != want {
		t.Errorf("\n got %q\nwant %q", got, want)
	}

	for _, a := range []string{"B", "C", "D", "E", "F", "G"} {
		info.Area = append(info.Area, naadxml.AlertArea{Description: a})
	}
	if got := areaSummary(info, maxDisplayAreas); got != "Yellowknife Region, B, C, D, E and 2 more" {
		t.Errorf("areaSummary = %q", got)
	}

	// A BEL inside alert content must not ring the bell.
	info = &naadxml.AlertInfo{Headline: "bell\x07", Severity: "Minor"}
	if got := AlertText(info, true); strings.Contains(got, t140.BEL) {
		t.Errorf("content BEL leaked: %q", got)
	}
}
