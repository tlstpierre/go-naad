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

	handleAlertSummary := func(w http.ResponseWriter, r *http.Request) {
		HandleAlertSummary(w, r, cache)
	}
	handleAlertDetail := func(w http.ResponseWriter, r *http.Request) {
		HandleAlertDetail(w, r, cache)
	}

	r := mux.NewRouter()
	r.HandleFunc("/api/history", handleGetAll).Methods("GET")
	r.HandleFunc("/api/history/{id}", handleGetAlert).Methods("GET")
	r.HandleFunc("/api/summary", handleGetSummary).Methods("GET")

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
	s.server.Shutdown(context.TODO())
}

func HandleGetAlert(w http.ResponseWriter, r *http.Request, cache *naadcache.Cache) {
}

func HandleGetAll(w http.ResponseWriter, r *http.Request, cache *naadcache.Cache) {
	history := cache.DumpHistory()
	encoder := json.NewEncoder(w)
	/*
		var data struct {
			History naadcache.CacheHistory
		}
		data.History = history
	*/
	err := encoder.Encode(history)
	if err != nil {
		log.Errorf("Problem encoding history - %v", err)
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func HandleGetSummary(w http.ResponseWriter, r *http.Request, cache *naadcache.Cache) {
	history := cache.DumpHistory()
	encoder := json.NewEncoder(w)
	summary := make([]naadxml.AlertSummary, len(history))
	for i := 0; i < len(history); i++ {
		summary[i] = history[i].Current.Summary()
	}
	err := encoder.Encode(summary)
	if err != nil {
		log.Errorf("Problem encoding history - %v", err)
		w.WriteHeader(http.StatusInternalServerError)
	}
}
