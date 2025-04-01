// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package configuration

import (
	"context"
	"fmt"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"

	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/flags"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/mocks"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/secrets"
)

func TestNewLenovoInitResourcesDefinitionLoader(t *testing.T) {
	secretProviderMock := mocks.NewMockSecretsService(gomock.NewController(t))
	secretProviderMock.EXPECT().Logout(gomock.Any()).Return()
	secretProviderMock.EXPECT().ReadSecret(gomock.Any(), gomock.AssignableToTypeOf("")).
		Return(map[string]interface{}{}, nil).AnyTimes()
	secretProviderMock.EXPECT().WriteSecret(gomock.Any(), gomock.AssignableToTypeOf(""), gomock.Any()).
		Return(nil, fmt.Errorf("cannot write secret")).AnyTimes()

	secrets.SecretServiceFactory = func(_ context.Context) (secrets.SecretsService, error) {
		return secretProviderMock, nil
	}

	irp, err := NewLenovoInitResourcesDefinitionLoader("../../configuration/default/resources-lenovo.json")
	require.NoError(t, err)
	require.NotNil(t, irp)
}

func TestNewLenovoInitResourcesDefinitionLoader_providedProviderHasMissingApiEndpoint(t *testing.T) {
	secretProviderMock := mocks.NewMockSecretsService(gomock.NewController(t))
	secretProviderMock.EXPECT().Logout(gomock.Any())

	secrets.SecretServiceFactory = func(_ context.Context) (secrets.SecretsService, error) {
		return secretProviderMock, nil
	}

	irp, err := NewLenovoInitResourcesDefinitionLoader("../../configuration/broken/lenovo/missing-endpoint.json")
	require.NoError(t, err)
	require.NotNil(t, irp)
	require.Len(t, irp.Get(), 0)
}

func TestNewLenovoInitResourcesDefinitionLoader_FailOnSecretServiceFactory(t *testing.T) {
	secrets.SecretServiceFactory = func(_ context.Context) (secrets.SecretsService, error) {
		return nil, fmt.Errorf("cannot provide SecretService instance")
	}
	irp, err := NewLenovoInitResourcesDefinitionLoader("any")
	require.Error(t, err)
	require.Nil(t, irp)
}

func TestNewLenovoInitResourcesDefinitionLoader_failOnMissingFile(t *testing.T) {
	trueValue := true
	flags.FlagDisableCredentialsManagement = &trueValue
	irp, err := NewLenovoInitResourcesDefinitionLoader("missing-file")
	require.Error(t, err)
	require.ErrorContains(t, err, "no such file")
	require.Nil(t, irp)
}

func TestNewLenovoInitResourcesDefinitionLoader_failInvalidFormatOfConfigFile(t *testing.T) {
	trueValue := true
	flags.FlagDisableCredentialsManagement = &trueValue
	irp, err := NewLenovoInitResourcesDefinitionLoader("../../configuration/broken/lenovo/no.json")
	require.Error(t, err)
	require.ErrorContains(t, err, "cannot unmarshal")
	require.Nil(t, irp)
}
