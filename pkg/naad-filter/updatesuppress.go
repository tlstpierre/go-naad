package naadfilter

import (
	//	log "github.com/sirupsen/logrus"
	"github.com/tlstpierre/go-naad/pkg/naad-xml"
	"sync"
	"time"
)

// Keys that we want compare to determine if a message is mostly the same
type DedupKey struct {
	Identifier string
	Urgency    naadxml.Urgency
	Severity   naadxml.Severity
}

type UpdateSuppressor struct {
	sync.RWMutex
	messageList map[DedupKey]time.Time
	repeat      time.Duration
}

// Factory function
func NewUpdateSuppressor(repeat time.Duration) *UpdateSuppressor {
	return &UpdateSuppressor{
		messageList: make(map[DedupKey]time.Time),
		repeat:      repeat,
	}
}

func updateSuppressKey(msg *naadxml.Alert) DedupKey {
	var key DedupKey
	if len(msg.Info) < 1 {
		return key
	}
	info := msg.Info[0]
	if info == nil {
		return key
	}

	key = DedupKey{
		Identifier: msg.Identifier,
		Urgency:    info.Urgency,
		Severity:   info.Severity,
	}
	return key
}

func (d *UpdateSuppressor) Insert(msg *naadxml.Alert) {
	d.Lock()
	defer d.Unlock()
	if len(msg.Info) < 1 {
		return
	}
	info := msg.Info[0]
	if info == nil {
		return
	}
	key := updateSuppressKey(msg)
	d.messageList[key] = msg.Sent
}

func (d *UpdateSuppressor) Delete(msg *naadxml.Alert) {
	d.Lock()
	defer d.Unlock()
	if len(msg.Info) < 1 {
		return
	}
	info := msg.Info[0]
	if info == nil {
		return
	}
	key := updateSuppressKey(msg)
	delete(d.messageList, key)
}

func (d *UpdateSuppressor) IsDuplicate(msg *naadxml.Alert) bool {
	d.Lock()
	defer d.Unlock()
	if len(msg.Info) < 1 {
		return false
	}
	info := msg.Info[0]
	if info == nil {
		return false
	}
	key := updateSuppressKey(msg)
	sent, found := d.messageList[key]
	if time.Since(sent) > d.repeat {
		return false
	}
	return found
}

func (d *UpdateSuppressor) Clean() {
	d.Lock()
	defer d.Unlock()
	for key, sent := range d.messageList {
		if time.Since(sent) > d.repeat {
			delete(d.messageList, key)
		}
	}
}
