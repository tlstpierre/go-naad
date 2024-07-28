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

func TTSText(info *naadxml.AlertInfo) string {
	var text string
	if info.ECLayer != nil {
		text = fmt.Sprintf("Environment Canada %s for %s.  %s", info.ECLayer.AlertType, info.ECLayer.AlertCoverage, info.Headline)
	} else {
		text = fmt.Sprintf("Alert for %s.\n%s.%s", info.Area.Description, info.Headline, info.Description)
	}
	text = strings.ReplaceAll(text, "###", ". .")
	//	text = strings.ReplaceAll(text, "/", " or ")
	text = strings.ReplaceAll(text, ".ca", " dot see aay ")
	text = strings.ReplaceAll(text, "Ontario", "ontaireo")
	return text
}
