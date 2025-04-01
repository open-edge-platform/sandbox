// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"context"
	"fmt"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	computev1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/compute/v1"
	inventory "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/inventory/v1"
	telemetryv1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/telemetry/v1"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/client"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/client/cache"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/errors"
)

type MockResponses struct {
	FindResourcesResponse    *inventory.FindResourcesResponse
	FindAllResourcesResponse []string
	ListResourcesResponse    *inventory.ListResourcesResponse
	DeleteResourceResponse   *inventory.DeleteResourceResponse
	GetResourceResponse      *inventory.GetResourceResponse
	UpdateResourceResponse   *inventory.Resource
	CreateResourceResponse   *inventory.Resource
}

type ResourceMaskTuple struct {
	mask     *fieldmaskpb.FieldMask
	resource *inventory.Resource
}

type MockInventoryServiceClient struct {
	FindResourcesCallCount int
	FindResourcesCalls     []*inventory.ResourceFilter
	FindResourcesResponse  *inventory.FindResourcesResponse

	FindAllResourcesCallCount int
	FindAllResourcesCalls     []*inventory.ResourceFilter
	FindAllResourcesResponse  []*inventory.FindResourcesResponse_ResourceTenantIDCarrier

	ListAllResourcesCallCount int
	ListAllResourcesCalls     []*inventory.ResourceFilter
	ListAllResourcesResponse  []*inventory.Resource

	ListResourcesCallCount int
	ListResourcesLimit     uint32
	ListResourcesOffset    uint32
	ListResourcesOrderBy   string
	ListResourcesFilter    string
	ListResourcesCalls     []*inventory.ResourceFilter
	ListResourcesResponse  *inventory.ListResourcesResponse

	DeleteResourceCallCount int
	DeleteResourceResponse  *inventory.DeleteResourceResponse

	DeleteAllResourcesCallCount int

	GetResourceCallCount int
	GetResourceResponse  *inventory.GetResourceResponse

	UpdateResourceCallCount int
	UpdateResourceResponse  *inventory.Resource

	CreateResourceCallCount int
	CreateResourceResponse  *inventory.Resource

	LastUpdateResourceRequest          string
	LastUpdateResourceRequestFieldMask *fieldmaskpb.FieldMask
}

func (m *MockInventoryServiceClient) GetTenantAwareInventoryClient() client.TenantAwareInventoryClient {
	// TODO implement me
	panic("implement me")
}

func (m *MockInventoryServiceClient) ListAll(ctx context.Context, filter *inventory.ResourceFilter) ([]*inventory.Resource, error) {
	m.ListAllResourcesCallCount += 1
	m.ListAllResourcesCalls = append(m.ListAllResourcesCalls, filter)

	resp := []*inventory.Resource{
		{
			Resource: &inventory.Resource_Host{Host: hostResource},
		},
		{
			Resource: &inventory.Resource_Hoststorage{Hoststorage: hostResourceStorage},
		},
		{
			Resource: &inventory.Resource_Hostnic{Hostnic: hostResourceNic},
		},
		{
			Resource: &inventory.Resource_Hostgpu{Hostgpu: hostResourceGpu},
		},
		{
			Resource: &inventory.Resource_Ipaddress{Ipaddress: ipAddressResource},
		},
		{
			Resource: &inventory.Resource_Hostusb{Hostusb: hostResourceUsb},
		},
		{
			Resource: &inventory.Resource_Region{Region: regionResource},
		},
		{
			Resource: &inventory.Resource_Site{Site: siteResource},
		},
		{
			Resource: &inventory.Resource_Ou{Ou: ouResource},
		},
		{
			Resource: &inventory.Resource_Os{Os: osrResource},
		},
		{
			Resource: &inventory.Resource_Singleschedule{Singleschedule: singlescheduleResource},
		},
		{
			Resource: &inventory.Resource_Repeatedschedule{Repeatedschedule: repeatedscheduleResource},
		},
		{
			Resource: &inventory.Resource_Workload{Workload: workloadResource},
		},
		{
			Resource: &inventory.Resource_WorkloadMember{WorkloadMember: workloadMember},
		},
		{
			Resource: &inventory.Resource_Instance{Instance: instanceResource},
		},
		{
			Resource: &inventory.Resource_TelemetryGroup{TelemetryGroup: telemetryGroupResource},
		},
		{
			Resource: &inventory.Resource_TelemetryProfile{TelemetryProfile: telemetryProfile},
		},
	}

	m.ListAllResourcesResponse = resp
	return m.ListAllResourcesResponse, nil
}

