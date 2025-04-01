// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package store_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	"github.com/open-edge-platform/infra-core/inventory/v2/internal/ent/tenant"
	inv_v1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/inventory/v1"
	tenantv1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/tenant/v1"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/client"
	inv_testing "github.com/open-edge-platform/infra-core/inventory/v2/pkg/testing"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/util"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/util/collections"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/util/filters"
)

func TestTenant(t *testing.T) {
	suite.Run(t, &tenantTestSuite{
		InvResourceDAO: inv_testing.NewInvResourceDAOOrFail(t),
	})
}

type tenantTestSuite struct {
	suite.Suite
	*inv_testing.InvResourceDAO
}

func (ts *tenantTestSuite) GetAPIClient() client.TenantAwareInventoryClient {
	panic("DO NOT USE API CLIENT")
}

func (ts *tenantTestSuite) GetRMClient() client.TenantAwareInventoryClient {
	panic("DO NOT USE RM CLIENT")
}

func (ts *tenantTestSuite) SuiteConfig() {
	ts.Require().Panics(func() {
		ts.GetAPIClient()
	})
	ts.Require().Panics(func() {
		ts.GetRMClient()
	})
}

func (ts *tenantTestSuite) TestInvalidCreationRequests() {
	tcs := []struct {
		name      string
		req       *tenantv1.Tenant
		errorCode codes.Code
	}{
		{
			name: "missing tenantID",
			req: &tenantv1.Tenant{
				DesiredState: tenantv1.TenantState_TENANT_STATE_CREATED,
			},
			errorCode: codes.InvalidArgument,
		},
		{
			name: "missing desiredState",
			req: &tenantv1.Tenant{
				TenantId: uuid.NewString(),
			},
			errorCode: codes.InvalidArgument,
		},
		{
			name: "bad resourceID is specified",
			req: &tenantv1.Tenant{
				TenantId:   uuid.NewString(),
				ResourceId: "dudu",
			},
			errorCode: codes.InvalidArgument,
		},
		{
			name: "resourceID is specified",
			req: &tenantv1.Tenant{
				TenantId:   uuid.NewString(),
				ResourceId: "tenant-12345678",
			},
			errorCode: codes.InvalidArgument,
		},
	}

	for _, tc := range tcs {
		ts.Run(tc.name, func() {
			res, err := util.WrapResource(tc.req)
			ts.Require().NoError(err)
			ts.Require().NotNil(res)

			rsp, err := ts.GetTCClient().Create(context.TODO(), tc.req.GetTenantId(), res)
			ts.Require().Error(err)
			ts.Require().Equal(tc.errorCode, status.Code(err))
			ts.Require().Nil(rsp)
		})
	}
}

func (ts *tenantTestSuite) TestCreateWithApiAndRMClients() {
	tenantID := uuid.NewString()
	res, err := util.WrapResource(&tenantv1.Tenant{
		DesiredState: tenantv1.TenantState_TENANT_STATE_CREATED,
		TenantId:     tenantID,
	})
	ts.Require().NoError(err)

	dao := inv_testing.NewInvResourceDAOOrFail(ts.T())

	_, err = dao.GetAPIClient().Create(context.TODO(), tenantID, res)
	ts.Require().Errorf(err, "API client shall not be able to create tenant")

	_, err = dao.GetRMClient().Create(context.TODO(), tenantID, res)
	ts.Require().Errorf(err, "RM client shall not be able to create tenant")
}

