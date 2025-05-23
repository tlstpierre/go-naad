package naadxml

import (
	"encoding/xml"
	"time"
)

type Alert struct {
	XMLName    xml.Name     `xml:"alert"`
	Receiver   string       `xml:"-"`
	Identifier string       `xml:"identifier"`
	Sender     string       `xml:"sender"`
	Sent       time.Time    `xml:"sent"`
	Status     AlertStatus  `xml:"status"`
	MsgType    MessageType  `xml:"msgType"`
	Scope      AlertScope   `xml:"scope"`
	Code       []string     `xml:"code"`
	Note       string       `xml:"note"`
	Info       []*AlertInfo `xml:"info"`
	References References   `xml:"references"`
	Layers     []Layer      `xml:"-"`
	Profiles   []Profile    `xml:"-"`
}

type AlertStatus string

const (
	Actual   AlertStatus = "Actual"
	Exercise             = "Exercise"
	System               = "System"
	Test                 = "Test"
	Draft                = "Draft"
)

type MessageType string

const (
	AlertMessage MessageType = "Alert"
	AlertUpdate              = "Update"
	AlertCancel              = "Cancel"
	AlertAck                 = "Ack"
	AlertError               = "Error"
)

type AlertScope string

const (
	Public          AlertScope = "Public"
	RestrictedScope            = "Restricted"
	PrivateScope               = "Private"
)

// Describes an emergency alert
type AlertInfo struct {
	Language    string         `xml:"language"`
	Category    []Category     `xml:"category"`
	Event       string         `xml:"event"`
	Response    []ResponseType `xml:"responseType"`
	Urgency     Urgency        `xml:"urgency"`
	Severity    Severity       `xml:"severity"`
	Certainty   Certainty      `xml:"certainty"`
	EventCode   Parameter      `xml:"eventCode"`
	Effective   time.Time      `xml:"effective"`
	Onset       time.Time      `xml:"onset"`
	Expires     time.Time      `xml:"expires"`
	SenderName  string         `xml:"senderName"`
	Headline    string         `xml:"headline"`
	Description string         `xml:"description"`
	Instruction string         `xml:"instruction"`
	Parameters  []Parameter    `xml:"parameter"`
	Area        []AlertArea    `xml:"area"`
	Resources   []Resource     `xml:"resource"`
	SoremLayer  *Sorem         `xml:"-"`
	ECLayer     *EC            `xml:"-"`
	CAPLayer    *CAP           `xml:"-"`
}

type Urgency string

const (
	Immediate      Urgency = "Immediate" // Responsive action should be taken immediately
	Expected               = "Expected"  // Responsive action should be taken soon, within the next hour
	Future                 = "Future"    // Responsive action should be taken in the near future
	Past                   = "Past"      // Responsive action is no longer required
	UnknownUrgency         = "Unknown"   // Urgency not known
)

type Severity string

const (
	Extreme         Severity = "Extreme"  // Extraordinary threat to life
	Severe                   = "Severe"   // Significant threat to life
	Moderate                 = "Moderate" // Possible threat to life
	Minor                    = "Minor"    // Minimal to no known threat to life
	UnknownSeverity          = "Unknown"  // Severity unknown
)

type Certainty string

const (
	Observed         Certainty = "Observed" // Determined to have occured or to be ongoing
	Likely                     = "Likely"   // Possibility > than 50%
	Possible                   = "Possible" // Possible but not likely - < 50%
	Unlikely                   = "Unlikely" // Not expected to occur
	UnknownCertainty           = "Unknown"  // Certainty not known
)

type Category string

const (
	Geo            Category = "Geo"
	Met                     = "Met"
	Safety                  = "Safety"
	Security                = "Security"
	Rescue                  = "Rescue"
	Fire                    = "Fire"
	Environment             = "Env"
	Transport               = "Transport"
	Infrastructure          = "Infra"
	CBRNE                   = "CBRNE"
	Other                   = "Other"
)

type ResponseType string

const (
	Shelter  ResponseType = "Shelter"
	Evacuate              = "Evacuate"
	Prepare               = "Prepare"
	Execute               = "Execute"
	Avoid                 = "Avoid"
	Monitor               = "Monitor"
	Assess                = "Assess"
	AllClear              = "AllClear"
	None                  = "None"
)

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
