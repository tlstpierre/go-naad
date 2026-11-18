package naadfilter

import (
	"github.com/tlstpierre/go-naad/pkg/naad-xml"
	//	"strconv"
	"strings"
)

func IsCAPArea(area naadxml.AlertArea, codematch string) bool {
	//	codeStr := strconv.FormatInt(int64(code), 10)
	for _, code := range area.Geocode {
		if !strings.HasPrefix(code.Name, "profile:CAP-CP:Location") {
			continue
		}
		if strings.HasPrefix(code.Value, codematch) {
			return true
		}
	}
	return false
}

// AlertIsCAPArea reports whether the alert is in effect for a location
// whose CAP-CP code starts with code. Like IsPlaceInAlert, it skips
// stand-down blocks and ended or cancelled threat areas.
func AlertIsCAPArea(alert *naadxml.Alert, code string) bool {
	for _, info := range alert.Info {
		if !info.InEffect() {
			continue
		}
		for _, area := range info.Area {
			if !area.InEffect() {
				continue
			}
			if IsCAPArea(area, code) {
				return true
			}
		}
	}
	return false
}
