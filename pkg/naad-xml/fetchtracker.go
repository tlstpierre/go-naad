package naadxml

import (
	"context"
	"fmt"
	log "github.com/sirupsen/logrus"
	"net/http"
	"sync"
	"time"
)

type fetchEntry struct {
	done  chan struct{}
	alert *Alert
	err   error
	ts    time.Time
}

// FetchTracker deduplicates concurrent archive fetches for the same alert
// identifier. The first caller for a given identifier performs the HTTP
// request; subsequent callers block until that result is available.
type FetchTracker struct {
	mu             sync.Mutex
	inflight       map[string]*fetchEntry
	httpClient     *http.Client
	archiveServers []string
	fetchTimeout   time.Duration
	entryTTL       time.Duration
}

func NewFetchTracker(client *http.Client, servers []string) *FetchTracker {
	return &FetchTracker{
		inflight:       make(map[string]*fetchEntry, 16),
		httpClient:     client,
		archiveServers: servers,
		fetchTimeout:   30 * time.Second,
		entryTTL:       2 * time.Minute,
	}
}

// Fetch returns the alert for the given reference. If another goroutine is
// already fetching the same identifier, Fetch blocks until that result is
// ready or ctx is cancelled. The first caller drives the actual HTTP
// requests through the archive server list.
func (ft *FetchTracker) Fetch(ctx context.Context, ref Reference) (*Alert, error) {
	ft.mu.Lock()
	entry, exists := ft.inflight[ref.Identifier]
	if exists {
		ft.mu.Unlock()
		select {
		case <-entry.done:
			return entry.alert, entry.err
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	entry = &fetchEntry{
		done: make(chan struct{}),
		ts:   time.Now(),
	}
	ft.inflight[ref.Identifier] = entry
	ft.mu.Unlock()

	ft.doFetch(ref, entry)

	return entry.alert, entry.err
}

// doFetch tries each archive server in order. It uses its own timeout
// independent of any caller's context, so the fetch completes even if the
// initiating caller's context is cancelled -- other waiters still get the
// result.
func (ft *FetchTracker) doFetch(ref Reference, entry *fetchEntry) {
	defer close(entry.done)

	ctx, cancel := context.WithTimeout(context.Background(), ft.fetchTimeout)
	defer cancel()

	var lastErr error
	for _, server := range ft.archiveServers {
		if ctx.Err() != nil {
			break
		}
		alert, err := ref.FetchWithContext(ctx, ft.httpClient, server)
		if err != nil {
			log.Warnf("Archive fetch of %s from %s failed: %v", ref.Identifier, server, err)
			lastErr = err
			continue
		}
		entry.alert = alert
		entry.err = nil
		return
	}

	if lastErr != nil {
		entry.err = fmt.Errorf("all archive servers failed for %s: %w", ref.Identifier, lastErr)
	} else {
		entry.err = fmt.Errorf("fetch of %s timed out after %s", ref.Identifier, ft.fetchTimeout)
	}
}

// CleanCompleted removes entries that finished more than entryTTL ago.
func (ft *FetchTracker) CleanCompleted() {
	ft.mu.Lock()
	defer ft.mu.Unlock()
	cutoff := time.Now().Add(-ft.entryTTL)
	for id, entry := range ft.inflight {
		select {
		case <-entry.done:
			if entry.ts.Before(cutoff) {
				delete(ft.inflight, id)
			}
		default:
		}
	}
}
