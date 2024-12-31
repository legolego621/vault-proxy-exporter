package config

import (
	"fmt"

	"github.com/caarlos0/env/v11"
	log "github.com/sirupsen/logrus"
)

type Config struct {
	VaultEndpoint                        string `env:"VAULT_PROXY_EXPORTER_VAULT_ENDPOINT"`
	VaultTLSInsecureSkipVerify           bool   `env:"VAULT_PROXY_EXPORTER_TLS_INSECURE_SKIP_VERIFY" envDefault:"false"`
	VaultAuthMethod                      string `env:"VAULT_PROXY_EXPORTER_VAULT_AUTH_METHOD" envDefault:"token"`
	VaultAppRolePath                     string `env:"VAULT_PROXY_EXPORTER_VAULT_APPROLE_PATH" envDefault:"approle"`
	VaultAppRoleID                       string `env:"VAULT_PROXY_EXPORTER_VAULT_APPROLE_ID"`
	VaultAppRoleSecret                   string `env:"VAULT_PROXY_EXPORTER_VAULT_APPROLE_SECRET"`
	VaultToken                           string `env:"VAULT_PROXY_EXPORTER_VAULT_TOKEN"`
	VaultApproleTokenUpdatePeriodSeconds int    `env:"VAULT_PROXY_EXPORTER_VAULT_APPROLE_TOKEN_UPDATE_PERIOD_SECONDS" envDefault:"60"`
}

const (
	VaultMethodAppRole           string = "approle"
	VaultMethodToken             string = "token"
	VaultPrometheusMetricsPath   string = "/v1/sys/metrics"
	VaultPrometheusMetricsParams string = "format=prometheus"
)

func New() *Config {
	return &Config{}
}

func (c *Config) Load() error {
	// load env
	if err := env.Parse(c); err != nil {
		return err
	}
	// validate
	if c.VaultEndpoint == "" {
		return fmt.Errorf("env VAULT_PROXY_EXPORTER_VAULT_ENDPOINT required")
	}

	if c.VaultTLSInsecureSkipVerify {
		log.Warn("enable tls insecure skip verify")
	}

	if c.VaultAuthMethod == "" {
		return fmt.Errorf("env VAULT_PROXY_EXPORTER_VAULT_AUTH_METHOD required")
	}

	switch c.VaultAuthMethod {
	case VaultMethodAppRole:
		if c.VaultAppRoleID == "" {
			return fmt.Errorf("env VAULT_PROXY_EXPORTER_VAULT_APPROLE_ID required")
		}
		if c.VaultAppRoleSecret == "" {
			return fmt.Errorf("env VAULT_PROXY_EXPORTER_VAULT_APPROLE_SECRET required")
		}
		if c.VaultAppRolePath == "" {
			return fmt.Errorf("env VAULT_PROXY_EXPORTER_VAULT_APPROLE_PATH required")
		}
		if c.VaultApproleTokenUpdatePeriodSeconds <= 5 {
			return fmt.Errorf("env VAULT_PROXY_EXPORTER_VAULT_APPROLE_TOKEN_UPDATE_PERIOD_SECONDS required and must be > 30")
		}
	case VaultMethodToken:
		if c.VaultToken == "" {
			return fmt.Errorf("env VAULT_PROXY_EXPORTER_VAULT_TOKEN required")
		}
	default:
		return fmt.Errorf("invalid env VAULT_PROXY_EXPORTER_VAULT_AUTH_METHOD: '%s', available: %s and %s",
			c.VaultAuthMethod,
			VaultMethodAppRole,
			VaultMethodToken)
	}
	return nil
}

func IsValidVaultAuthMethod(method string) bool {
	switch method {
	case VaultMethodAppRole, VaultMethodToken:
		return true
	default:
		return false
	}
}
