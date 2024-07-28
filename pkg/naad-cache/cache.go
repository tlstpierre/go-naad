package naadcache

import (
	log "github.com/sirupsen/logrus"
	"github.com/tlstpierre/go-naad/pkg/naad-xml"
	"sort"
	"sync"
	"time"
)

type Cache struct {
	sync.RWMutex
	alerts      map[string]*naadxml.Alert
	history     map[string]*AlertHistory
	handled     uint64
	ArchiveTime time.Duration
}

type AlertHistory struct {
	LastUpdate time.Time
	Expired    time.Time
	Identifier string
	Current    *naadxml.Alert
	Previous   []*naadxml.Alert
}

func NewCache() *Cache {
	return &Cache{
		alerts:      make(map[string]*naadxml.Alert, 32),
		history:     make(map[string]*AlertHistory, 32),
		ArchiveTime: 72 * time.Hour,
	}
}

func (c *Cache) Add(alert *naadxml.Alert) {
	c.Lock()
	defer c.Unlock()
	c.handled++
	c.alerts[alert.Identifier] = alert

	if alert.IsUpdate() {
		originalID := alert.References.References[0].Identifier
		log.Infof("Updating original alert %s with %s", originalID, alert.Identifier)
		history, hasOriginal := c.history[originalID]
		if !hasOriginal {
			c.history[originalID] = newHistory(alert, originalID)

		} else {
			history.Push(alert)
		}
		return
	}
	c.history[alert.Identifier] = newHistory(alert, alert.Identifier)
}

func (c *Cache) Get(id string) (*naadxml.Alert, *AlertHistory) {
	c.Lock()
	defer c.Unlock()
	alert := c.alerts[id]
	history, found := c.history[id]
	if found {
		return alert, history
	}
	log.Infof("Could not find ID %s", id)
	return alert, nil
}

func (c *Cache) Clean() {
	c.Lock()
	defer c.Unlock()
	for historyID, history := range c.history {
		for _, info := range history.Current.Info {
			if info.Expires.After(time.Now()) {
				break
			}
			log.Infof("Alert %s expired at %s", info.Expires)
			history.Expired = info.Expires
		}
		if !history.Expired.IsZero() && time.Since(history.Expired) > c.ArchiveTime {
			delete(c.history, historyID)
			log.Infof("Deleted ID %s from history", historyID)
		}
	}

	for alertID, alert := range c.alerts {
		var current bool
		for _, info := range alert.Info {
			if time.Since(info.Expires) < c.ArchiveTime {
				current = true
			}
		}
		if !current {
			delete(c.alerts, alertID)
		}
	}
}

func newHistory(alert *naadxml.Alert, id string) *AlertHistory {
	return &AlertHistory{
		Identifier: id,
		LastUpdate: alert.Sent,
		Current:    alert,
		Previous:   make([]*naadxml.Alert, 0, 2),
	}
}

func (h *AlertHistory) Push(alert *naadxml.Alert) {
	h.Previous = append(h.Previous, h.Current)
	h.Current = alert
	h.LastUpdate = alert.Sent
}

func (c *Cache) DumpHistory() CacheHistory {
	c.RLock()
	defer c.RUnlock()
	history := make([]*AlertHistory, len(c.history))
	var index int
	for _, entry := range c.history {
		history[index] = entry
		index++
	}
	cacheHistory := CacheHistory(history)
	sort.Sort(cacheHistory)
	return cacheHistory
}

type CacheHistory []*AlertHistory

func (h CacheHistory) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
}

func (h CacheHistory) Len() int {
	return len(h)
}

func (h CacheHistory) Less(i, j int) bool {
	return h[i].LastUpdate.After(h[j].LastUpdate)
}