func (ts *tenantTestSuite) TestLifecycle() {
	tenantID := uuid.NewString()

	var tenantInstance *tenantv1.Tenant
	ts.Run("create", func() {
		tenantInstance = ts.CreateTenantWithOpts(
			ts.T(),
			tenantID,
			false,
			inv_testing.TenantDesiredState(tenantv1.TenantState_TENANT_STATE_CREATED),
		)
		// get created tenant
		getRsp, err := ts.GetTCClient().Get(context.TODO(), tenantInstance.GetTenantId(), tenantInstance.GetResourceId())
		ts.Require().NoError(err)
		ts.Require().NotEmpty(getRsp)
	})

	ts.Run("update", func() {
		updRsp, err := ts.GetTCClient().Update(
			context.TODO(),
			tenantInstance.GetTenantId(),
			tenantInstance.GetResourceId(),
			&fieldmaskpb.FieldMask{Paths: []string{tenant.FieldWatcherOsmanager}},
			&inv_v1.Resource{
				Resource: &inv_v1.Resource_Tenant{
					Tenant: &tenantv1.Tenant{
						WatcherOsmanager: true,
					},
				},
			},
		)
		ts.Require().NoError(err)
		ts.Require().NotNil(updRsp)

		// confirm requested update
		rsp, err := ts.GetTCClient().Get(context.TODO(), tenantInstance.GetTenantId(), tenantInstance.GetResourceId())
		ts.Require().NoError(err)
		ts.Require().NotEmpty(rsp)
		ts.Require().NotNil(rsp.GetResource().GetTenant())
		ts.Require().Equal(true, rsp.GetResource().GetTenant().GetWatcherOsmanager())
	})

	ts.Run("update current state", func() {
		updRsp, err := ts.GetTCClient().Update(
			context.TODO(),
			tenantInstance.GetTenantId(),
			tenantInstance.GetResourceId(),
			&fieldmaskpb.FieldMask{Paths: []string{tenant.FieldCurrentState}},
			&inv_v1.Resource{
				Resource: &inv_v1.Resource_Tenant{
					Tenant: &tenantv1.Tenant{
						CurrentState: tenantv1.TenantState_TENANT_STATE_CREATED,
					},
				},
			},
		)
		ts.Require().NoError(err)
		ts.Require().NotNil(updRsp)
	})

	ts.Run("soft delete", func() {
		// soft delete tenant
		ts.DeleteResource(ts.T(), tenantInstance.GetTenantId(), tenantInstance.GetResourceId())
		// confirm soft deletion
		getRsp, err := ts.GetTCClient().Get(context.TODO(), tenantInstance.GetTenantId(), tenantInstance.GetResourceId())
		ts.Require().NoError(err)
		ts.Require().NotEmpty(getRsp)
		ts.Require().NotNil(getRsp.GetResource().GetTenant())
		ts.Require().Equal(tenantv1.TenantState_TENANT_STATE_DELETED, getRsp.GetResource().GetTenant().GetDesiredState())
	})

	ts.Run("hard delete", func() {
		// RM reports deletion
		updRsp, err := ts.GetTCClient().Update(
			context.TODO(),
			tenantInstance.GetTenantId(),
			tenantInstance.GetResourceId(),
			&fieldmaskpb.FieldMask{Paths: []string{tenant.FieldCurrentState}},
			&inv_v1.Resource{
				Resource: &inv_v1.Resource_Tenant{
					Tenant: &tenantv1.Tenant{
						CurrentState: tenantv1.TenantState_TENANT_STATE_DELETED,
					},
				},
			},
		)
		ts.Require().NoError(err)
		ts.Require().NotNil(updRsp)
		// confirm tenant deletion
		getRsp, err := ts.GetTCClient().Get(context.TODO(), tenantInstance.GetTenantId(), tenantInstance.GetResourceId())
		ts.Require().Error(err)
		ts.Require().Nil(getRsp)
	})
}

