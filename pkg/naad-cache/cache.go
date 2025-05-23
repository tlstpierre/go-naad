package naadcache

import (
	log "github.com/sirupsen/logrus"
	"github.com/tlstpierre/go-naad/pkg/naad-xml"
	"net/http"
	"sort"
	"sync"
	"time"
)

var (
	httpClient = &http.Client{
		Timeout: 10 * time.Second,
	}
)

type Cache struct {
	sync.RWMutex
	alerts         map[string]*naadxml.Alert
	history        map[string]*AlertHistory
	handled        uint64
	httpClient     *http.Client
	archiveServers []string
	ArchiveTime    time.Duration
}

type AlertHistory struct {
	IsOriginal bool
	Updated    bool
	IsUpdate   bool
	LastUpdate time.Time
	Expired    time.Time
	Identifier string
	Current    *naadxml.Alert
	Original   *naadxml.Alert
	UpdatedBy  []string
	// Updates    []string
}

func NewCache() *Cache {
	return &Cache{
		alerts:      make(map[string]*naadxml.Alert, 32),
		history:     make(map[string]*AlertHistory, 32),
		ArchiveTime: 72 * time.Hour,
		httpClient:  httpClient,
	}
}

func (c *Cache) SetArchive(servers []string) {
	c.httpClient = &http.Client{
		Timeout: 10 * time.Second,
	}
	c.archiveServers = servers
}

func (c *Cache) Add(alert *naadxml.Alert) {
	c.Lock()
	defer c.Unlock()
	c.handled++
	c.alerts[alert.Identifier] = alert

	if alert.IsUpdate() {
		//		thisHistory.IsUpdate = true
		for _, reference := range alert.References.References {
			originalID := reference.Identifier
			log.Infof("Updating original alert %s with %s", originalID, alert.Identifier)
			history, hasOriginal := c.history[originalID]
			if !hasOriginal {
				for _, server := range c.archiveServers {
					refAlert, err := reference.Fetch(c.httpClient, server)
					if err == nil {
						c.alerts[refAlert.Identifier] = refAlert
						if !refAlert.IsUpdate() {
							refHistory := newHistory(refAlert, refAlert.Identifier)
							//						refHistory.UpdatedBy = append(refHistory.UpdatedBy, alert.Identifier)
							refHistory.ApplyUpdate(alert)
							c.history[refAlert.Identifier] = refHistory
						} else {
							log.Warnf("Referenced alert %s is also an update??", refAlert.Identifier)
						}
						log.Infof("Fetched %s from archive", refAlert.Identifier)
						break
					} else {
						log.Errorf("Problem fetching previous missed message %s - %v", alert.Identifier, err)
					}
				}
			} else {
				history.ApplyUpdate(alert)
			}
			//			thisHistory.Updates = append(thisHistory.Updates, originalID)
		}
	} else {
		thisHistory := newHistory(alert, alert.Identifier)
		c.history[alert.Identifier] = thisHistory
		thisHistory.IsOriginal = true
	}
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
		Original:   alert,
		UpdatedBy:  make([]string, 0, 2),
		//		Updates:    make([]string, 0, 2),
	}
}

func (h *AlertHistory) ApplyUpdate(alert *naadxml.Alert) {
	h.Updated = true
	h.LastUpdate = alert.Sent
	h.UpdatedBy = append(h.UpdatedBy, alert.Identifier)
	h.Current = alert
	/*
		for i, updateInfo := range alert.Info {
			previous, exits := h.Current[i]
			if !exists {
				continue
			}
		}
	*/
}

/*
func (h *AlertHistory) Push(alert *naadxml.Alert) {
	h.Previous = append(h.Previous, h.Current)
	h.Current = alert
	h.LastUpdate = alert.Sent
}
*/

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

type AlertList []*naadxml.Alert

func (l AlertList) Swap(i, j int) {
	l[i], l[j] = l[j], l[i]
}

func (l AlertList) Len() int {
	return len(l)
}

func (l AlertList) Less(i, j int) bool {
	if l[i].Sent != l[j].Sent {
		return l[i].Sent.After(l[j].Sent)
	}
	return l[i].Identifier < l[j].Identifier
}

func (c *Cache) DumpAlerts() AlertList {
	c.RLock()
	defer c.RUnlock()
	list := make([]*naadxml.Alert, len(c.alerts))
	var index int
	for _, entry := range c.alerts {
		list[index] = entry
		index++
	}
	alertList := AlertList(list)
	sort.Sort(alertList)
	return alertList
}

type CacheHistory []*AlertHistory

func (h CacheHistory) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
}

func (h CacheHistory) Len() int {
	return len(h)
}

func (h CacheHistory) Less(i, j int) bool {
	// If both were updated at the same time, sort by sent date
	if h[i].LastUpdate != h[j].LastUpdate {
		return h[i].LastUpdate.After(h[j].LastUpdate)
	}
	if h[i].Original.Sent != h[j].Original.Sent {
		return h[i].Original.Sent.After(h[j].Original.Sent)
	}
	return h[i].Original.Identifier < h[j].Original.Identifier

}
