package naadxml

import (
	"encoding/base64"
	"encoding/xml"
	"fmt"
	log "github.com/sirupsen/logrus"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	// "strconv"
	"strings"
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
	if r.MimeType != "application/x-url" {
		log.Infof("Cannot fetch %s - not a URL", r.Description)
		return nil
	}

	// TODO fetch the content here
	resp, err := http.Get(r.URI)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("Got response %d trying to fetch %s", resp.StatusCode, r.URI)
	}
	r.Content, err = io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("Problem reading content from %s - %v", r.URI, err)
	}
	r.MimeType = resp.Header.Get("ContentType")
	r.Size = uint64(len(r.Content))
	log.Infof("URI is %s", r.URI)
	r.URI = strings.TrimPrefix(r.URI, "http:/")
	r.URI = strings.TrimPrefix(r.URI, "https:/")
	r.URI = filepath.Base(r.URI)
	log.Infof("URI is %s", r.URI)
	r.URI = "test.mp3"
	log.Infof("URI is %s", r.URI)

	return nil
}

func (r *Resource) SaveFile(path string) error {
	filename, err := url.PathUnescape(r.URI)
	if err != nil {
		return err
	}
	filename = fmt.Sprintf("%s/%s", path, filename)
	log.Infof("Saving resource %s as file %s", r.Description, filename)
	var f *os.File
	f, err = os.OpenFile(filename, os.O_RDWR|os.O_CREATE, 0644)
	defer f.Close()
	var count int
	count, err = f.Write(r.Content)
	if err != nil {
		return err
	}
	if count != int(r.Size) {
		log.Errorf("Wrote %d bytes to file %s but resource was %d bytes", count, r.URI, r.Size)
	}
	return nil
}