func (m *MockInventoryServiceClient) FindAll(ctx context.Context, filter *inventory.ResourceFilter) (
	[]*inventory.FindResourcesResponse_ResourceTenantIDCarrier, error,
) {
	m.FindAllResourcesCallCount += 1
	m.FindAllResourcesCalls = append(m.FindAllResourcesCalls, filter)
	return m.FindAllResourcesResponse, nil
}

func (m *MockInventoryServiceClient) TestingOnlySetClient(c inventory.InventoryServiceClient) {
	panic("implement me")
}

func (m *MockInventoryServiceClient) TestGetClientCache() *cache.InventoryCache {
	panic("implement me")
}

func (m *MockInventoryServiceClient) Create(
	ctx context.Context,
	in *inventory.Resource,
) (*inventory.Resource, error) {
	m.CreateResourceCallCount += 1
	m.CreateResourceResponse = new(inventory.Resource)
	return m.CreateResourceResponse, nil
}

func (m *MockInventoryServiceClient) Find(
	ctx context.Context,
	in *inventory.ResourceFilter,
) (*inventory.FindResourcesResponse, error) {
	m.FindResourcesCallCount += 1
	m.FindResourcesCalls = append(m.FindResourcesCalls, in)
	return m.FindResourcesResponse, nil
}

