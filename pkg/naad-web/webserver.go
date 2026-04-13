package naadweb

import (
	"context"
	"encoding/json"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	"github.com/tlstpierre/go-naad/pkg/naad-cache"
	"github.com/tlstpierre/go-naad/pkg/naad-xml"
	"net/http"
	"time"
)

type Server struct {
	server *http.Server
	router *mux.Router
	cache  *naadcache.Cache
}

func NewServer(listen string, cache *naadcache.Cache) (*Server, error) {
	handleGetAll := func(w http.ResponseWriter, r *http.Request) {
		HandleGetAll(w, r, cache)
	}

	handleGetAlert := func(w http.ResponseWriter, r *http.Request) {
		HandleGetAlert(w, r, cache)
	}

	handleGetSummary := func(w http.ResponseWriter, r *http.Request) {
		log.Info("Getting summary")
		HandleGetSummary(w, r, cache)
	}

	handleAlertCache := func(w http.ResponseWriter, r *http.Request) {
		HandleAlertCache(w, r, cache)
	}

	handleAlertSummary := func(w http.ResponseWriter, r *http.Request) {
		HandleAlertSummary(w, r, cache)
	}
	handleAlertDetail := func(w http.ResponseWriter, r *http.Request) {
		HandleAlertDetail(w, r, cache)
	}

	handleHealth := func(w http.ResponseWriter, r *http.Request) {
		HandleHealth(w, r, cache)
	}

	r := mux.NewRouter()
	r.HandleFunc("/", HandleIndex).Methods("GET")
	r.HandleFunc("/health", handleHealth).Methods("GET")
	r.HandleFunc("/api/history", handleGetAll).Methods("GET")
	r.HandleFunc("/api/history/{id}", handleGetAlert).Methods("GET")
	r.HandleFunc("/api/summary", handleGetSummary).Methods("GET")

	r.HandleFunc("/alertcache", handleAlertCache).Methods("GET")
	r.HandleFunc("/alertsummary", handleAlertSummary).Methods("GET")
	r.HandleFunc("/alertdetail/{id}", handleAlertDetail).Methods("GET")

	server := &http.Server{
		Addr:           listen,
		Handler:        r,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	webServer := &Server{
		server: server,
		router: r,
	}

	go func() {
		log.Error(server.ListenAndServe())
	}()

	return webServer, nil
}

func (s *Server) Shutdown() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	s.server.Shutdown(ctx)
}

func HandleHealth(w http.ResponseWriter, r *http.Request, cache *naadcache.Cache) {
	w.Header().Set("Content-Type", "application/json")
	stats := cache.Stats()
	json.NewEncoder(w).Encode(stats)
}

func HandleIndex(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(`<!DOCTYPE html>
<html><head><title>NAAD Receiver</title></head><body>
<h2>NAAD Alert Receiver</h2>
<ul>
<li><a href="/alertsummary">Alert Summary</a></li>
<li><a href="/alertcache">Alert Cache</a></li>
<li><a href="/api/history">API: History (JSON)</a></li>
<li><a href="/api/summary">API: Summary (JSON)</a></li>
</ul>
</body></html>`))
}

func HandleGetAlert(w http.ResponseWriter, r *http.Request, cache *naadcache.Cache) {
	vars := mux.Vars(r)
	id := vars["id"]
	alert, history := cache.Get(id)
	if alert == nil {
		http.Error(w, "alert not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	data := struct {
		Alert   *naadxml.Alert          `json:"alert"`
		History *naadcache.AlertHistory `json:"history,omitempty"`
	}{Alert: alert, History: history}
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Errorf("Problem encoding alert %s - %v", id, err)
	}
}

func HandleGetAll(w http.ResponseWriter, r *http.Request, cache *naadcache.Cache) {
	history := cache.DumpHistory()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(history); err != nil {
		log.Errorf("Problem encoding history - %v", err)
	}
}

func HandleGetSummary(w http.ResponseWriter, r *http.Request, cache *naadcache.Cache) {
	history := cache.DumpHistory()
	summary := make([]naadxml.AlertSummary, len(history))
	for i := 0; i < len(history); i++ {
		summary[i] = history[i].Current.Summary()
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(summary); err != nil {
		log.Errorf("Problem encoding summary - %v", err)
	}
}
