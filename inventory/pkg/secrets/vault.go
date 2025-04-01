// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package secrets

import (
	"context"
	"os"
	"time"

	vault "github.com/hashicorp/vault/api"
	"github.com/hashicorp/vault/api/auth/kubernetes"
	"google.golang.org/grpc/codes"

	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/errors"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/logging"
)

const (
	DefaultTimeout = 3 * time.Second

	vaultSecretBaseURL = `/secret/data/`

	DefaultVaultRole = "orch-svc"
	DefaultVaultURL  = "http://vault.orch-platform.svc.cluster.local:8200"

	EnvNameVaultURL     = "VAULT_URL"
	EnvNameVaultPKIRole = "VAULT_PKI_ROLE"
)

var zlog = logging.GetLogger("VaultService")

var (
	AuthMethod       = kubernetes.NewKubernetesAuth
	NewClientFactory = vault.NewClient
)

// VaultAPI wraps vault under interface to enable mocking for unit testing.
//
//go:generate mockgen -package mocks -destination=../mocks/vault_mock.go . VaultAPI
type VaultAPI interface {
	Read(ctx context.Context, path string) (*vault.Secret, error)
	Write(ctx context.Context, path string, data map[string]interface{}) (*vault.Secret, error)
	RevokeToken(ctx context.Context) error
}

type vaultAPI struct {
	vaultClient *vault.Client
}

func (v vaultAPI) Read(ctx context.Context, path string) (*vault.Secret, error) {
	return v.vaultClient.Logical().ReadWithContext(ctx, path)
}

func (v vaultAPI) Write(ctx context.Context, path string, data map[string]interface{}) (*vault.Secret, error) {
	return v.vaultClient.Logical().WriteWithContext(ctx, path, data)
}

func (v vaultAPI) RevokeToken(ctx context.Context) error {
	// token can be left empty, see lib docs
	return v.vaultClient.Auth().Token().RevokeSelfWithContext(ctx, "")
}

type vaultService struct {
	vaultClient VaultAPI
}

func newVaultService(ctx context.Context) (SecretsService, error) {
	vaultURL := os.Getenv(EnvNameVaultURL)
	if vaultURL == "" {
		zlog.InfraSec().Warn().Msgf("%s env variable is not set, using default value", EnvNameVaultURL)
		vaultURL = DefaultVaultURL
	}

	vaultRole := os.Getenv(EnvNameVaultPKIRole)
	if vaultRole == "" {
		zlog.InfraSec().Warn().Msgf("%s env variable is not set, using default value", EnvNameVaultPKIRole)
		vaultRole = DefaultVaultRole
	}

	ss := &vaultService{}
	err := ss.login(ctx, vaultURL, vaultRole)
	if err != nil {
		return nil, err
	}
	return ss, err
}

func getVaultClient(vaultURL string) (*vault.Client, error) {
	config := vault.DefaultConfig()
	config.Address = vaultURL

	client, err := NewClientFactory(config)
	if err != nil {
		returnErr := errors.Errorf("Failed to create Vault client")
		zlog.InfraSec().InfraErr(err).Msg("")
		return nil, returnErr
	}

	return client, nil
}

func loginToVault(ctx context.Context, vaultCli *vault.Client, vaultRole string) error {
	ctx, cancel := context.WithTimeout(ctx, DefaultTimeout)
	defer cancel()

	k8sAuth, err := AuthMethod(vaultRole)
	if err != nil {
		returnErr := errors.Errorf("Failed to create K8s auth credentials")
		zlog.InfraSec().InfraErr(err).Msg("")
		return returnErr
	}

	authInfo, err := vaultCli.Auth().Login(ctx, k8sAuth)
	if err != nil {
		returnErr := errors.Errorf("Failed to login to Vault")
		zlog.InfraSec().InfraErr(err).Msg("")
		return returnErr
	}

	if authInfo == nil {
		returnErr := errors.Errorf("no auth info was returned after login to Vault")
		zlog.InfraSec().InfraErr(returnErr).Msg("")
		return returnErr
	}

	return nil
}

func (v *vaultService) login(ctx context.Context, vaultURL, vaultRole string) error {
	client, err := getVaultClient(vaultURL)
	if err != nil {
		return err
	}
	err = loginToVault(ctx, client, vaultRole)
	if err != nil {
		return err
	}
	v.vaultClient = &vaultAPI{client}
	return nil
}

func (v *vaultService) ReadSecret(ctx context.Context, secretName string) (map[string]interface{}, error) {
	secret, err := v.vaultClient.Read(ctx, vaultSecretBaseURL+secretName)
	if err != nil {
		returnErr := errors.Errorf("Failed to read secret from Vault")
		zlog.InfraSec().Err(err).Msg("")
		return nil, returnErr
	}

	// There are scenarios in which secret will be nil, even if there is no error.
	// For example, this can happen in the case of 204 No Content response.
	// See: https://github.com/hashicorp/vault/issues/18836
	if secret == nil {
		return nil, errors.Errorfc(codes.NotFound, "Secret %s not found", secretName)
	}

	return secret.Data, nil
}

func (v *vaultService) WriteSecret(ctx context.Context, secretName string, data map[string]interface{}) (
	map[string]interface{}, error,
) {
	secret, err := v.vaultClient.Write(ctx, vaultSecretBaseURL+secretName, data)
	if err != nil {
		returnErr := errors.Errorf("Failed to write secret in Vault")
		zlog.InfraSec().Err(err).Msg("")
		return nil, returnErr
	}

	// There are scenarios in which secret will be nil, even if there is no error.
	// For example, see in case EOF in logical.go
	if secret == nil {
		return nil, errors.Errorf("Write Secret %s not successful", secretName)
	}

	return secret.Data, nil
}

func (v *vaultService) Logout(ctx context.Context) {
	err := v.vaultClient.RevokeToken(ctx)
	if err != nil {
		zlog.InfraSec().Err(err).Msgf("Failed to log out from Vault")
	}
}