func (m *MockInventoryServiceClient) Get(ctx context.Context, in string) (*inventory.GetResourceResponse, error) {
	m.GetResourceCallCount += 1
	resp := inventory.GetResourceResponse{
		RenderedMetadata: &inventory.GetResourceResponse_ResourceMetadata{
			PhyMetadata:  metadataResource,
			LogiMetadata: metadataResource,
		},
	}

	if strings.Contains(in, "os") {
		osrResource.ResourceId = in
		resp.Resource = &inventory.Resource{
			Resource: &inventory.Resource_Os{
				Os: osrResource,
			},
		}
	}
	if strings.Contains(in, "region") {
		regionResource.ResourceId = in
		resp.Resource = &inventory.Resource{
			Resource: &inventory.Resource_Region{
				Region: regionResource,
			},
		}
	}
	if strings.Contains(in, "site") {
		siteResource.ResourceId = in
		resp.Resource = &inventory.Resource{
			Resource: &inventory.Resource_Site{
				Site: siteResource,
			},
		}
	}
	if strings.Contains(in, "ou") {
		ouResource.ResourceId = in
		resp.Resource = &inventory.Resource{
			Resource: &inventory.Resource_Ou{
				Ou: ouResource,
			},
		}
	}
	if strings.Contains(in, "single") {
		singlescheduleResource.ResourceId = in
		resp.Resource = &inventory.Resource{
			Resource: &inventory.Resource_Singleschedule{
				Singleschedule: singlescheduleResource,
			},
		}
	}
	if strings.Contains(in, "repeated") {
		repeatedscheduleResource.ResourceId = in
		resp.Resource = &inventory.Resource{
			Resource: &inventory.Resource_Repeatedschedule{
				Repeatedschedule: repeatedscheduleResource,
			},
		}
	}
	if strings.Contains(in, "host") {
		hostResource.ResourceId = in
		resp.Resource = &inventory.Resource{
			Resource: &inventory.Resource_Host{
				Host: hostResource,
			},
		}
	}
	if strings.Contains(in, "nic") {
		hostResourceNic.ResourceId = in
		resp.Resource = &inventory.Resource{
			Resource: &inventory.Resource_Hostnic{
				Hostnic: hostResourceNic,
			},
		}
	}
	if strings.Contains(in, "ipaddr") {
		ipAddressResource.ResourceId = in
		resp.Resource = &inventory.Resource{
			Resource: &inventory.Resource_Ipaddress{
				Ipaddress: ipAddressResource,
			},
		}
	}
	if strings.Contains(in, "usb") {
		hostResourceUsb.ResourceId = in
		resp.Resource = &inventory.Resource{
			Resource: &inventory.Resource_Hostusb{
				Hostusb: hostResourceUsb,
			},
		}
	}
	if strings.Contains(in, "storage") {
		hostResourceStorage.ResourceId = in
		resp.Resource = &inventory.Resource{
			Resource: &inventory.Resource_Hoststorage{
				Hoststorage: hostResourceStorage,
			},
		}
	}
	if strings.Contains(in, "gpu") {
		hostResourceGpu.ResourceId = in
		resp.Resource = &inventory.Resource{
			Resource: &inventory.Resource_Hostgpu{
				Hostgpu: hostResourceGpu,
			},
		}
	}
	if strings.Contains(in, "workload") {
		workloadResource.ResourceId = in
		resp.Resource = &inventory.Resource{
			Resource: &inventory.Resource_Workload{
				Workload: workloadResource,
			},
		}
	}
	if strings.Contains(in, "workloadmember") {
		workloadMember.ResourceId = in
		resp.Resource = &inventory.Resource{
			Resource: &inventory.Resource_WorkloadMember{
				WorkloadMember: workloadMember,
			},
		}
	}
	if strings.Contains(in, "inst") {
		instanceResource.ResourceId = in
		resp.Resource = &inventory.Resource{
			Resource: &inventory.Resource_Instance{
				Instance: instanceResource,
			},
		}
	}
	if strings.Contains(in, "telemetrygroup") {
		telemetryGroupResource.ResourceId = in
		resp.Resource = &inventory.Resource{
			Resource: &inventory.Resource_TelemetryGroup{
				TelemetryGroup: telemetryGroupResource,
			},
		}
	}
	if strings.Contains(in, "telemetryprofile") {
		telemetryProfile.ResourceId = in
		resp.Resource = &inventory.Resource{
			Resource: &inventory.Resource_TelemetryProfile{
				TelemetryProfile: telemetryProfile,
			},
		}
	}

	if strings.Contains(in, "provider") {
		providerResource.ResourceId = in
		resp.Resource = &inventory.Resource{
			Resource: &inventory.Resource_Provider{
				Provider: providerResource,
			},
		}
	}

	m.GetResourceResponse = &resp
	return m.GetResourceResponse, nil
}

func (m *MockInventoryServiceClient) Update(
	ctx context.Context,
	resID string,
	fm *fieldmaskpb.FieldMask,
	in *inventory.Resource,
) (*inventory.Resource, error) {
	m.UpdateResourceCallCount += 1
	m.LastUpdateResourceRequest = resID
	m.LastUpdateResourceRequestFieldMask = fm
	return m.UpdateResourceResponse, nil
}

func (m *MockInventoryServiceClient) Delete(ctx context.Context, in string) (*inventory.DeleteResourceResponse, error) {
	m.DeleteResourceCallCount += 1
	return m.DeleteResourceResponse, nil
}

func (m *MockInventoryServiceClient) DeleteAllResources(ctx context.Context, in string, kind inventory.ResourceKind) error {
	m.DeleteAllResourcesCallCount += 1
	return nil
}

