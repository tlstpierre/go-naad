package naadxml

import "strings"

// Whether an alert is in effect over a given area is carried in two places
// in ECCC alerts, both read here from the raw parameters and geocodes so
// they work on alerts that haven't been through ProcessAlert (such as
// references fetched from the archive):
//
//   - Each <info> block has an EC-MSC-SMC Alert_Location_Status parameter:
//     "active" for the areas now under the alert, or "ended" (and, per the
//     ECCC release notes, cancelled or transitioned out) for areas it no
//     longer covers. ECCC splits a message into one block per status.
//   - Since August 2026, tornado and severe thunderstorm warnings add an
//     <area> with a forecaster-drawn threat polygon whose EC-MSC-SMC DLC
//     geocode is "issued", "continued", "ended" or "cancelled".
//
// Any issuer's block with responseType AllClear is a stand-down too.

// LocationStatus is the block's EC Alert_Location_Status ("active",
// "ended", ...), lower case, or "" if it has none.
func (i *AlertInfo) LocationStatus() string {
	for _, p := range i.Parameters {
		if strings.HasPrefix(p.Name, "layer:EC-MSC-SMC:") && strings.HasSuffix(p.Name, ":Alert_Location_Status") {
			return strings.ToLower(strings.TrimSpace(p.Value))
		}
	}
	return ""
}

// InEffect reports whether the block describes areas where the alert is
// in effect, as opposed to a stand-down: an EC location status other than
// active, or a responseType of AllClear.
func (i *AlertInfo) InEffect() bool {
	for _, r := range i.Response {
		if strings.EqualFold(string(r), "AllClear") {
			return false
		}
	}
	status := i.LocationStatus()
	return status == "" || status == "active"
}

// ThreatStatus is the area's EC DLC geocode ("issued", "continued",
// "ended", "cancelled"), lower case, or "" for an ordinary area.
func (a AlertArea) ThreatStatus() string {
	for _, g := range a.Geocode {
		if strings.HasPrefix(g.Name, "layer:EC-MSC-SMC:") && strings.HasSuffix(g.Name, ":DLC") {
			return strings.ToLower(strings.TrimSpace(g.Value))
		}
	}
	return ""
}

// InEffect reports whether the area is under the alert: an ordinary area,
// or a drawn threat area that is newly issued or continued. Ended and
// cancelled threat areas are kept in the alert (for maps) but are not.
func (a AlertArea) InEffect() bool {
	switch a.ThreatStatus() {
	case "", "issued", "continued":
		return true
	}
	return false
}

// RefersBack reports whether the message amends or cancels earlier ones:
// an Update or Cancel with references.
func (a *Alert) RefersBack() bool {
	return (a.MsgType == AlertUpdate || a.MsgType == AlertCancel) && len(a.References.References) > 0
}
