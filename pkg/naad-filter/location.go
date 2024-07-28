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

func AlertIsCAPArea(alert *naadxml.Alert, code string) bool {
	for _, info := range alert.Info {
		if IsCAPArea(info.Area, code) {
			return true
		}
	}
	return false
}