func (m *MockInventoryServiceClient) List(
	ctx context.Context,
	in *inventory.ResourceFilter,
) (*inventory.ListResourcesResponse, error) {
	m.ListResourcesCallCount += 1
	m.ListResourcesCalls = append(m.ListResourcesCalls, in)
	m.ListResourcesLimit = in.Limit
	m.ListResourcesOffset = in.Offset
	m.ListResourcesOrderBy = in.OrderBy
	m.ListResourcesFilter = in.Filter

	resp := &inventory.ListResourcesResponse{}

	var err error

	switch in.GetResource().GetResource().(type) {
	case *inventory.Resource_Host:
		resp = &inventory.ListResourcesResponse{
			Resources: []*inventory.GetResourceResponse{
				{
					Resource: &inventory.Resource{
						Resource: &inventory.Resource_Host{Host: hostResource},
					},
					RenderedMetadata: &inventory.GetResourceResponse_ResourceMetadata{
						PhyMetadata:  metadataResource,
						LogiMetadata: metadataResource,
					},
				},
			},
		}
	case *inventory.Resource_Hoststorage:
		resp = &inventory.ListResourcesResponse{
			Resources: []*inventory.GetResourceResponse{
				{
					Resource: &inventory.Resource{
						Resource: &inventory.Resource_Hoststorage{Hoststorage: hostResourceStorage},
					},
					RenderedMetadata: &inventory.GetResourceResponse_ResourceMetadata{
						PhyMetadata:  metadataResource,
						LogiMetadata: metadataResource,
					},
				},
			},
		}
	case *inventory.Resource_Hostnic:
		resp = &inventory.ListResourcesResponse{
			Resources: []*inventory.GetResourceResponse{
				{
					Resource: &inventory.Resource{
						Resource: &inventory.Resource_Hostnic{Hostnic: hostResourceNic},
					},
					RenderedMetadata: &inventory.GetResourceResponse_ResourceMetadata{
						PhyMetadata:  metadataResource,
						LogiMetadata: metadataResource,
					},
				},
			},
		}
	case *inventory.Resource_Hostgpu:
		resp = &inventory.ListResourcesResponse{
			Resources: []*inventory.GetResourceResponse{
				{
					Resource: &inventory.Resource{
						Resource: &inventory.Resource_Hostgpu{Hostgpu: hostResourceGpu},
					},
					RenderedMetadata: &inventory.GetResourceResponse_ResourceMetadata{
						PhyMetadata:  metadataResource,
						LogiMetadata: metadataResource,
					},
				},
			},
		}
	case *inventory.Resource_Ipaddress:
		if strings.Contains(in.GetResource().GetIpaddress().GetNic().GetResourceId(), hostResourceNic.GetResourceId()) {
			resp = &inventory.ListResourcesResponse{
				Resources: []*inventory.GetResourceResponse{
					{
						Resource: &inventory.Resource{
							Resource: &inventory.Resource_Ipaddress{Ipaddress: ipAddressResource},
						},
						RenderedMetadata: &inventory.GetResourceResponse_ResourceMetadata{
							PhyMetadata:  metadataResource,
							LogiMetadata: metadataResource,
						},
					},
				},
			}
		} else {
			err = errors.Errorfc(codes.NotFound, "Not found")
		}
	case *inventory.Resource_Hostusb:
		resp = &inventory.ListResourcesResponse{
			Resources: []*inventory.GetResourceResponse{
				{
					Resource: &inventory.Resource{
						Resource: &inventory.Resource_Hostusb{Hostusb: hostResourceUsb},
					},
					RenderedMetadata: &inventory.GetResourceResponse_ResourceMetadata{
						PhyMetadata:  metadataResource,
						LogiMetadata: metadataResource,
					},
				},
			},
		}
	case *inventory.Resource_Region:
		resp = &inventory.ListResourcesResponse{
			Resources: []*inventory.GetResourceResponse{
				{
					Resource: &inventory.Resource{
						Resource: &inventory.Resource_Region{Region: regionResource},
					},
					RenderedMetadata: &inventory.GetResourceResponse_ResourceMetadata{
						PhyMetadata:  metadataResource,
						LogiMetadata: metadataResource,
					},
				},
			},
		}
	case *inventory.Resource_Site:
		resp = &inventory.ListResourcesResponse{
			Resources: []*inventory.GetResourceResponse{
				{
					Resource: &inventory.Resource{
						Resource: &inventory.Resource_Site{Site: siteResource},
					},
					RenderedMetadata: &inventory.GetResourceResponse_ResourceMetadata{
						PhyMetadata:  metadataResource,
						LogiMetadata: metadataResource,
					},
				},
			},
		}
	case *inventory.Resource_Ou:
		resp = &inventory.ListResourcesResponse{
			Resources: []*inventory.GetResourceResponse{
				{
					Resource: &inventory.Resource{
						Resource: &inventory.Resource_Ou{Ou: ouResource},
					},
					RenderedMetadata: &inventory.GetResourceResponse_ResourceMetadata{
						PhyMetadata:  metadataResource,
						LogiMetadata: metadataResource,
					},
				},
			},
		}
	case *inventory.Resource_Os:
		resp = &inventory.ListResourcesResponse{
			Resources: []*inventory.GetResourceResponse{
				{
					Resource: &inventory.Resource{
						Resource: &inventory.Resource_Os{Os: osrResource},
					},
					RenderedMetadata: &inventory.GetResourceResponse_ResourceMetadata{
						PhyMetadata:  metadataResource,
						LogiMetadata: metadataResource,
					},
				},
			},
		}
	case *inventory.Resource_Singleschedule:
		singlescheduleResource.ResourceId = SingleSchedResID
		resp = &inventory.ListResourcesResponse{
			Resources: []*inventory.GetResourceResponse{
				{
					Resource: &inventory.Resource{
						Resource: &inventory.Resource_Singleschedule{Singleschedule: singlescheduleResource},
					},
					RenderedMetadata: &inventory.GetResourceResponse_ResourceMetadata{
						PhyMetadata:  metadataResource,
						LogiMetadata: metadataResource,
					},
				},
			},
		}
	case *inventory.Resource_Repeatedschedule:
		repeatedscheduleResource.ResourceId = RepeatedSchedResID
		resp = &inventory.ListResourcesResponse{
			Resources: []*inventory.GetResourceResponse{
				{
					Resource: &inventory.Resource{
						Resource: &inventory.Resource_Repeatedschedule{Repeatedschedule: repeatedscheduleResource},
					},
					RenderedMetadata: &inventory.GetResourceResponse_ResourceMetadata{
						PhyMetadata:  metadataResource,
						LogiMetadata: metadataResource,
					},
				},
			},
		}
	case *inventory.Resource_Workload:
		repeatedscheduleResource.ResourceId = RepeatedSchedResID
		resp = &inventory.ListResourcesResponse{
			Resources: []*inventory.GetResourceResponse{
				{
					Resource: &inventory.Resource{
						Resource: &inventory.Resource_Workload{Workload: workloadResource},
					},
				},
			},
		}
	case *inventory.Resource_WorkloadMember:
		repeatedscheduleResource.ResourceId = RepeatedSchedResID
		resp = &inventory.ListResourcesResponse{
			Resources: []*inventory.GetResourceResponse{
				{
					Resource: &inventory.Resource{
						Resource: &inventory.Resource_WorkloadMember{WorkloadMember: workloadMember},
					},
				},
			},
		}
	case *inventory.Resource_Instance:
		repeatedscheduleResource.ResourceId = RepeatedSchedResID
		resp = &inventory.ListResourcesResponse{
			Resources: []*inventory.GetResourceResponse{
				{
					Resource: &inventory.Resource{
						Resource: &inventory.Resource_Instance{Instance: instanceResource},
					},
				},
			},
		}
	case *inventory.Resource_TelemetryGroup:
		resp = &inventory.ListResourcesResponse{
			Resources: []*inventory.GetResourceResponse{
				{
					Resource: &inventory.Resource{
						Resource: &inventory.Resource_TelemetryGroup{TelemetryGroup: telemetryGroupResource},
					},
				},
			},
		}
	case *inventory.Resource_TelemetryProfile:
		resp = &inventory.ListResourcesResponse{
			Resources: []*inventory.GetResourceResponse{
				{
					Resource: &inventory.Resource{
						Resource: &inventory.Resource_TelemetryProfile{TelemetryProfile: telemetryProfile},
					},
				},
			},
		}
	}

	m.ListResourcesResponse = resp
	return m.ListResourcesResponse, err
}

