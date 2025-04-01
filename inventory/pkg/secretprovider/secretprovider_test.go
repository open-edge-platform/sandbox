// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

//nolint:testpackage // testing internal functions
package secretprovider

import (
	"context"
	"flag"
	"fmt"
	"os"
	"testing"

	"github.com/golang/mock/gomock"

	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/flags"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/mocks"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/secrets"
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

func Test_Init(t *testing.T) {
	type args struct {
		ctx         context.Context
		secretNames []string
	}
	tests := []struct {
		name                         string
		args                         args
		disableCredentialsManagement bool
		wantErr                      bool
	}{
		{
			name: "Vault service error",
			args: args{
				ctx:         context.Background(),
				secretNames: []string{"test-client-secret-1", "test-client-secret-2"},
			},
			disableCredentialsManagement: false,
			wantErr:                      true,
		},
		{
			name: "Skip secrets initialization to avoid error",
			args: args{
				ctx:         context.Background(),
				secretNames: []string{"test-client-secret-1", "test-client-secret-2"},
			},
			disableCredentialsManagement: true,
			wantErr:                      false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			*flags.FlagDisableCredentialsManagement = tt.disableCredentialsManagement

			if err := Init(tt.args.ctx, tt.args.secretNames); (err != nil) != tt.wantErr {
				t.Errorf("Init() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
		*flags.FlagDisableCredentialsManagement = false
	}
}

func Test_Init_MockSecretsService(t *testing.T) {
	m := mocks.NewMockSecretsService(gomock.NewController(t))

	secrets.SecretServiceFactory = func(context.Context) (secrets.SecretsService, error) {
		return m, nil
	}

	type args struct {
		ctx         context.Context
		secretNames []string
	}
	tests := []struct {
		name          string
		args          args
		credentials   map[string]interface{}
		readSecretErr error
		wantErr       bool
	}{
		{
			name: "Happy path",
			args: args{
				ctx:         context.Background(),
				secretNames: []string{"test-client-secret-1"},
			},
			credentials: map[string]interface{}{
				"data": map[string]interface{}{
					"client_id":     "client-id-value",
					"client_secret": "client-secret-value",
				},
			},
			readSecretErr: nil,
			wantErr:       false,
		},
		{
			name: "Missing data key in credentials map",
			args: args{
				ctx:         context.Background(),
				secretNames: []string{"test-client-secret-2"},
			},
			credentials: map[string]interface{}{
				"no-data-key": map[string]interface{}{},
			},
			readSecretErr: nil,
			wantErr:       true,
		},
		{
			name: "Non string secret value type",
			args: args{
				ctx:         context.Background(),
				secretNames: []string{"test-client-secret-3"},
			},
			credentials: map[string]interface{}{
				"data": map[string]interface{}{
					"client_id": 123,
				},
			},
			readSecretErr: nil,
			wantErr:       true,
		},
		{
			name: "ReadSecret returns an error",
			args: args{
				ctx:         context.Background(),
				secretNames: []string{"test-client-secret-4"},
			},
			credentials:   nil,
			readSecretErr: fmt.Errorf("ReadSecret() error"),
			wantErr:       true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m.EXPECT().ReadSecret(tt.args.ctx, tt.args.secretNames[0]).Return(tt.credentials, tt.readSecretErr)
			m.EXPECT().Logout(tt.args.ctx)
			if err := Init(tt.args.ctx, tt.args.secretNames); (err != nil) != tt.wantErr {
				t.Errorf("Init() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_GetSecret(t *testing.T) {
	type args struct {
		secretName string
		secretKey  string
	}
	tests := []struct {
		name            string
		args            args
		clearSecretsMap bool
		want            string
	}{
		{
			name: "Get existing secret",
			args: args{
				secretName: "test-client-secret-1",
				secretKey:  "client_id",
			},
			clearSecretsMap: false,
			want:            "client-id-value",
		},
		{
			name: "Get secret from empty secrets map",
			args: args{
				secretName: "test-client-secret-1",
				secretKey:  "client_id",
			},
			clearSecretsMap: true,
			want:            "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.clearSecretsMap {
				secretsMap = make(map[string]string)
			}
			if got := GetSecret(tt.args.secretName, tt.args.secretKey); got != tt.want {
				t.Errorf("GetClientID() = %v, want %v", got, tt.want)
			}
		})
	}
}
