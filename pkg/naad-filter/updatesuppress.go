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
	// Pick a deterministic info block regardless of arrival order.
	// Use the first block sorted by language so the key is stable
	// even if info blocks arrive in a different order.
	info := msg.Info[0]
	for _, candidate := range msg.Info {
		if candidate != nil && (info == nil || candidate.Language < info.Language) {
			info = candidate
		}
	}
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
	key := updateSuppressKey(msg)
	if key.Identifier == "" {
		return
	}
	d.Lock()
	defer d.Unlock()
	d.messageList[key] = msg.Sent
}

func (d *UpdateSuppressor) Delete(msg *naadxml.Alert) {
	key := updateSuppressKey(msg)
	if key.Identifier == "" {
		return
	}
	d.Lock()
	defer d.Unlock()
	delete(d.messageList, key)
}

func (d *UpdateSuppressor) IsDuplicate(msg *naadxml.Alert) bool {
	d.RLock()
	defer d.RUnlock()
	key := updateSuppressKey(msg)
	if key.Identifier == "" {
		return false
	}
	sent, found := d.messageList[key]
	if !found || time.Since(sent) > d.repeat {
		return false
	}
	return true
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