func (m *MockInventoryServiceClient) Close() error {
	return nil
}

func (m *MockInventoryServiceClient) UpdateSubscriptions(
	ctx context.Context,
	in []inventory.ResourceKind,
) error {
	// TODO: implement when needed
	return nil
}

func (m *MockInventoryServiceClient) ListInheritedTelemetryProfiles(
	ctx context.Context,
	inheritBy *inventory.ListInheritedTelemetryProfilesRequest_InheritBy,
	filter string,
	orderBy string,
	limit, offset uint32,
) (*inventory.ListInheritedTelemetryProfilesResponse, error) {
	return &inventory.ListInheritedTelemetryProfilesResponse{
		TelemetryProfiles: []*telemetryv1.TelemetryProfile{telemetryProfile},
		TotalElements:     1,
	}, nil
}

func (m *MockInventoryServiceClient) GetTreeHierarchy(
	ctx context.Context,
	getTreeReq *inventory.GetTreeHierarchyRequest,
) ([]*inventory.GetTreeHierarchyResponse_TreeNode, error) {
	return []*inventory.GetTreeHierarchyResponse_TreeNode{}, nil
}

func (m *MockInventoryServiceClient) GetSitesPerRegion(
	ctx context.Context,
	getReq *inventory.GetSitesPerRegionRequest,
) (*inventory.GetSitesPerRegionResponse, error) {
	return &inventory.GetSitesPerRegionResponse{}, nil
}

