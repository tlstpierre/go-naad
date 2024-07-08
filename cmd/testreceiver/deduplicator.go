package main

import (
	log "github.com/sirupsen/logrus"
	"sync"
	"time"
)

type DeDuplicate struct {
	sync.RWMutex
	messageList         map[string]time.Time
	retrievedReferences map[string]time.Time
}

func NewDeDuplicate() *DeDuplicate {
	return &DeDuplicate{
		messageList:         make(map[string]time.Time, 16),
		retrievedReferences: make(map[string]time.Time, 8),
	}
}

func (d *DeDuplicate) HasMessage(msg string) bool {
	d.RLock()
	defer d.RUnlock()
	_, found := d.messageList[msg]
	return found
}

func (d *DeDuplicate) MarkMessage(msg string) {
	d.Lock()
	defer d.Unlock()
	d.messageList[msg] = time.Now()
}

func (d *DeDuplicate) HasReference(ref string) bool {
	d.RLock()
	defer d.RUnlock()
	_, found := d.retrievedReferences[ref]
	return found
}

func (d *DeDuplicate) MarkReference(ref string) {
	d.Lock()
	defer d.Unlock()
	d.retrievedReferences[ref] = time.Now()
}

func (d *DeDuplicate) CleanOld(age time.Duration) {
	d.Lock()
	defer d.Unlock()
	var msgCount int
	var refCount int
	for msg, timestamp := range d.messageList {
		if time.Since(timestamp) > age {
			delete(d.messageList, msg)
			msgCount++
		}
	}
	for ref, timestamp := range d.retrievedReferences {
		if time.Since(timestamp) > age {
			delete(d.retrievedReferences, ref)
			refCount++
		}
	}
	log.Infof("Deleted %d messages markers and %d ref markers older than %s", msgCount, refCount, age)
}
