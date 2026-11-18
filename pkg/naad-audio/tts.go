package naadaudio

import (
	"fmt"
	"github.com/tlstpierre/go-naad/pkg/naad-xml"
	"strings"
)

// TTSReplacement rewrites text before it is spoken, usually to fix
// pronunciation. From matches on word boundaries: "mm" replaces the unit in
// "25 mm" or "25mm" but leaves "community" alone (see replaceWords).
type TTSReplacement struct {
	From string `yaml:"from"`
	To   string `yaml:"to"`
}

var DefaultTTSReplacements = []TTSReplacement{
	{"###", ". ."},
	{".ca", " dot see aay "},
	{"St.", "saint"},
	{"Ontario", "ontaireo"},
	{"km/h", "Kilometers per hour"},
	{"km", "Kilometers"},
	{"mm", "milli meters"},
}

// TTSText builds the spoken announcement for an alert without SOREM
// broadcast text: a sentence naming the alert and its areas, the headline
// and (if full) the description, prepared with SpeechText.
func TTSText(info *naadxml.AlertInfo, full, stripcomment bool, replacements []TTSReplacement) string {
	areas := make([]string, 0, len(info.Area))
	for _, v := range info.Area {
		areas = append(areas, v.Description)
	}
	areaList := strings.Join(areas, ". ")

	var lines []string
	if info.ECLayer != nil {
		lines = append(lines, fmt.Sprintf("Environment Canada %s for %s.", info.ECLayer.AlertType, areaList), info.Headline)
	} else {
		category := "Emergency"
		if len(info.Category) > 0 {
			category = string(info.Category[0])
		}
		lines = append(lines, fmt.Sprintf("%s alert for %s. Urgency: %s, Severity: %s.", category, areaList, info.Urgency, info.Severity), info.Headline)
	}
	if full {
		description := info.Description
		if stripcomment {
			description = StripComments(description)
		}
		lines = append(lines, description)
	}
	return SpeechText(strings.Join(lines, "\n"), ttsReplacements(replacements))
}

// ttsReplacements returns the configured replacements, or the defaults if
// none are configured.
func ttsReplacements(configured []TTSReplacement) []TTSReplacement {
	if len(configured) == 0 {
		return DefaultTTSReplacements
	}
	return configured
}