func (m *MockInventoryServiceClient) GetHostByUUID(ctx context.Context, uuid string) (*computev1.HostResource, error) {
	return hostResource, nil
}

func (m *MockInventoryServiceClient) TestGetClientCacheUUID() *cache.InventoryCache {
	panic("implement me")
}

func NewMockInventoryServiceClient(responses MockResponses) *MockInventoryServiceClient {
	return &MockInventoryServiceClient{
		FindResourcesCallCount: 0,
		FindResourcesCalls:     []*inventory.ResourceFilter{},
		FindResourcesResponse:  responses.FindResourcesResponse,

		ListResourcesCallCount: 0,
		ListResourcesCalls:     []*inventory.ResourceFilter{},
		ListResourcesResponse:  responses.ListResourcesResponse,

		DeleteResourceCallCount: 0,
		DeleteResourceResponse:  responses.DeleteResourceResponse,

		GetResourceCallCount: 0,
		GetResourceResponse:  responses.GetResourceResponse,

		UpdateResourceCallCount: 0,
		UpdateResourceResponse:  responses.UpdateResourceResponse,

		CreateResourceCallCount: 0,
		CreateResourceResponse:  responses.CreateResourceResponse,
	}
}

// Mock Inventory Client Errors

type MockInventoryServiceClientError struct{}

func (m *MockInventoryServiceClientError) GetTenantAwareInventoryClient() client.TenantAwareInventoryClient {
	// TODO implement me
	panic("implement me")
}

func (m *MockInventoryServiceClientError) ListAll(ctx context.Context, filter *inventory.ResourceFilter) ([]*inventory.Resource, error) {
	invErr := fmt.Errorf("inventory error")
	return nil, errors.Errorfc(codes.InvalidArgument, invErr.Error())
}

func (m *MockInventoryServiceClientError) FindAll(ctx context.Context, filter *inventory.ResourceFilter) (
	[]*inventory.FindResourcesResponse_ResourceTenantIDCarrier, error,
) {
	invErr := fmt.Errorf("inventory error")
	return nil, errors.Errorfc(codes.InvalidArgument, invErr.Error())
}

func NewMockInventoryServiceClientError() *MockInventoryServiceClientError {
	return &MockInventoryServiceClientError{}
}

func (m *MockInventoryServiceClientError) Close() error {
	return nil
}

func (m *MockInventoryServiceClientError) TestingOnlySetClient(c inventory.InventoryServiceClient) {
	panic("implement me")
}

func (m *MockInventoryServiceClientError) TestGetClientCache() *cache.InventoryCache {
	panic("implement me")
}

func (m *MockInventoryServiceClientError) Create(
	ctx context.Context,
	in *inventory.Resource,
) (*inventory.Resource, error) {
	invErr := fmt.Errorf("inventory error")
	return nil, errors.Errorfc(codes.InvalidArgument, invErr.Error())
}

func (m *MockInventoryServiceClientError) Find(
	ctx context.Context,
	in *inventory.ResourceFilter,
) (*inventory.FindResourcesResponse, error) {
	invErr := fmt.Errorf("inventory error")
	return nil, errors.Errorfc(codes.InvalidArgument, invErr.Error())
}

