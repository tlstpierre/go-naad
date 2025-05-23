package naadaudio

import (
	"fmt"
	"github.com/tlstpierre/go-naad/pkg/naad-xml"
	"github.com/tlstpierre/mc-audio/pkg/piper-tts"
	"strings"
)

var (
	speaker *pipertts.Speaker
)

func TTSText(info *naadxml.AlertInfo, full, stripcomment bool) string {
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
	text = strings.ReplaceAll(text, "###", ". .")
	//	text = strings.ReplaceAll(text, "/", " or ")
	text = strings.ReplaceAll(text, ".ca", " dot see aay ")
	text = strings.ReplaceAll(text, "St.", "saint")
	text = strings.ReplaceAll(text, "Ontario", "ontaireo")
	text = strings.ReplaceAll(text, "km/h", "Kilometers per hour")
	text = strings.ReplaceAll(text, "km", "Kilometers")
	text = strings.ReplaceAll(text, "mm", "milli meters")
	return text
}
