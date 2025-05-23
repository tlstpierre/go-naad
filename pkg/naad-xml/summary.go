package naadxml

import (
	"strings"
	"time"
)

type AlertSummary struct {
	HasSorem   bool
	HasEC      bool
	HasCAP     bool
	Expired    bool
	Identifier string
	Sender     string
	Sent       time.Time
	Status     AlertStatus
	MsgType    MessageType
	Scope      AlertScope
	Info       map[string]InfoSummary
}

type InfoSummary struct {
	Language    string
	Category    []Category
	Event       string
	Urgency     Urgency
	Severity    Severity
	Certainty   Certainty
	EventCode   Parameter
	Expires     time.Time
	SenderName  string
	Headline    string
	Description string
	Instruction string
	Area        string
	SoremLayer  *Sorem
	ECLayer     *EC
	CAPLayer    *CAP
}

func (a Alert) Summary() AlertSummary {
	summary := AlertSummary{
		Identifier: a.Identifier,
		Sender:     a.Sender,
		Sent:       a.Sent,
		Status:     a.Status,
		MsgType:    a.MsgType,
		Scope:      a.Scope,
		Info:       make(map[string]InfoSummary, len(a.Info)),
		Expired:    true,
	}
	for _, info := range a.Info {
		summary.Info[info.Language] = info.Summary()
		if info.SoremLayer != nil {
			summary.HasSorem = true
		}
		if info.ECLayer != nil {
			summary.HasEC = true
		}
		if info.CAPLayer != nil {
			summary.HasCAP = true
		}
		if info.Expires.After(time.Now()) {
			summary.Expired = false
		}
	}
	return summary
}

func (i AlertInfo) Summary() InfoSummary {
	summary := InfoSummary{
		Language:    i.Language,
		Category:    i.Category,
		Event:       i.Event,
		Urgency:     i.Urgency,
		Severity:    i.Severity,
		Certainty:   i.Certainty,
		EventCode:   i.EventCode,
		Expires:     i.Expires,
		SenderName:  i.SenderName,
		Headline:    i.Headline,
		Description: i.Description,
		Instruction: i.Instruction,
		SoremLayer:  i.SoremLayer,
		ECLayer:     i.ECLayer,
		CAPLayer:    i.CAPLayer,
	}
	areas := make([]string, len(i.Area))
	for index, v := range i.Area {
		areas[index] = v.Description
	}
	summary.Area = strings.Join(areas, ", ")
	return summary
}