func (m *MockInventoryServiceClientError) Get(ctx context.Context, in string) (*inventory.GetResourceResponse, error) {
	invErr := fmt.Errorf("inventory error")
	return nil, errors.Errorfc(codes.InvalidArgument, invErr.Error())
}

func (m *MockInventoryServiceClientError) Update(
	ctx context.Context,
	resID string,
	fm *fieldmaskpb.FieldMask,
	in *inventory.Resource,
) (*inventory.Resource, error) {
	invErr := fmt.Errorf("inventory error")
	return nil, errors.Errorfc(codes.InvalidArgument, invErr.Error())
}

func (m *MockInventoryServiceClientError) Delete(ctx context.Context, in string) (*inventory.DeleteResourceResponse, error) {
	invErr := fmt.Errorf("inventory error")
	return nil, errors.Errorfc(codes.InvalidArgument, invErr.Error())
}

func (m *MockInventoryServiceClientError) DeleteAllResources(ctx context.Context, in string, kind inventory.ResourceKind, _ bool) error {
	invErr := fmt.Errorf("inventory error")
	return errors.Errorfc(codes.InvalidArgument, invErr.Error())
}

func (m *MockInventoryServiceClientError) List(
	ctx context.Context,
	in *inventory.ResourceFilter,
) (*inventory.ListResourcesResponse, error) {
	invErr := fmt.Errorf("inventory error")
	return nil, errors.Errorfc(codes.InvalidArgument, invErr.Error())
}

func (m *MockInventoryServiceClientError) UpdateSubscriptions(
	ctx context.Context,
	in []inventory.ResourceKind,
) error {
	invErr := fmt.Errorf("inventory error")
	return errors.Errorfc(codes.InvalidArgument, invErr.Error())
}

func (m *MockInventoryServiceClientError) ListInheritedTelemetryProfiles(
	ctx context.Context,
	inheritBy *inventory.ListInheritedTelemetryProfilesRequest_InheritBy,
	filter string,
	orderBy string,
	limit, offset uint32,
) (*inventory.ListInheritedTelemetryProfilesResponse, error) {
	invErr := fmt.Errorf("inventory error")
	return nil, errors.Errorfc(codes.InvalidArgument, invErr.Error())
}

func (m *MockInventoryServiceClientError) GetHostByUUID(ctx context.Context, uuid string) (*computev1.HostResource, error) {
	invErr := fmt.Errorf("inventory error")
	return nil, errors.Errorfc(codes.InvalidArgument, invErr.Error())
}

func (m *MockInventoryServiceClientError) TestGetClientCacheUUID() *cache.InventoryCache {
	panic("implement me")
}

func (m *MockInventoryServiceClientError) GetTreeHierarchy(
	ctx context.Context,
	getTreeReq *inventory.GetTreeHierarchyRequest,
) ([]*inventory.GetTreeHierarchyResponse_TreeNode, error) {
	return nil, errors.Errorfc(codes.InvalidArgument, "Not implemented")
}

func (m *MockInventoryServiceClientError) GetSitesPerRegion(
	ctx context.Context,
	getReq *inventory.GetSitesPerRegionRequest,
) (*inventory.GetSitesPerRegionResponse, error) {
	return nil, errors.Errorfc(codes.InvalidArgument, "Not implemented")
}

// TODO: implement proper mock.
type MockTenantAwareInventoryServiceClient struct {
	*MockInventoryServiceClient
}

func NewTenantAwareMockInventoryServiceClient(responses MockResponses) *MockTenantAwareInventoryServiceClient {
	return &MockTenantAwareInventoryServiceClient{
		MockInventoryServiceClient: NewMockInventoryServiceClient(responses),
	}
}

func (m *MockTenantAwareInventoryServiceClient) GetInventoryClient() *MockInventoryServiceClient {
	return m.MockInventoryServiceClient
}

func (m *MockTenantAwareInventoryServiceClient) List(ctx context.Context, in *inventory.ResourceFilter) (*inventory.ListResourcesResponse, error) {
	return m.MockInventoryServiceClient.List(ctx, in)
}

