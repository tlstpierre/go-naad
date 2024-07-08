package naadxml

import ()

type CAP struct {
	MinorChange     bool
	MinorChangeType ChangeType
}

type ChangeType string

const (
	TextChange     ChangeType = "text"
	InfoCorrection            = "correction"
	ResourceChange            = "resource"
	LayerChange               = "layer"
	OtherChange               = "other"
	NoChange                  = "none"
)

func CAPParam(caplayer *CAP, version, parameter, value string) error {
	switch parameter {
	case "MinorChange":
		caplayer.MinorChange = true
		caplayer.MinorChangeType = ChangeType(value)
	}

	return nil
}
