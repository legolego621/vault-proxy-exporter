package proxy

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/legolego621/vault-proxy-exporter/internal/config"
	"github.com/legolego621/vault-proxy-exporter/internal/vault"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
)

const (
	VaultPathMetrics = "/vault/metrics"
	ProxyPathMetrics = "/exporter/metrics"
	ProxyPathHealth  = "/exporter/health"
	ProxyTimeout     = 60 * time.Second
)

type Proxy struct {
	AddressMetricsServer string
	AddressHealthServer  string
	VaultConfig          *config.Config
	Health               bool
}

func New(addrMetricsServer, addrHealthServer string, vaultConfig *config.Config) *Proxy {
	return &Proxy{
		AddressMetricsServer: addrMetricsServer,
		AddressHealthServer:  addrHealthServer,
		VaultConfig:          vaultConfig,
	}
}

func (p *Proxy) Run() error {
	log.Info("init proxy")

	ctx, stop := signal.NotifyContext(context.Background(),
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)
	defer stop()

	var wg sync.WaitGroup

	wg.Add(2)
	errChan := make(chan error, 2)

	go p.startMetricsServer(ctx, &wg, errChan)
	go p.startHealthServer(ctx, &wg, errChan)

	if p.VaultConfig.VaultAuthMethod == config.VaultMethodAppRole {
		wg.Add(1)
		go updateToken(ctx, &wg, p)
	}

	select {
	case <-ctx.Done():
		log.Info("stopping vault-proxy-exporter")
	case err := <-errChan:
		if err != nil {
			log.Errorf("error encountered: %v", err)

			return err
		}
	}

	wg.Wait()

	return nil
}

func (p *Proxy) startMetricsServer(ctx context.Context, wg *sync.WaitGroup, errChan chan<- error) {
	defer wg.Done()

	log.Infof("starting metrics web server on %s", p.AddressMetricsServer)
	log.Infof("proxy metrics path: %s", ProxyPathMetrics)
	log.Infof("vault metrics path: %s", VaultPathMetrics)

	target := p.VaultConfig.VaultEndpoint

	targetURL, err := url.Parse(target)
	if err != nil {
		log.Errorf("error parsing url '%s' vault metrics endpoint: %v", target, err)
	}

	proxy := httputil.NewSingleHostReverseProxy(targetURL)
	proxy.Transport = &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: p.VaultConfig.VaultTLSInsecureSkipVerify, //nolint:gosec // allow insecure skip verify to vault localhost
		},
	}

	http.HandleFunc(VaultPathMetrics, p.handlerVaultMetrics(proxy, targetURL))
	http.Handle(ProxyPathMetrics, promhttp.Handler())

	server := &http.Server{
		Addr:              p.AddressMetricsServer,
		ReadHeaderTimeout: ProxyTimeout,
	}

	errChanServe := make(chan error, 1)

	go func() {
		errChanServe <- server.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		log.Info("shutting down metrics web server")
		errChan <- server.Shutdown(ctx)

		return
	case err := <-errChanServe:
		if err != nil && err != http.ErrServerClosed {
			errChan <- fmt.Errorf("metrics web server error: %v", err)

			return
		}
	}
}

func (p *Proxy) startHealthServer(ctx context.Context, wg *sync.WaitGroup, errChan chan<- error) {
	defer wg.Done()

	log.Infof("starting health web server on %s", p.AddressHealthServer)
	log.Infof("health path: %s", ProxyPathHealth)

	http.HandleFunc(ProxyPathHealth, p.handlerHealth)

	server := &http.Server{
		Addr:              p.AddressHealthServer,
		ReadHeaderTimeout: ProxyTimeout,
	}

	errChanServe := make(chan error, 1)

	go func() {
		errChanServe <- server.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		log.Info("shutting down health web server")
		errChan <- server.Shutdown(ctx)

		return
	case err := <-errChanServe:
		if err != nil && err != http.ErrServerClosed {
			errChan <- fmt.Errorf("health web server error: %v", err)

			return
		}
	}
}

func (p *Proxy) handlerHealth(w http.ResponseWriter, r *http.Request) {
	if p.isHealthy() {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "Exporter is healthy")
	} else {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, "Exporter is unhealthy")
	}
}

func (p *Proxy) handlerVaultMetrics(rp *httputil.ReverseProxy, targetURL *url.URL) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		realIP := getClientIP(r)
		realPath := config.VaultPrometheusMetricsPath
		params := config.VaultPrometheusMetricsParams

		r.Host = targetURL.Host
		r.URL.Path = realPath
		r.URL.RawQuery = params
		r.Header.Set("Authorization", "Bearer "+p.VaultConfig.VaultToken)
		r.Header.Set("X-Forwarded-For", realIP)
		r.Header.Set("X-Forwarded-Host", r.Host)

		w.Header().Set("X-Proxy-Server", "vault-proxy-exporter")

		uri := fmt.Sprintf("%s://%s%s", targetURL.Scheme, targetURL.Host, r.URL.RequestURI())
		log.Debugf("ip: %s, method: %s, proxy-path: %s, uri: %s",
			realIP,
			r.Method,
			r.URL.Path,
			uri,
		)

		rp.ServeHTTP(w, r)
	}
}

func (p *Proxy) isHealthy() bool {
	return p.Health
}

func getClientIP(r *http.Request) string {
	// try to get from X-Forwarded-For
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	// try to get from X-Real-IP
	xri := r.Header.Get("X-Real-IP")
	if xri != "" {
		return xri
	}

	// real ip
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return "unknown"
	}

	return ip
}

func updateToken(ctx context.Context, wg *sync.WaitGroup, p *Proxy) {
	defer wg.Done()

	log.Info("starting token updater")

	updater := func(p *Proxy) error {
		token, err := vault.ApproleAuthGetToken(p.VaultConfig)
		if err != nil {
			p.Health = false

			return err
		}

		p.VaultConfig.VaultToken = token
		p.Health = true

		return nil
	}

	ticker := time.NewTicker(time.Duration(p.VaultConfig.VaultApproleTokenUpdatePeriodSeconds) * time.Second)
	defer ticker.Stop()

	// init
	if err := updater(p); err != nil {
		log.Errorf("error updating vault token: %v", err)
	}

	for {
		select {
		case <-ctx.Done():
			log.Info("stopping token updater")

			return
		case <-ticker.C:
			if err := updater(p); err != nil {
				log.Errorf("error getting vault token: %v", err)
			}

			log.Debug("vault token successfully updated")
		}
	}
}
