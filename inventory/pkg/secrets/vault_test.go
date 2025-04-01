// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

//nolint:testpackage // testing internal functions
package secrets

import (
	"context"
	"flag"
	"fmt"
	"os"
	"testing"

	"github.com/golang/mock/gomock"
	vault "github.com/hashicorp/vault/api"
	"github.com/hashicorp/vault/api/auth/kubernetes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/mocks"
)

func TestMain(m *testing.M) {
	// Only needed to suppress the error
	flag.String(
		"policyBundle",
		"/rego/policy_bundle.tar.gz",
		"Path of policy rego file",
	)
	flag.Parse()

	run := m.Run() // run all tests
	os.Exit(run)
}

func Test_getVaultClient(t *testing.T) {
	type args struct {
		vaultURL string

		vaultClientFactory func(c *vault.Config) (*vault.Client, error)
	}
	tests := []struct {
		name    string
		args    args
		want    *vault.Client
		wantErr bool
	}{
		{
			name: "Success",
			args: args{
				vaultURL: DefaultVaultURL,
			},
			wantErr: false,
		},
		{
			name: "Failed",
			args: args{
				vaultURL: DefaultVaultURL,
				vaultClientFactory: func(_ *vault.Config) (*vault.Client, error) {
					return nil, fmt.Errorf("")
				},
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.args.vaultClientFactory != nil {
				NewClientFactory = tt.args.vaultClientFactory
			}

			_, err := getVaultClient(tt.args.vaultURL)
			if (err != nil) != tt.wantErr {
				t.Errorf("getVaultClient() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func Test_vaultService_Logout(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		m1 := mocks.NewMockVaultAPI(gomock.NewController(t))
		m1.EXPECT().RevokeToken(gomock.Any()).Return(nil)

		ctx := context.Background()
		v := &vaultService{
			vaultClient: m1,
		}
		v.Logout(ctx)
	})

	t.Run("Failed", func(t *testing.T) {
		m2 := mocks.NewMockVaultAPI(gomock.NewController(t))
		m2.EXPECT().RevokeToken(gomock.Any()).Return(fmt.Errorf(""))
		ctx := context.Background()
		v := &vaultService{
			vaultClient: m2,
		}
		v.Logout(ctx)
	})
}

func Test_vaultService_ReadSecret(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		ctx := context.Background()
		m := mocks.NewMockVaultAPI(gomock.NewController(t))
		m.EXPECT().Read(gomock.Eq(ctx), gomock.Eq(vaultSecretBaseURL+"path")).Return(&vault.Secret{
			Data: map[string]interface{}{
				"test": 11,
			},
		}, nil)
		v := &vaultService{
			vaultClient: m,
		}
		data, err := v.ReadSecret(ctx, "path")
		require.NoError(t, err)
		require.NotNil(t, data)
		assert.Equal(t, data["test"], 11)
	})

	t.Run("Failed to read", func(t *testing.T) {
		ctx := context.Background()
		m := mocks.NewMockVaultAPI(gomock.NewController(t))
		m.EXPECT().Read(gomock.Eq(ctx), gomock.Eq(vaultSecretBaseURL+"path")).Return(nil, fmt.Errorf(""))

		v := &vaultService{
			vaultClient: m,
		}
		data, err := v.ReadSecret(ctx, "path")
		require.Error(t, err)
		require.Nil(t, data)
	})

	t.Run("Failed to read - empty response", func(t *testing.T) {
		ctx := context.Background()
		m := mocks.NewMockVaultAPI(gomock.NewController(t))
		m.EXPECT().Read(gomock.Eq(ctx), gomock.Eq(vaultSecretBaseURL+"path")).Return(nil, nil)

		v := &vaultService{
			vaultClient: m,
		}
		data, err := v.ReadSecret(ctx, "path")
		require.Error(t, err)
		require.Nil(t, data)
	})
}

func Test_vaultService_WriteSecret(t *testing.T) {
	secret := map[string]interface{}{
		"test": 11,
	}

	t.Run("Success", func(t *testing.T) {
		ctx := context.Background()

		m := mocks.NewMockVaultAPI(gomock.NewController(t))
		m.EXPECT().Write(gomock.Any(), gomock.Eq(vaultSecretBaseURL+"path"), gomock.Eq(secret)).Return(&vault.Secret{
			Data: secret,
		}, nil)
		v := &vaultService{
			vaultClient: m,
		}
		data, err := v.WriteSecret(ctx, "path", secret)
		require.NoError(t, err)
		require.NotNil(t, data)
		assert.Equal(t, data["test"], 11)
	})

	t.Run("Failed to write", func(t *testing.T) {
		ctx := context.Background()
		m := mocks.NewMockVaultAPI(gomock.NewController(t))
		m.EXPECT().Write(gomock.Any(), gomock.Eq(vaultSecretBaseURL+"path"), gomock.Eq(secret)).Return(nil, fmt.Errorf(""))
		v := &vaultService{
			vaultClient: m,
		}
		data, err := v.WriteSecret(ctx, "path", secret)
		require.Error(t, err)
		require.Nil(t, data)
	})

	t.Run("Failed to write - empty response", func(t *testing.T) {
		ctx := context.Background()
		m := mocks.NewMockVaultAPI(gomock.NewController(t))
		m.EXPECT().Write(gomock.Any(), gomock.Eq(vaultSecretBaseURL+"path"), gomock.Eq(secret)).Return(nil, nil)
		v := &vaultService{
			vaultClient: m,
		}
		data, err := v.WriteSecret(ctx, "path", secret)
		require.Error(t, err)
		require.Nil(t, data)
	})
}

func Test_newVaultService(t *testing.T) {
	t.Run("VaultSvcFail1", func(t *testing.T) {
		ctx := context.Background()
		AuthMethod = mockNewKubernetesAuth
		ss, err := newVaultService(ctx)
		assert.Nil(t, ss)
		assert.Error(t, err)
	})

	t.Run("VaultSvcFail2", func(t *testing.T) {
		ctx := context.Background()
		AuthMethod = mockNewKubernetesAuthFail
		ss, err := newVaultService(ctx)
		assert.Nil(t, ss)
		assert.Error(t, err)
	})
}

func mockNewKubernetesAuth(_ string, _ ...kubernetes.LoginOption) (*kubernetes.KubernetesAuth, error) {
	auth := &kubernetes.KubernetesAuth{}
	err := kubernetes.WithMountPath("kubernetes")(auth)
	return auth, err
}

func mockNewKubernetesAuthFail(_ string, _ ...kubernetes.LoginOption) (*kubernetes.KubernetesAuth, error) {
	return nil, fmt.Errorf("")
}
