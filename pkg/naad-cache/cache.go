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
	alerts      map[string]*AlertHistory
	handled     uint64
	ArchiveTime time.Duration
}

type AlertHistory struct {
	LastUpdate time.Time
	Expired    time.Time
	Current    *naadxml.Alert
	Previous   []*naadxml.Alert
}

func NewCache() *Cache {
	return &Cache{
		alerts:      make(map[string]*AlertHistory, 32),
		ArchiveTime: 72 * time.Hour,
	}
}

func (c *Cache) Add(alert *naadxml.Alert) {
	c.Lock()
	defer c.Unlock()
	c.handled++
	if alert.IsUpdate() {
		originalID := alert.References.References[0].Identifier
		log.Infof("Updating original alert %s with %s", originalID, alert.Identifier)
		history, hasOriginal := c.alerts[originalID]
		if !hasOriginal {
			c.alerts[originalID] = newHistory(alert)
		} else {
			history.Push(alert)
		}
		return
	}
	log.Infof("Adding alert %s to cache", alert.Identifier)
	c.alerts[alert.Identifier] = newHistory(alert)

}

func (c *Cache) Clean() {
	c.Lock()
	defer c.Unlock()
	for historyID, history := range c.alerts {
		for _, info := range history.Current.Info {
			if info.Expires.After(time.Now()) {
				break
			}
			log.Infof("Alert %s expired at %s", info.Expires)
			history.Expired = info.Expires
		}
		if !history.Expired.IsZero() && time.Since(history.Expired) > c.ArchiveTime {
			delete(c.alerts, historyID)
			log.Infof("Deleted ID %s from history", historyID)
		}
	}
}

func newHistory(alert *naadxml.Alert) *AlertHistory {
	return &AlertHistory{
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
	history := make([]*AlertHistory, len(c.alerts))
	var index int
	for _, entry := range c.alerts {
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
	return h[i].LastUpdate.Before(h[j].LastUpdate)
}