func (m *MockTenantAwareInventoryServiceClient) ListAll(ctx context.Context, in *inventory.ResourceFilter) ([]*inventory.Resource, error) {
	return m.MockInventoryServiceClient.ListAll(ctx, in)
}

func (m *MockTenantAwareInventoryServiceClient) Find(ctx context.Context, in *inventory.ResourceFilter) (*inventory.FindResourcesResponse, error) {
	return m.MockInventoryServiceClient.Find(ctx, in)
}

func (m *MockTenantAwareInventoryServiceClient) FindAll(ctx context.Context, in *inventory.ResourceFilter) (
	[]*inventory.FindResourcesResponse_ResourceTenantIDCarrier, error,
) {
	return m.MockInventoryServiceClient.FindAll(ctx, in)
}

func (m *MockTenantAwareInventoryServiceClient) Get(ctx context.Context, _, id string) (*inventory.GetResourceResponse, error) {
	return m.MockInventoryServiceClient.Get(ctx, id)
}

func (m *MockTenantAwareInventoryServiceClient) Create(ctx context.Context, _ string, res *inventory.Resource) (*inventory.Resource, error) {
	return m.MockInventoryServiceClient.Create(ctx, res)
}

func (m *MockTenantAwareInventoryServiceClient) Update(
	ctx context.Context, _, id string, fm *fieldmaskpb.FieldMask, res *inventory.Resource) (
	*inventory.Resource, error,
) {
	return m.MockInventoryServiceClient.Update(ctx, id, fm, res)
}

func (m *MockTenantAwareInventoryServiceClient) Delete(ctx context.Context, _, id string) (
	*inventory.DeleteResourceResponse, error,
) {
	return m.MockInventoryServiceClient.Delete(ctx, id)
}

func (m *MockTenantAwareInventoryServiceClient) DeleteAllResources(ctx context.Context, id string, kind inventory.ResourceKind, _ bool) error {
	return m.MockInventoryServiceClient.DeleteAllResources(ctx, id, kind)
}

func (m *MockTenantAwareInventoryServiceClient) UpdateSubscriptions(ctx context.Context, _ string, kinds []inventory.ResourceKind) error {
	return m.MockInventoryServiceClient.UpdateSubscriptions(ctx, kinds)
}

func (m *MockTenantAwareInventoryServiceClient) ListInheritedTelemetryProfiles(
	ctx context.Context,
	_ string,
	inheritBy *inventory.ListInheritedTelemetryProfilesRequest_InheritBy,
	filter string,
	orderBy string,
	limit, offset uint32,
) (*inventory.ListInheritedTelemetryProfilesResponse, error) {
	return m.MockInventoryServiceClient.ListInheritedTelemetryProfiles(ctx, inheritBy, filter, orderBy, limit, offset)
}

func (m *MockTenantAwareInventoryServiceClient) GetHostByUUID(ctx context.Context, _ string, uuid string) (*computev1.HostResource, error) {
	return m.MockInventoryServiceClient.GetHostByUUID(ctx, uuid)
}

func (m *MockTenantAwareInventoryServiceClient) GetTreeHierarchy(ctx context.Context, in *inventory.GetTreeHierarchyRequest) (
	[]*inventory.GetTreeHierarchyResponse_TreeNode, error,
) {
	return m.MockInventoryServiceClient.GetTreeHierarchy(ctx, in)
}

func (m *MockTenantAwareInventoryServiceClient) GetSitesPerRegion(ctx context.Context, in *inventory.GetSitesPerRegionRequest) (
	*inventory.GetSitesPerRegionResponse, error,
) {
	return m.MockInventoryServiceClient.GetSitesPerRegion(ctx, in)
}

func (m *MockTenantAwareInventoryServiceClient) TestingOnlySetClient(_ inventory.InventoryServiceClient) {
	panic("implement me")
}

func (m *MockTenantAwareInventoryServiceClient) TestGetClientCache() *cache.InventoryCache {
	panic("implement me")
}

func (m *MockTenantAwareInventoryServiceClient) TestGetClientCacheUUID() *cache.InventoryCache {
	panic("implement me")
}