func (ts *tenantTestSuite) TestList() {
	tenants := []*tenantv1.Tenant{
		ts.CreateTenantWithOpts(ts.T(), uuid.NewString(), true,
			inv_testing.TenantDesiredStateCreated(), inv_testing.TenantCurrentStateDeleted()),
		ts.CreateTenantWithOpts(ts.T(), uuid.NewString(), true,
			inv_testing.TenantDesiredStateCreated(), inv_testing.TenantCurrentStateDeleted()),
		ts.CreateTenantWithOpts(ts.T(), uuid.NewString(), true,
			inv_testing.TenantDesiredStateDeleted(), inv_testing.TenantCurrentStateCreated()),
		ts.CreateTenantWithOpts(ts.T(), uuid.NewString(), true, inv_testing.TenantDesiredStateCreated(),
			inv_testing.TenantCurrentStateCreated(), inv_testing.TenantWatcherOSManager(true)),
	}

	ts.T().Run("ListAll", func(t *testing.T) {
		res, err := ts.GetTCClient().ListAll(
			context.TODO(),
			&inv_v1.ResourceFilter{Resource: &inv_v1.Resource{Resource: &inv_v1.Resource_Tenant{}}})
		require.NoError(t, err)
		require.Len(t, res, len(tenants))
	})

	tcs := []struct {
		name            string
		filter          *inv_v1.ResourceFilter
		expectedTenants int
	}{
		{
			name:            "filter:all",
			filter:          &inv_v1.ResourceFilter{Resource: &inv_v1.Resource{Resource: &inv_v1.Resource_Tenant{}}},
			expectedTenants: len(tenants),
		},
		{
			name:            "filter:all,limit: 1",
			filter:          &inv_v1.ResourceFilter{Resource: &inv_v1.Resource{Resource: &inv_v1.Resource_Tenant{}}, Limit: 1},
			expectedTenants: 1,
		},
		{
			name:            "filter:all,limit: 2",
			filter:          &inv_v1.ResourceFilter{Resource: &inv_v1.Resource{Resource: &inv_v1.Resource_Tenant{}}, Limit: 2},
			expectedTenants: 2,
		},
		{
			name: "filter:desired_state eq CREATED",
			filter: &inv_v1.ResourceFilter{
				Resource: &inv_v1.Resource{Resource: &inv_v1.Resource_Tenant{}},
				Filter: filters.NewBuilderWith(
					filters.ValEq(tenant.FieldDesiredState, tenantv1.TenantState_TENANT_STATE_CREATED),
				).Build(),
			},
			expectedTenants: len(collections.Filter(tenants, func(t *tenantv1.Tenant) bool {
				return t.DesiredState == tenantv1.TenantState_TENANT_STATE_CREATED
			})),
		},
		{
			name: "filter:current_state eq DELETED",
			filter: &inv_v1.ResourceFilter{
				Resource: &inv_v1.Resource{Resource: &inv_v1.Resource_Tenant{}},
				Filter: filters.NewBuilderWith(
					filters.ValEq(tenant.FieldCurrentState, tenantv1.TenantState_TENANT_STATE_DELETED),
				).Build(),
			},
			expectedTenants: len(collections.Filter(tenants, func(t *tenantv1.Tenant) bool {
				return t.CurrentState == tenantv1.TenantState_TENANT_STATE_DELETED
			})),
		},
		{
			name: "filter:current_state eq CREATED and desired_state eq DELETED",
			filter: &inv_v1.ResourceFilter{
				Resource: &inv_v1.Resource{Resource: &inv_v1.Resource_Tenant{}},
				Filter: filters.NewBuilderWith(
					filters.ValEq(tenant.FieldCurrentState, tenantv1.TenantState_TENANT_STATE_CREATED),
				).And(filters.ValEq(tenant.FieldDesiredState, tenantv1.TenantState_TENANT_STATE_CREATED)).Build(),
			},
			expectedTenants: len(collections.Filter(tenants, func(t *tenantv1.Tenant) bool {
				return t.CurrentState == tenantv1.TenantState_TENANT_STATE_CREATED &&
					t.DesiredState == tenantv1.TenantState_TENANT_STATE_CREATED
			})),
		},
	}

	for _, tc := range tcs {
		ts.T().Run(tc.name, func(t *testing.T) {
			res, err := ts.GetTCClient().List(context.TODO(), tc.filter)
			require.NoError(t, err)
			require.Len(t, res.Resources, tc.expectedTenants)
		})
	}
}
