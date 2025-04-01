// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package server_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"

	computev1 "github.com/open-edge-platform/infra-core/apiv2/v2/internal/pbapi/resources/compute/v1"
	restv1 "github.com/open-edge-platform/infra-core/apiv2/v2/internal/pbapi/services/v1"
	inv_server "github.com/open-edge-platform/infra-core/apiv2/v2/internal/server"
	inv_computev1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/compute/v1"
	inventory "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/inventory/v1"
)

//nolint:funlen // Test functions are long but necessary to test all the cases.
func TestHost_Create(t *testing.T) {
	mockedClient := newMockedInventoryTestClient()
	server := inv_server.InventorygRPCServer{InvClient: mockedClient}

	cases := []struct {
		name    string
		mocks   func() []*mock.Call
		ctx     context.Context
		req     *restv1.CreateHostRequest
		wantErr bool
	}{
		{
			name: "Create Host",
			mocks: func() []*mock.Call {
				return []*mock.Call{
					mockedClient.On("Create", mock.Anything, mock.Anything).
						Return(&inventory.Resource{
							Resource: &inventory.Resource_Host{
								Host: &inv_computev1.HostResource{
									ResourceId: "host-12345678",
									Name:       "example-host",
								},
							},
						}, nil).Once(),
				}
			},
			ctx: context.Background(),
			req: &restv1.CreateHostRequest{
				Host: &computev1.HostResource{
					Name: "example-host",
				},
			},
			wantErr: false,
		},
		{
			name: "Create Host with error",
			mocks: func() []*mock.Call {
				return []*mock.Call{
					mockedClient.On("Create", mock.Anything, mock.Anything).
						Return(nil, errors.New("error")).Once(),
				}
			},
			ctx:     context.Background(),
			req:     &restv1.CreateHostRequest{},
			wantErr: true,
		},
		{
			name: "Create Host with all fields",
			mocks: func() []*mock.Call {
				return []*mock.Call{
					mockedClient.On("Create", mock.Anything, mock.Anything).
						Return(&inventory.Resource{
							Resource: &inventory.Resource_Host{
								Host: exampleInvHostResource,
							},
						}, nil).Once(),
				}
			},
			ctx: context.Background(),
			req: &restv1.CreateHostRequest{
				Host: &computev1.HostResource{
					Name: "example-host",
				},
			},
			wantErr: false,
		},
		{
			name: "Create Host with all fields and error",
			mocks: func() []*mock.Call {
				return []*mock.Call{
					mockedClient.On("Create", mock.Anything, mock.Anything).
						Return(nil, errors.New("error")).Once(),
				}
			},
			ctx: context.Background(),
			req: &restv1.CreateHostRequest{
				Host: &computev1.HostResource{
					Name: "example-host",
				},
			},
			wantErr: true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.mocks != nil {
				tc.mocks()
			}

			reply, err := server.CreateHost(tc.ctx, tc.req)
			if tc.wantErr {
				if err == nil {
					t.Errorf("CreateHost() got err = nil, want err")
				}
				return
			}
			if err != nil {
				t.Errorf("CreateHost() got err = %v, want nil", err)
				return
			}
			if reply == nil {
				t.Errorf("CreateHost() got reply = nil, want non-nil")
				return
			}
			compareProtoMessages(t, tc.req.GetHost(), reply)
		})
	}
}

func TestHost_Get(t *testing.T) {
	mockedClient := newMockedInventoryTestClient()
	server := inv_server.InventorygRPCServer{InvClient: mockedClient}

	cases := []struct {
		name    string
		mocks   func() []*mock.Call
		ctx     context.Context
		req     *restv1.GetHostRequest
		wantErr bool
	}{
		{
			name: "Get Host",
			mocks: func() []*mock.Call {
				return []*mock.Call{
					mockedClient.On("Get", mock.Anything, "host-12345678").
						Return(&inventory.GetResourceResponse{
							Resource: &inventory.Resource{
								Resource: &inventory.Resource_Host{
									Host: exampleInvHostResource,
								},
							},
						}, nil).Once(),
				}
			},
			ctx: context.Background(),
			req: &restv1.GetHostRequest{
				ResourceId: "host-12345678",
			},
			wantErr: false,
		},
		{
			name: "Get Host with error",
			mocks: func() []*mock.Call {
				return []*mock.Call{
					mockedClient.On("Get", mock.Anything, "host-12345678").
						Return(nil, errors.New("error")).Once(),
				}
			},
			ctx: context.Background(),
			req: &restv1.GetHostRequest{
				ResourceId: "host-12345678",
			},
			wantErr: true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.mocks != nil {
				tc.mocks()
			}

			reply, err := server.GetHost(tc.ctx, tc.req)
			if tc.wantErr {
				if err == nil {
					t.Errorf("GetHost() got err = nil, want err")
				}
				return
			}
			if err != nil {
				t.Errorf("GetHost() got err = %v, want nil", err)
				return
			}
			if reply == nil {
				t.Errorf("GetHost() got reply = nil, want non-nil")
				return
			}
			compareProtoMessages(t, exampleAPIHostResource, reply)
		})
	}
}

