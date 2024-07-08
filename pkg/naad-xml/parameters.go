package naadxml

import (
	"strings"
)

func (i *AlertInfo) ProcessParams() error {
	for _, param := range i.Parameters {
		if strings.HasPrefix(param.Name, "layer:") {
			i.ProcessLayerParam(param)
		} else if strings.HasPrefix(param.Name, "profile:") {
			i.ProcessProfileParam(param)
		}
	}
	return nil
}
