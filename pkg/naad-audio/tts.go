package naadaudio

import (
	"fmt"
	"github.com/tlstpierre/go-naad/pkg/naad-xml"
	"strings"
)

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

func TTSText(info *naadxml.AlertInfo, full, stripcomment bool, replacements []TTSReplacement) string {
	var text string

	areas := make([]string, len(info.Area))
	for i, v := range info.Area {
		areas[i] = v.Description
	}
	areaList := strings.Join(areas, ". ")

	if info.ECLayer != nil {
		text = fmt.Sprintf("Environment Canada %s for %s.  %s. ", info.ECLayer.AlertType, areaList, info.Headline)
	} else {
		text = fmt.Sprintf("%s alert for %s. Urgency: %s, Severity: %s, %s.", info.Category[0], areaList, info.Urgency, info.Severity, info.Headline)
	}
	if full {
		description := info.Description
		if stripcomment {
			descParts := strings.Split(description, "###")
			description = descParts[0]
		}
		text += description
	}
	if len(replacements) == 0 {
		replacements = DefaultTTSReplacements
	}
	for _, r := range replacements {
		text = strings.ReplaceAll(text, r.From, r.To)
	}
	return text
}
