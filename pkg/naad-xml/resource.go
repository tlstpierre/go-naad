package naadxml

import (
	"encoding/base64"
	"encoding/xml"
	log "github.com/sirupsen/logrus"
	// "fmt"
	"io"
	// "strconv"
	// "strings"
)

type Resource struct {
	Description string        `xml:"resourceDesc"`
	MimeType    string        `xml:"mimeType"`
	Size        uint64        `xml:"size"`
	URI         string        `xml:"uri"`
	Content     Base64Content `xml:"derefUri"`
}

type Base64Content []byte

func (c *Base64Content) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	for {
		t, err := d.Token()
		if err == io.EOF {
			break
		}
		//		log.Infof("XML start element is %+v and token is %+v", start, t)
		switch t.(type) {
		case xml.CharData:

			content, err := base64.StdEncoding.DecodeString(string(t.(xml.CharData)))
			if err != nil {
				return err
			}
			*c = Base64Content(content)
		}
	}
	return nil
}

func (r *Resource) Fetch() error {
	if r.MimeType != "appliaction/x-url" {
		log.Infof("Cannot fetch %s - not a URL", r.Description)
		return nil
	}

	// TODO fetch the content here
	return nil
}
