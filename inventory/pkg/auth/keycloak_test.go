// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

//nolint:testpackage // testing internal functions
package auth

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/Nerzal/gocloak/v13"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/mocks"
	inv_testing "github.com/open-edge-platform/infra-core/inventory/v2/pkg/testing"
)

var testValue = "testValue"

func Test_newKeycloakSecretService(t *testing.T) {
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name    string
		args    args
		envVars map[string]string
		want    AuthService
		wantErr bool
	}{
		{
			name: fmt.Sprintf("Missing %s env variable", EnvNameOnboardingManagerClientName),
			args: args{
				ctx: context.Background(),
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: fmt.Sprintf("Missing %s env variable", EnvNameOnboardingCredentialsSecretName),
			args: args{
				ctx: context.Background(),
			},
			envVars: map[string]string{
				EnvNameOnboardingManagerClientName: testValue,
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: fmt.Sprintf("Missing %s env variable", EnvNameOnboardingCredentialsSecretKey),
			args: args{
				ctx: context.Background(),
			},
			envVars: map[string]string{
				EnvNameOnboardingManagerClientName:     testValue,
				EnvNameOnboardingCredentialsSecretName: testValue,
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "Failed to login to Keycloak",
			args: args{
				ctx: context.Background(),
			},
			envVars: map[string]string{
				EnvNameOnboardingManagerClientName:     testValue,
				EnvNameOnboardingCredentialsSecretName: testValue,
				EnvNameOnboardingCredentialsSecretKey:  testValue,
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if len(tt.envVars) > 0 {
				if err := inv_testing.SetEnvVariables(tt.envVars); err != nil {
					t.Errorf("SetEnvVariables() error = %v", err)
				}
			}

			got, err := newKeycloakSecretService(tt.args.ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("newKeycloakSecretService() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("newKeycloakSecretService() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getEdgeNodeClientFromTemplate(t *testing.T) {
	type args struct {
		uuid     string
		tenantID string
	}
	tests := []struct {
		name string
		args args
		want gocloak.Client
	}{
		{
			name: "Get configured Keycloak client",
			args: args{
				uuid:     "",
				tenantID: "",
			},
			want: gocloak.Client{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getEdgeNodeClientFromTemplate(tt.args.tenantID, tt.args.uuid); reflect.DeepEqual(got, tt.want) {
				t.Errorf("getEdgeNodeClientFromTemplate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_keycloakService_login(t *testing.T) {
	type args struct {
		ctx                 context.Context
		keycloakURL         string
		loginKeycloakClient func(ctx context.Context, keycloakURL string) (*gocloak.GoCloak, *gocloak.JWT, error)
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "Failed to login",
			args: args{
				ctx:                 context.Background(),
				keycloakURL:         "",
				loginKeycloakClient: nil,
			},
			wantErr: true,
		},
		{
			name: "Success",
			args: args{
				ctx:         context.Background(),
				keycloakURL: "",
				loginKeycloakClient: func(_ context.Context, _ string) (*gocloak.GoCloak, *gocloak.JWT, error) {
					return gocloak.NewClient(""), &gocloak.JWT{}, nil
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		if tt.args.loginKeycloakClient != nil {
			LoginMethod = tt.args.loginKeycloakClient
		}

		t.Run(tt.name, func(t *testing.T) {
			k := &keycloakService{}
			if err := k.login(tt.args.ctx, tt.args.keycloakURL); (err != nil) != tt.wantErr {
				t.Errorf("keycloakService.login() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_keycloakService_getServiceAccountUserIDByClientName(t *testing.T) {
	m1 := mocks.NewMockKeycloakAPI(gomock.NewController(t))
	m1.EXPECT().GetUsers(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, nil)

	m2 := mocks.NewMockKeycloakAPI(gomock.NewController(t))
	m2.EXPECT().GetUsers(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return([]*gocloak.User{{ID: &testValue}}, nil)
	m3 := mocks.NewMockKeycloakAPI(gomock.NewController(t))
	m3.EXPECT().GetUsers(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return([]*gocloak.User{{ID: &testValue}, {}}, nil)

	type fields struct {
		keycloakClient KeycloakAPI
		jwtToken       *gocloak.JWT
	}
	type args struct {
		ctx        context.Context
		clientName string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "Cannot retrieve Keycloak user",
			fields: fields{
				keycloakClient: gocloak.NewClient(""),
				jwtToken:       &gocloak.JWT{},
			},
			args: args{
				ctx: context.Background(),
			},
			wantErr: true,
		},
		{
			name: "No Keycloak users found",
			fields: fields{
				keycloakClient: m1,
				jwtToken:       &gocloak.JWT{},
			},
			args: args{
				ctx: context.Background(),
			},
			wantErr: true,
		},
		{
			name: "Success",
			fields: fields{
				keycloakClient: m2,
				jwtToken:       &gocloak.JWT{},
			},
			args: args{
				ctx: context.Background(),
			},
			want:    testValue,
			wantErr: false,
		},
		{
			name: "SuccessMultiUser",
			fields: fields{
				keycloakClient: m3,
				jwtToken:       &gocloak.JWT{},
			},
			args: args{
				ctx: context.Background(),
			},
			want:    testValue,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k := &keycloakService{
				keycloakClient: tt.fields.keycloakClient,
				jwtToken:       tt.fields.jwtToken,
			}
			got, err := k.getServiceAccountUserIDByClientName(tt.args.ctx, tt.args.clientName)
			if (err != nil) != tt.wantErr {
				t.Errorf("keycloakService.getServiceAccountUserIDByClientName() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("keycloakService.getServiceAccountUserIDByClientName() = %v, want %v", got, tt.want)
			}
		})
	}
}

//nolint:funlen // table-driven tests
func Test_keycloakService_CreateCredentialsWithUUID(t *testing.T) {
	type fields struct {
		keycloakClient func(t2 *testing.T) KeycloakAPI
		jwtToken       *gocloak.JWT
	}
	type args struct {
		ctx      context.Context
		uuid     string
		tenantID string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    string
		want1   string
		wantErr bool
	}{
		{
			name: "Failed to create Keycloak client with UUID",
			fields: fields{
				keycloakClient: func(_ *testing.T) KeycloakAPI {
					return gocloak.NewClient("")
				},
				jwtToken: &gocloak.JWT{},
			},
			args: args{
				ctx: context.Background(),
			},
			wantErr: true,
		},
		{
			name: "Failed to add default client roles for host",
			fields: fields{
				keycloakClient: func(t *testing.T) KeycloakAPI {
					t.Helper()
					m1 := mocks.NewMockKeycloakAPI(gomock.NewController(t))
					m1.EXPECT().CreateClient(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
						Return("id", nil)
					m1.EXPECT().GetUsers(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
						Return(nil, fmt.Errorf(""))
					return m1
				},
				jwtToken: &gocloak.JWT{},
			},
			args: args{
				ctx: context.Background(),
			},
			wantErr: true,
		},
		{
			name: "Failed to get Keycloak client secret",
			fields: fields{
				keycloakClient: func(t *testing.T) KeycloakAPI {
					t.Helper()
					m2 := mocks.NewMockKeycloakAPI(gomock.NewController(t))
					m2.EXPECT().CreateClient(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
						Return("id", nil)
					m2.EXPECT().GetUsers(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
						Return([]*gocloak.User{{ID: &testValue}}, nil)
					m2.EXPECT().GetGroups(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
						Return([]*gocloak.Group{{ID: &testValue}}, nil)
					m2.EXPECT().AddUserToGroup(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
						Return(nil)
					m2.EXPECT().GetClientSecret(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
						Return(&gocloak.CredentialRepresentation{}, fmt.Errorf(""))
					return m2
				},
				jwtToken: &gocloak.JWT{},
			},
			args: args{
				ctx: context.Background(),
			},
			wantErr: true,
		},
		{
			name: "Received empty client secret",
			fields: fields{
				keycloakClient: func(t *testing.T) KeycloakAPI {
					t.Helper()
					m3 := mocks.NewMockKeycloakAPI(gomock.NewController(t))
					m3.EXPECT().CreateClient(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
						Return("id", nil)
					m3.EXPECT().GetUsers(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
						Return([]*gocloak.User{{ID: &testValue}}, nil)
					m3.EXPECT().GetGroups(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
						Return([]*gocloak.Group{{ID: &testValue}}, nil)
					m3.EXPECT().AddUserToGroup(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
						Return(nil)
					m3.EXPECT().GetClientSecret(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
						Return(&gocloak.CredentialRepresentation{Value: nil}, nil)
					return m3
				},
				jwtToken: &gocloak.JWT{},
			},
			args: args{
				ctx: context.Background(),
			},
			wantErr: true,
		},
		{
			name: "Received empty groups",
			fields: fields{
				keycloakClient: func(t *testing.T) KeycloakAPI {
					t.Helper()
					m3 := mocks.NewMockKeycloakAPI(gomock.NewController(t))
					m3.EXPECT().CreateClient(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
						Return("id", nil)
					m3.EXPECT().GetUsers(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
						Return([]*gocloak.User{{ID: &testValue}}, nil)
					m3.EXPECT().GetGroups(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
						Return([]*gocloak.Group{}, nil)
					return m3
				},
				jwtToken: &gocloak.JWT{},
			},
			args: args{
				ctx: context.Background(),
			},
			wantErr: true,
		},
		{
			name: "Failed to add user to group",
			fields: fields{
				keycloakClient: func(t *testing.T) KeycloakAPI {
					t.Helper()
					m3 := mocks.NewMockKeycloakAPI(gomock.NewController(t))
					m3.EXPECT().CreateClient(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
						Return("id", nil)
					m3.EXPECT().GetUsers(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
						Return([]*gocloak.User{{ID: &testValue}}, nil)
					m3.EXPECT().GetGroups(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
						Return([]*gocloak.Group{{ID: &testValue}}, nil)
					m3.EXPECT().AddUserToGroup(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
						Return(fmt.Errorf(""))
					return m3
				},
				jwtToken: &gocloak.JWT{},
			},
			args: args{
				ctx: context.Background(),
			},
			wantErr: true,
		},
		{
			name: "Success",
			fields: fields{
				keycloakClient: func(t *testing.T) KeycloakAPI {
					t.Helper()
					m4 := mocks.NewMockKeycloakAPI(gomock.NewController(t))
					m4.EXPECT().CreateClient(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
						Return("id", nil)
					m4.EXPECT().GetUsers(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
						Return([]*gocloak.User{{ID: &testValue}}, nil)
					m4.EXPECT().GetGroups(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
						Return([]*gocloak.Group{{ID: &testValue}}, nil)
					m4.EXPECT().AddUserToGroup(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
						Return(nil)
					m4.EXPECT().GetClientSecret(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
						Return(&gocloak.CredentialRepresentation{Value: &testValue}, nil)
					return m4
				},
				jwtToken: &gocloak.JWT{},
			},
			args: args{
				ctx: context.Background(),
			},
			want:    "edgenode-",
			want1:   testValue,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k := &keycloakService{
				keycloakClient: tt.fields.keycloakClient(t),
				jwtToken:       tt.fields.jwtToken,
			}
			got, got1, err := k.CreateCredentialsWithUUID(tt.args.ctx, tt.args.tenantID, tt.args.uuid)
			if (err != nil) != tt.wantErr {
				t.Errorf("keycloakService.CreateCredentialsWithUUID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("keycloakService.CreateCredentialsWithUUID() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("keycloakService.CreateCredentialsWithUUID() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

//nolint:funlen // table-driven tests
func Test_keycloakService_GetCredentialsByUUID(t *testing.T) {
	type fields struct {
		keycloakClient func(*testing.T) KeycloakAPI
		jwtToken       *gocloak.JWT
	}
	type args struct {
		ctx      context.Context
		uuid     string
		tenantID string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    string
		want1   string
		wantErr bool
	}{
		{
			name: "Keycloak client for edge node by UUID does not exist",
			fields: fields{
				keycloakClient: func(_ *testing.T) KeycloakAPI {
					return gocloak.NewClient("")
				},
				jwtToken: &gocloak.JWT{},
			},
			args: args{
				ctx: context.Background(),
			},
			wantErr: true,
		},
		{
			name: "No Keycloak clients found",
			fields: fields{
				keycloakClient: func(t *testing.T) KeycloakAPI {
					t.Helper()
					m := mocks.NewMockKeycloakAPI(gomock.NewController(t))
					m.EXPECT().GetClients(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
						Return(nil, nil)
					return m
				},
				jwtToken: &gocloak.JWT{},
			},
			args: args{
				ctx: context.Background(),
			},
			wantErr: true,
		},
		{
			name: "Success",
			fields: fields{
				keycloakClient: func(t *testing.T) KeycloakAPI {
					t.Helper()
					m := mocks.NewMockKeycloakAPI(gomock.NewController(t))
					m.EXPECT().GetClients(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
						Return([]*gocloak.Client{{Secret: &testValue}}, nil)
					return m
				},
				jwtToken: &gocloak.JWT{},
			},
			args: args{
				ctx: context.Background(),
			},
			want:    "edgenode-",
			want1:   testValue,
			wantErr: false,
		},
		{
			name: "Failed to get Keycloak client secret",
			fields: fields{
				keycloakClient: func(t *testing.T) KeycloakAPI {
					t.Helper()
					m := mocks.NewMockKeycloakAPI(gomock.NewController(t))
					m.EXPECT().GetClients(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
						Return([]*gocloak.Client{{Secret: nil, ID: &testValue}}, nil)
					m.EXPECT().GetClientSecret(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
						Return(&gocloak.CredentialRepresentation{}, fmt.Errorf(""))
					return m
				},
				jwtToken: &gocloak.JWT{},
			},
			args: args{
				ctx: context.Background(),
			},
			wantErr: true,
		},
		{
			name: "Received empty client secret",
			fields: fields{
				keycloakClient: func(t *testing.T) KeycloakAPI {
					t.Helper()
					m := mocks.NewMockKeycloakAPI(gomock.NewController(t))
					m.EXPECT().GetClients(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
						Return([]*gocloak.Client{{Secret: nil, ID: &testValue}}, nil)
					m.EXPECT().GetClientSecret(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
						Return(&gocloak.CredentialRepresentation{}, nil)
					return m
				},
				jwtToken: &gocloak.JWT{},
			},
			args: args{
				ctx: context.Background(),
			},
			wantErr: true,
		},
		{
			name: "Success",
			fields: fields{
				keycloakClient: func(t *testing.T) KeycloakAPI {
					t.Helper()
					m := mocks.NewMockKeycloakAPI(gomock.NewController(t))
					m.EXPECT().GetClients(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
						Return([]*gocloak.Client{{Secret: nil, ID: &testValue}}, nil)
					m.EXPECT().GetClientSecret(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
						Return(&gocloak.CredentialRepresentation{Value: &testValue}, nil)
					return m
				},
				jwtToken: &gocloak.JWT{},
			},
			args: args{
				ctx: context.Background(),
			},
			want:    "edgenode-",
			want1:   testValue,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k := &keycloakService{
				keycloakClient: tt.fields.keycloakClient(t),
				jwtToken:       tt.fields.jwtToken,
			}
			got, got1, err := k.GetCredentialsByUUID(tt.args.ctx, tt.args.tenantID, tt.args.uuid)
			if (err != nil) != tt.wantErr {
				t.Errorf("keycloakService.GetCredentialsByUUID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("keycloakService.GetCredentialsByUUID() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("keycloakService.GetCredentialsByUUID() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

//nolint:funlen // table-driven tests
func Test_keycloakService_RevokeCredentialsByUUID(t *testing.T) {
	type fields struct {
		keycloakClient func(*testing.T) KeycloakAPI
		jwtToken       *gocloak.JWT
	}
	type args struct {
		ctx      context.Context
		uuid     string
		tenantID string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "Keycloak client for edge node by UUID does not exist",
			fields: fields{
				keycloakClient: func(_ *testing.T) KeycloakAPI {
					return gocloak.NewClient("")
				},
				jwtToken: &gocloak.JWT{},
			},
			args: args{
				ctx: context.Background(),
			},
			wantErr: true,
		},
		{
			name: "No Keycloak clients found for UUID",
			fields: fields{
				keycloakClient: func(t *testing.T) KeycloakAPI {
					t.Helper()
					m1 := mocks.NewMockKeycloakAPI(gomock.NewController(t))
					m1.EXPECT().GetClients(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, nil)
					return m1
				},
				jwtToken: &gocloak.JWT{},
			},
			args: args{
				ctx: context.Background(),
			},
			wantErr: true,
		},
		{
			name: "Multiple Keycloak clients found for UUID",
			fields: fields{
				keycloakClient: func(t *testing.T) KeycloakAPI {
					t.Helper()
					m1 := mocks.NewMockKeycloakAPI(gomock.NewController(t))
					m1.EXPECT().GetClients(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
						Return([]*gocloak.Client{{}, {}}, nil)
					return m1
				},
				jwtToken: &gocloak.JWT{},
			},
			args: args{
				ctx: context.Background(),
			},
			wantErr: false,
		},
		{
			name: "Found Keycloak client for UUID with empty ID",
			fields: fields{
				keycloakClient: func(t *testing.T) KeycloakAPI {
					t.Helper()
					m2 := mocks.NewMockKeycloakAPI(gomock.NewController(t))
					m2.EXPECT().GetClients(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
						Return([]*gocloak.Client{{}}, nil)
					return m2
				},
				jwtToken: &gocloak.JWT{},
			},
			args: args{
				ctx: context.Background(),
			},
			wantErr: false,
		},
		{
			name: "Failed to delete Keycloak client",
			fields: fields{
				keycloakClient: func(t *testing.T) KeycloakAPI {
					t.Helper()
					m3 := mocks.NewMockKeycloakAPI(gomock.NewController(t))
					m3.EXPECT().GetClients(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
						Return([]*gocloak.Client{{ID: &testValue}}, nil)
					m3.EXPECT().DeleteClient(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
						Return(fmt.Errorf(""))
					return m3
				},
				jwtToken: &gocloak.JWT{},
			},
			args: args{
				ctx: context.Background(),
			},
			wantErr: true,
		},
		{
			name: "Success",
			fields: fields{
				keycloakClient: func(t *testing.T) KeycloakAPI {
					t.Helper()
					m4 := mocks.NewMockKeycloakAPI(gomock.NewController(t))
					m4.EXPECT().GetClients(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
						Return([]*gocloak.Client{{ID: &testValue}}, nil)
					m4.EXPECT().DeleteClient(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
					return m4
				},
				jwtToken: &gocloak.JWT{},
			},
			args: args{
				ctx: context.Background(),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k := &keycloakService{
				keycloakClient: tt.fields.keycloakClient(t),
				jwtToken:       tt.fields.jwtToken,
			}
			if err := k.RevokeCredentialsByUUID(tt.args.ctx, tt.args.tenantID, tt.args.uuid); (err != nil) != tt.wantErr {
				t.Errorf("keycloakService.RevokeCredentialsByUUID() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_keycloakService_Logout(t *testing.T) {
	type fields struct {
		keycloakClient func(*testing.T) KeycloakAPI
		jwtToken       *gocloak.JWT
	}
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{
			name: "Skip logging out if refresh_token is not provided",
			fields: fields{
				keycloakClient: func(_ *testing.T) KeycloakAPI {
					return gocloak.NewClient("")
				},
				jwtToken: &gocloak.JWT{},
			},
			args: args{
				ctx: context.Background(),
			},
		},
		{
			name: "Failed to logout from Keycloak",
			fields: fields{
				keycloakClient: func(t *testing.T) KeycloakAPI {
					t.Helper()
					m := mocks.NewMockKeycloakAPI(gomock.NewController(t))
					m.EXPECT().Logout(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
						Return(fmt.Errorf(""))
					return m
				},
				jwtToken: &gocloak.JWT{RefreshToken: testValue},
			},
			args: args{
				ctx: context.Background(),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(_ *testing.T) {
			k := &keycloakService{
				keycloakClient: tt.fields.keycloakClient(t),
				jwtToken:       tt.fields.jwtToken,
			}
			k.Logout(tt.args.ctx)
		})
	}
}

func Test_RevokeHostCredentials(t *testing.T) {
	tenantID := "11111111-1111-1111-1111-111111111111"

	t.Run("AuthServiceFactory is not initialized. Revoking should fail", func(t *testing.T) {
		err := RevokeHostCredentials(context.Background(), tenantID, "host-12345678")
		assert.Error(t, err)
	})

	t.Run("Initializing AuthServiceFactory, revoking of credentials should fail", func(t *testing.T) {
		AuthServiceFactory = AuthServiceMockFactory(t, false, false, true)
		// revoking should fail (mock won't allow revokeation to succeed)
		err := RevokeHostCredentials(context.Background(), tenantID, "host-12345678")
		assert.Error(t, err)
	})

	t.Run("Initializing AuthServiceFactory again, revoking of credentials should succeed", func(t *testing.T) {
		// initializing AuthServiceFactory again
		// revoking of credentials should succeed
		AuthServiceFactory = AuthServiceMockFactory(t, false, false, false)
		// revoking should succeed
		err := RevokeHostCredentials(context.Background(), tenantID, "host-12345678")
		assert.NoError(t, err)
	})
}

func Test_keycloakService_getGroupIDByName(t *testing.T) {
	m1 := mocks.NewMockKeycloakAPI(gomock.NewController(t))
	m1.EXPECT().GetGroups(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return([]*gocloak.Group(nil), status.Error(codes.Aborted, ""))
	m2 := mocks.NewMockKeycloakAPI(gomock.NewController(t))
	m2.EXPECT().GetGroups(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return([]*gocloak.Group{{ID: &testValue}}, nil)
	m3 := mocks.NewMockKeycloakAPI(gomock.NewController(t))
	m3.EXPECT().GetGroups(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return([]*gocloak.Group{}, nil)
	m4 := mocks.NewMockKeycloakAPI(gomock.NewController(t))
	m4.EXPECT().GetGroups(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return([]*gocloak.Group{{ID: &testValue}, {}}, nil)

	type fields struct {
		keycloakClient KeycloakAPI
		jwtToken       *gocloak.JWT
	}
	type args struct {
		ctx      context.Context
		roleName string
	}
	tests := []struct {
		name         string
		fields       fields
		args         args
		expErrorCode codes.Code
		wantErr      bool
	}{
		{
			name: "error getting roles",
			fields: fields{
				keycloakClient: m1,
				jwtToken:       &gocloak.JWT{},
			},
			args: args{
				ctx: context.Background(),
			},
			expErrorCode: codes.Internal,
			wantErr:      true,
		},
		{
			name: "Success",
			fields: fields{
				keycloakClient: m2,
				jwtToken:       &gocloak.JWT{},
			},
			args: args{
				ctx: context.Background(),
			},
			wantErr: false,
		},
		{
			name: "NoGroups",
			fields: fields{
				keycloakClient: m3,
				jwtToken:       &gocloak.JWT{},
			},
			args: args{
				ctx: context.Background(),
			},
			expErrorCode: codes.NotFound,
			wantErr:      true,
		},
		{
			name: "SuccessMultiGroups",
			fields: fields{
				keycloakClient: m4,
				jwtToken:       &gocloak.JWT{},
			},
			args: args{
				ctx: context.Background(),
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k := &keycloakService{
				keycloakClient: tt.fields.keycloakClient,
				jwtToken:       tt.fields.jwtToken,
			}
			_, err := k.getGroupIDByName(tt.args.ctx, tt.args.roleName)
			require.Equalf(t, tt.wantErr, err != nil,
				"keycloakService.getRoleIDFromRoleName() error = %v, wantErr %v", err, tt.wantErr)
			if tt.wantErr {
				assert.Equal(t, tt.expErrorCode, status.Code(err))
			}
		})
	}
}

func Test_keycloakService_getGroupIDByName_withRoleCache(t *testing.T) {
	m1 := mocks.NewMockKeycloakAPI(gomock.NewController(t))
	m1.EXPECT().GetGroups(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return([]*gocloak.Group{{ID: &testValue}}, nil).Times(1)

	k := &keycloakService{
		keycloakClient:   m1,
		jwtToken:         &gocloak.JWT{},
		enableGroupCache: true,
		groupIDCache:     setupGroupCache(),
	}
	_, err := k.getGroupIDByName(context.Background(), "")
	require.NoError(t, err)

	// Second time we should be hitting the cache, otherwise it will error
	_, err = k.getGroupIDByName(context.Background(), "")
	require.NoError(t, err)
}
