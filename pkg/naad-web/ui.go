package naadweb

import (
	"embed"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	"github.com/tlstpierre/go-naad/pkg/naad-cache"
	"github.com/tlstpierre/go-naad/pkg/naad-xml"
	"html/template"
	"net/http"
	"net/url"
)

//go:embed html/*.html
var htmlFS embed.FS

var (
	alertList   = template.New("alerts.html")
	alertDetail = template.New("alertdetail.html")
)

func init() {
	alertList.Funcs(template.FuncMap{
		"idencode": IDEncode,
	})
	_, err := alertList.ParseFS(htmlFS, "html/alerts.html")
	if err != nil {
		log.Fatal(err)
	}
	_, err = alertDetail.ParseFS(htmlFS, "html/alertdetail.html")
	if err != nil {
		log.Fatal(err)
	}
}

func IDEncode(in string) string {
	return url.QueryEscape(in)
}

func IDDecode(in string) string {
	out, err := url.QueryUnescape(in)
	if err != nil {
		log.Errorf("Problem unescaping URL %s - %v", in, err)
		return ""
	}
	return out
}

func HandleAlertSummary(w http.ResponseWriter, r *http.Request, cache *naadcache.Cache) {
	history := cache.DumpHistory()
	summary := make([]naadxml.AlertSummary, len(history))
	for i := 0; i < len(history); i++ {
		summary[i] = history[i].Current.Summary()
	}

	var templateData struct {
		Summary []naadxml.AlertSummary
	}
	templateData.Summary = summary

	err := alertList.Execute(w, templateData)
	if err != nil {
		log.Errorf("Problem with alert list template - %v", err)
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func HandleAlertDetail(w http.ResponseWriter, r *http.Request, cache *naadcache.Cache) {
	vars := mux.Vars(r)
	id, found := vars["id"]
	if !found {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	log.Infof("Detail requested for ID %s", id)
	id = IDDecode(id)
	log.Infof("Detail for ID unescapes to %s", id)
	alert, history := cache.Get(id)

	if alert == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	var templateData struct {
		History *naadcache.AlertHistory
		Summary naadxml.AlertSummary
	}
	templateData.History = history
	templateData.Summary = alert.Summary()
	log.Infof("Summary is %+v", alert.Summary())

	err := alertDetail.Execute(w, templateData)
	if err != nil {
		log.Errorf("Problem with alert detail template - %v", err)
		w.WriteHeader(http.StatusInternalServerError)
	}
}