func TestHost_List(t *testing.T) {
	mockedClient := newMockedInventoryTestClient()
	server := inv_server.InventorygRPCServer{InvClient: mockedClient}

	cases := []struct {
		name    string
		mocks   func() []*mock.Call
		ctx     context.Context
		req     *restv1.ListHostsRequest
		wantErr bool
	}{
		{
			name: "List Hosts",
			mocks: func() []*mock.Call {
				return []*mock.Call{
					mockedClient.On("List", mock.Anything, mock.Anything).
						Return(&inventory.ListResourcesResponse{
							Resources: []*inventory.GetResourceResponse{
								{
									Resource: &inventory.Resource{
										Resource: &inventory.Resource_Host{
											Host: exampleInvHostResource,
										},
									},
								},
							},
							TotalElements: 1,
							HasNext:       false,
						}, nil).Once(),
				}
			},
			ctx: context.Background(),
			req: &restv1.ListHostsRequest{
				PageSize: 10,
				Offset:   0,
			},
			wantErr: false,
		},
		{
			name: "List Hosts with error",
			mocks: func() []*mock.Call {
				return []*mock.Call{
					mockedClient.On("List", mock.Anything, mock.Anything).
						Return(nil, errors.New("error")).Once(),
				}
			},
			ctx: context.Background(),
			req: &restv1.ListHostsRequest{
				PageSize: 10,
				Offset:   0,
			},
			wantErr: true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.mocks != nil {
				tc.mocks()
			}

			reply, err := server.ListHosts(tc.ctx, tc.req)
			if tc.wantErr {
				if err == nil {
					t.Errorf("ListHosts() got err = nil, want err")
				}
				return
			}
			if err != nil {
				t.Errorf("ListHosts() got err = %v, want nil", err)
				return
			}
			if reply == nil {
				t.Errorf("ListHosts() got reply = nil, want non-nil")
				return
			}
			if len(reply.GetHosts()) != 1 {
				t.Errorf("ListHosts() got %v hosts, want 1", len(reply.GetHosts()))
			}
			compareProtoMessages(t, exampleAPIHostResource, reply.GetHosts()[0])
		})
	}
}

func TestHost_Update(t *testing.T) {
	mockedClient := newMockedInventoryTestClient()
	server := inv_server.InventorygRPCServer{InvClient: mockedClient}

	cases := []struct {
		name    string
		mocks   func() []*mock.Call
		ctx     context.Context
		req     *restv1.UpdateHostRequest
		wantErr bool
	}{
		{
			name: "Update Host",
			mocks: func() []*mock.Call {
				return []*mock.Call{
					mockedClient.On("Update", mock.Anything, "host-12345678", mock.Anything, mock.Anything).
						Return(&inventory.Resource{
							Resource: &inventory.Resource_Host{
								Host: exampleInvHostResource,
							},
						}, nil).Once(),
				}
			},
			ctx: context.Background(),
			req: &restv1.UpdateHostRequest{
				ResourceId: "host-12345678",
				Host:       exampleAPIHostResource,
			},
			wantErr: false,
		},
		{
			name: "Update Host with error",
			mocks: func() []*mock.Call {
				return []*mock.Call{
					mockedClient.On("Update", mock.Anything, "host-12345678", mock.Anything, mock.Anything).
						Return(nil, errors.New("error")).Once(),
				}
			},
			ctx: context.Background(),
			req: &restv1.UpdateHostRequest{
				ResourceId: "host-12345678",
				Host:       exampleAPIHostResource,
			},
			wantErr: true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.mocks != nil {
				tc.mocks()
			}

			reply, err := server.UpdateHost(tc.ctx, tc.req)
			if tc.wantErr {
				if err == nil {
					t.Errorf("UpdateHost() got err = nil, want err")
				}
				return
			}
			if err != nil {
				t.Errorf("UpdateHost() got err = %v, want nil", err)
				return
			}
			if reply == nil {
				t.Errorf("UpdateHost() got reply = nil, want non-nil")
				return
			}
			compareProtoMessages(t, exampleAPIHostResource, reply)
		})
	}
}

func TestHost_Delete(t *testing.T) {
	mockedClient := newMockedInventoryTestClient()
	server := inv_server.InventorygRPCServer{InvClient: mockedClient}

	cases := []struct {
		name    string
		mocks   func() []*mock.Call
		ctx     context.Context
		req     *restv1.DeleteHostRequest
		wantErr bool
	}{
		{
			name: "Delete Host",
			mocks: func() []*mock.Call {
				return []*mock.Call{
					mockedClient.On("Delete", mock.Anything, "host-12345678").
						Return(&inventory.DeleteResourceResponse{}, nil).Once(),
				}
			},
			ctx: context.Background(),
			req: &restv1.DeleteHostRequest{
				ResourceId: "host-12345678",
			},
			wantErr: false,
		},
		{
			name: "Delete Host with error",
			mocks: func() []*mock.Call {
				return []*mock.Call{
					mockedClient.On("Delete", mock.Anything, "host-12345678").
						Return(nil, errors.New("error")).Once(),
				}
			},
			ctx: context.Background(),
			req: &restv1.DeleteHostRequest{
				ResourceId: "host-12345678",
			},
			wantErr: true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.mocks != nil {
				tc.mocks()
			}

			reply, err := server.DeleteHost(tc.ctx, tc.req)
			if tc.wantErr {
				if err == nil {
					t.Errorf("DeleteHost() got err = nil, want err")
				}
				return
			}
			if err != nil {
				t.Errorf("DeleteHost() got err = %v, want nil", err)
				return
			}
			if reply == nil {
				t.Errorf("DeleteHost() got reply = nil, want non-nil")
				return
			}
		})
	}
}
