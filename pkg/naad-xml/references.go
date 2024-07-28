package naadxml

import (
	"encoding/xml"
	"fmt"
	//	"github.com/google/uuid"
	"io"
	"net/http"
	"strings"
	"time"
)

type References struct {
	References []Reference
}

type Reference struct {
	Sender     string
	Identifier string
	Sent       time.Time
}

func (a *Alert) IsUpdate() bool {
	if a.MsgType == AlertUpdate && len(a.References.References) > 0 {
		return true
	}
	return false
}

func (r *References) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	for {
		t, err := d.Token()
		if err == io.EOF {
			break
		}
		//		log.Infof("XML start element is %+v and token is %+v", start, t)
		switch t.(type) {
		case xml.CharData:
			items := strings.Split(string(t.(xml.CharData)), " ")
			r.References = make([]Reference, len(items))
			for i, item := range items {
				elements := strings.Split(item, ",")
				if len(elements) != 3 {
					return fmt.Errorf("Parse error separating elements in %s - expecting 3 elements", item)
				}
				reference := Reference{
					Sender:     elements[0],
					Identifier: elements[1],
				}
				var err error
				reference.Sent, err = time.Parse(time.RFC3339, elements[2])
				if err != nil {
					return fmt.Errorf("Problem parsing time %s for reference %s - %v", elements[2], elements[1], err)
				}
				r.References[i] = reference
			}
		}
	}
	return nil
}

func (r Reference) URL(host string) string {
	year, month, day := r.Sent.Date()
	dateFormat := r.Sent.Format(time.RFC3339) //"2006_01_02T15_04_05-07_00")
	dateFormat = strings.ReplaceAll(dateFormat, "Z", "_00_00")
	return fmt.Sprintf("http://%s/%d-%02d-%02d/%sI%s.xml", host, year, month, day, addSubstitution(dateFormat), addSubstitution(r.Identifier))
}

func addSubstitution(input string) string {
	input = strings.ReplaceAll(input, "-", "_")
	input = strings.ReplaceAll(input, "+", "p")
	input = strings.ReplaceAll(input, ":", "_")
	return input
}

func (r Reference) Fetch(client *http.Client, server string) (*Alert, error) {
	url := r.URL(server)
	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("Could not retrieve message ref %s - %v", r.Identifier, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("Error from archive server fetching ref %s - %s", r.Identifier, resp.Status)
	}
	decoder := xml.NewDecoder(resp.Body)
	alert := new(Alert)
	err = decoder.Decode(alert)
	if err != nil {
		return nil, fmt.Errorf("Could not decode retrieved reference %s from %s - %v", r.Identifier, url, err)
	}
	return alert, nil
}
