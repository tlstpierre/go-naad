package naadsocket

import (
	"github.com/tlstpierre/go-naad/pkg/naad-xml"
)

type Receiver interface {
	Connect() error
	Disconnect()
	SetHandler(func(*naadxml.Alert) error)
}
