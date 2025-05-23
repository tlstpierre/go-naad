package main

import (
	"context"
	"flag"
	log "github.com/sirupsen/logrus"
	"github.com/tlstpierre/go-naad/pkg/naad-cache"
	"github.com/tlstpierre/go-naad/pkg/naad-socket"
	"github.com/tlstpierre/go-naad/pkg/naad-web"
	"github.com/tlstpierre/go-naad/pkg/naad-xml"
	"io"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

var (
	LogLevel   = flag.String("loglevel", "info", "Log Level")
	LogFile    = flag.String("logfile", "", "Path to log file")
	ConfigFile = flag.String("config", "", "Config file path")
	configData *Config
	wg         *sync.WaitGroup
	NaadCache  *naadcache.Cache
)

func main() {
	flag.Parse()
	lvl, _ := log.ParseLevel(*LogLevel)
	log.SetLevel(lvl)

	configData = new(Config)
	configData.Initialize()

	var err error
	if *ConfigFile != "" {
		err = configData.LoadFile(*ConfigFile)
		if err != nil {
			log.Fatal(err)
		}
	} else {
		log.Infof("Using default config")
		configData.OutputDefault()
	}
	if *LogFile != "" {
		var lf io.Writer
		lf, err = os.OpenFile(*LogFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0664)
		if err != nil {
			log.Fatalf("Cannot open log file at %s", *LogFile)
		} else {
			log.Warnf("Logging will now be directed to %s", *LogFile)
			log.SetOutput(lf)
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	wg = &sync.WaitGroup{}

	NaadCache = naadcache.NewCache()
	NaadCache.SetArchive(configData.ArchiveServers)

	rxchan := make(chan *naadxml.Alert, 4)
	infochan := make(chan *naadxml.AlertInfo, 16)
	var rxg *naadsocket.ReceiverGroup
	rxg, err = naadsocket.NewReceiverGroup(configData.StreamServers, configData.ArchiveServers, rxchan, ctx)
	if err != nil {
		log.Fatal(err)
	}

	initFilter()
	var webServer *naadweb.Server
	webServer, err = naadweb.NewServer(configData.WebListen, NaadCache)
	AnnouncerInit(ctx, wg)
	_ = NewProcessor(rxchan, infochan, ctx)
	err = rxg.Start()
	if err != nil {
		log.Fatal(err)
	}

	quitFunc := func() {
		cancel()
		webServer.Shutdown()
		wg.Wait()
		os.Exit(0)
	}

	// setup signal catching
	sigs := make(chan os.Signal, 1)
	// catch all signals since not explicitly listing
	signal.Notify(sigs)
	cacheCleaner := time.NewTicker(1 * time.Hour)
	for {
		select {
		case s := <-sigs:
			switch s {
			case syscall.SIGQUIT, syscall.SIGKILL, syscall.SIGTERM, syscall.SIGINT:
				quitFunc()
			case syscall.SIGHUP:
			}
		case <-cacheCleaner.C:
			NaadCache.Clean()

		case info := <-infochan:
			displayInfo(info)
		}
	}
}
