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
)

type Proxy struct {
	AddressServer string
	VaultConfig   *config.Config
}

func New(addressServer string, vaultConfig *config.Config) *Proxy {
	return &Proxy{
		AddressServer: addressServer,
		VaultConfig:   vaultConfig,
	}
}

func (p *Proxy) Run() error {
	log.Info("init proxy")

	var wg sync.WaitGroup

	wg.Add(2)

	ctx, stop := signal.NotifyContext(context.Background(),
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)
	defer stop()

	errChan := make(chan error, 1)

	go p.ProxyStart(ctx, &wg, errChan)
	go updateToken(ctx, &wg, p, errChan)

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

func (p *Proxy) ProxyStart(ctx context.Context, wg *sync.WaitGroup, errChan chan<- error) {
	defer wg.Done()

	log.Infof("starting web server on %s", p.AddressServer)
	log.Infof("proxy metrics path: %s", ProxyPathMetrics)
	log.Infof("vault metrics path: %s", VaultPathMetrics)

	target := p.VaultConfig.VaultEndpoint

	targetUrl, err := url.Parse(target)
	if err != nil {
		fmt.Errorf("error parsing url '%s' vault metrics endpoint: %v", target, err)
	}

	proxy := httputil.NewSingleHostReverseProxy(targetUrl)
	proxy.Transport = &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: p.VaultConfig.VaultTLSInsecureSkipVerify,
		},
	}

	handler := func(rp *httputil.ReverseProxy) func(http.ResponseWriter, *http.Request) {
		return func(w http.ResponseWriter, r *http.Request) {
			realIP := getClientIP(r)
			realPath := config.VaultPrometheusMetricsPath
			params := config.VaultPrometheusMetricsParams

			r.Host = targetUrl.Host
			r.URL.Path = realPath
			r.URL.RawQuery = params
			r.Header.Set("Authorization", "Bearer "+p.VaultConfig.VaultToken)
			r.Header.Set("X-Forwarded-For", realIP)
			r.Header.Set("X-Forwarded-Host", r.Host)

			w.Header().Set("X-Proxy-Server", "vault-proxy-exporter")

			uri := fmt.Sprintf("%s://%s%s", targetUrl.Scheme, targetUrl.Host, r.URL.RequestURI())
			log.Debugf("ip: %s, method: %s, proxy-path: %s, uri: %s",
				realIP,
				r.Method,
				r.URL.Path,
				uri,
			)

			rp.ServeHTTP(w, r)
		}
	}

	http.HandleFunc(VaultPathMetrics, handler(proxy))
	http.Handle(ProxyPathMetrics, promhttp.Handler())

	server := &http.Server{
		Addr: p.AddressServer,
	}

	errChanLocal := make(chan error, 1)

	go func() {
		errChanLocal <- server.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		log.Info("shutting down web server")
		errChan <- server.Shutdown(ctx)

		return
	case err := <-errChanLocal:
		if err != nil && err != http.ErrServerClosed {
			errChan <- fmt.Errorf("server error: %v", err)

			return
		}
	}

	return
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

func updateToken(ctx context.Context, wg *sync.WaitGroup, p *Proxy, errChan chan<- error) {
	defer wg.Done()

	log.Info("starting token updater")

	updater := func(p *Proxy) error {
		token, err := vault.ApproleAuthGetToken(p.VaultConfig)
		if err != nil {
			return err
		}

		p.VaultConfig.VaultToken = token

		return nil
	}

	ticker := time.NewTicker(time.Duration(p.VaultConfig.VaultApproleTokenUpdatePeriodSeconds) * time.Second)
	defer ticker.Stop()

	// init
	if err := updater(p); err != nil {
		errChan <- fmt.Errorf("error updating vault token: %v", err)

		return
	}

	for {
		select {
		case <-ctx.Done():
			log.Info("stopping token updater")

			return
		case <-ticker.C:
			if err := updater(p); err != nil {
				errChan <- fmt.Errorf("error getting vault token: %v", err)

				return
			}

			log.Debug("vault token successfully updated")
		}
	}
}
