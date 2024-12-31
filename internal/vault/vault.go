package vault

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/vault-client-go"
	"github.com/hashicorp/vault-client-go/schema"
	"github.com/legolego621/vault-proxy-exporter/internal/config"
)

const (
	RequestTimeoutSeconds = 30 * time.Second
)

func ApproleAuthGetToken(cfg *config.Config) (string, error) {

	ctx := context.Background()

	// prepare a client with the given base address
	client, err := vault.New(
		vault.WithAddress(cfg.VaultEndpoint),
		vault.WithRequestTimeout(RequestTimeoutSeconds),
	)
	if err != nil {
		return "", fmt.Errorf("error creating vault client: %v", err)
	}

	resp, err := client.Auth.AppRoleLogin(
		ctx,
		schema.AppRoleLoginRequest{
			RoleId:   cfg.VaultAppRoleID,
			SecretId: cfg.VaultAppRoleSecret,
		},
		vault.WithMountPath(cfg.VaultAppRolePath),
	)

	if err != nil {
		return "", fmt.Errorf("error getting vault token: %v", err)
	}

	return resp.Auth.ClientToken, nil
}
