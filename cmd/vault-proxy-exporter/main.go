package main

import (
	"flag"

	"github.com/legolego621/vault-proxy-exporter/internal/config"
	"github.com/legolego621/vault-proxy-exporter/internal/proxy"
	log "github.com/sirupsen/logrus"
)

func main() {
	addrServer := flag.String("web.listen-address", ":9010", "Listening address of web server")
	logLevel := flag.String("log.level", "info", "Set log level")

	flag.Parse()
	log.SetFormatter(&log.JSONFormatter{})

	logLevelParsed, err := log.ParseLevel(*logLevel)
	if err != nil {
		log.Fatalf("error parsing log level: %v", err)
	}

	log.SetLevel(logLevelParsed)

	cfg := config.New()
	if err := cfg.Load(); err != nil {
		log.Fatalf("error loading configuration: %v", err)
	}

	p := proxy.New(*addrServer, cfg)
	if err := p.Run(); err != nil {
		log.Fatalf("error running vault-proxy-exporter: %v", err)
	}
}
