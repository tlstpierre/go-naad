package naadxml

import (
	"encoding/xml"
	"github.com/google/uuid"
	"time"
)

type Alert struct {
	XMLName    xml.Name    `xml:"alert"`
	Identifier uuid.UUID   `xml:"identifier"`
	Sender     string      `xml:"sender"`
	Sent       time.Time   `xml:"sent"`
	Status     AlertStatus `xml:"status"`
	MsgType    MessageType `xml:"msgType"`
	Scope      AlertScope  `xml:"scope"`
	Code       []string    `xml:"code"`
	Info       []AlertInfo `xml:"info"`
	References References  `xml:"references"`
}

type AlertStatus string

const (
	Actual AlertStatus = "Actual"
)

type MessageType string

const (
	AlertMessage MessageType = "Alert"
)

type AlertScope string

const (
	Public AlertScope = "Public"
)

// TODO type these fields instead of string
type AlertInfo struct {
	Language    string      `xml:"language"`
	Category    string      `xml:"category"`
	Event       string      `xml:"event"`
	Urgency     string      `xml:"urgency"`
	Severity    string      `xml:"severity"`
	Certainty   string      `xml:"certainty"`
	EventCode   Parameter   `xml:"eventCode"`
	Expires     time.Time   `xml:"expires"`
	SenderName  string      `xml:"senderName"`
	Headline    string      `xml:"headline"`
	Description string      `xml:"description"`
	Parameters  []Parameter `xml:"parameter"`
	Area        AlertArea   `xml:"area"`
	Resources   []Resource  `xml:"resource"`
}

type Parameter struct {
	Name  string `xml:"valueName"`
	Value string `xml:"value"`
}

type AlertArea struct {
	Description string      `xml:"areaDesc"`
	Polygon     *GeoPolygon `xml:"polygon"`
	Circle      *GeoCircle  `xml:"circle"`
	Geocode     []Parameter `xml:"geocode"`
}
