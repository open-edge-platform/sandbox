// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package secretprovider

import (
	"context"
	"sync"

	inv_errors "github.com/open-edge-platform/infra-core/inventory/v2/pkg/errors"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/flags"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/logging"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/secrets"
)

var (
	zlog = logging.GetLogger("SecretProvider")

	inst       = &secretProvider{}
	secretsMap = make(map[string]string)
	mt         = sync.Mutex{}
)

// SecretProvider implements the interaction with the secrets storage (e.g., Vault).
// It retrieves the given secrets by names.
type SecretProvider interface {
	// Init initializes the SecretProvider.
	// It should always be invoked at the very beginning, before other methods are used.
	Init(ctx context.Context, secretName []string) error
	// GetSecret obtains a value of the `secretKey` from the secret identified by the `secretName`.
	GetSecret(secretName, secretKey string) string
}

type secretProvider struct{}

func GetSecret(secretName, secretKey string) string {
	return inst.GetSecret(secretName, secretKey)
}

func Init(ctx context.Context, secretNames []string) error {
	return inst.Init(ctx, secretNames)
}

func (ss *secretProvider) GetSecret(secretName, secretKey string) string {
	mt.Lock()
	defer mt.Unlock()
	if len(secretsMap) == 0 {
		zlog.Error().Msgf("Empty client secrets map. Ensure that SecretProvider initialization was invoked.")
		return ""
	}
	res, ok := secretsMap[secretName+secretKey]
	if !ok {
		zlog.Error().Msgf("Didn't find any secret for provided key")
	}
	return res
}

func (ss *secretProvider) Init(ctx context.Context, secretNames []string) error {
	if *flags.FlagDisableCredentialsManagement {
		zlog.Warn().Msgf("disableCredentialsManagement flag is set to false, " +
			"skip secrets initialization")
		return nil
	}

	vaultS, err := secrets.SecretServiceFactory(ctx)
	if err != nil {
		return err
	}
	defer vaultS.Logout(ctx)

	for _, secretName := range secretNames {
		credentials, err := vaultS.ReadSecret(ctx, secretName)
		if err != nil {
			return err
		}

		dataMap, ok := credentials["data"].(map[string]interface{})
		if !ok {
			err = inv_errors.Errorf("Cannot read credentials data from Vault secret")
			zlog.InfraSec().Err(err).Msg("")
			return err
		}

		for secretKey, secretValue := range dataMap {
			secret, ok := secretValue.(string)
			if !ok {
				err = inv_errors.Errorf("Wrong format of %v read from Vault, expected string, got %T", secretKey, secretValue)
				zlog.InfraSec().Err(err).Msg("")
				return err
			}
			mt.Lock()
			secretsMap[secretName+secretKey] = secret
			mt.Unlock()
		}
	}

	zlog.InfraSec().Debug().Msgf("Secrets successfully initialized")

	return nil
}
