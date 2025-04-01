// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package store_test

import (
	"context"
	"fmt"
	"math/rand"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	"github.com/open-edge-platform/infra-core/inventory/v2/internal/ent/hostresource"
	"github.com/open-edge-platform/infra-core/inventory/v2/internal/ent/instanceresource"
	"github.com/open-edge-platform/infra-core/inventory/v2/internal/ent/remoteaccessconfiguration"
	computev1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/compute/v1"
	inv_v1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/inventory/v1"
	remoteaccessv1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/remoteaccess/v1"
	statusv1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/status/v1"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/client"
	inv_testing "github.com/open-edge-platform/infra-core/inventory/v2/pkg/testing"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/util"
)

func Test_RemoteAccessConfiguration_Lifecycle(t *testing.T) {
	os := inv_testing.CreateOs(t)

	host1 := inv_testing.CreateHost(t, nil, nil)
	instance1 := inv_testing.CreateInstance(t, host1, os)
	host2 := inv_testing.CreateHost(t, nil, nil)
	instance2 := inv_testing.CreateInstance(t, host2, os)

	testcases := map[string]struct {
		in     *remoteaccessv1.RemoteAccessConfiguration
		valid  bool
		errMsg string
	}{
		"CreateValidResource": {
			in: &remoteaccessv1.RemoteAccessConfiguration{
				DesiredState:        remoteaccessv1.RemoteAccessState_REMOTE_ACCESS_STATE_ENABLED,
				Instance:            instance1,
				ExpirationTimestamp: uint64(time.Now().Add(time.Minute * 15).Unix()),
			},
			valid: true,
		},
		"CreateBadResource_requested_remote_access_is_shorter_than_required_min": {
			in: &remoteaccessv1.RemoteAccessConfiguration{
				DesiredState:        remoteaccessv1.RemoteAccessState_REMOTE_ACCESS_STATE_ENABLED,
				Instance:            instance1,
				ExpirationTimestamp: uint64(time.Now().Add(time.Minute).Unix()),
			},
			valid:  false,
			errMsg: "remote access cannot be granted for less than",
		},
		"CreateBadResource_requested_remote_access_is_longer_than_required_max": {
			in: &remoteaccessv1.RemoteAccessConfiguration{
				DesiredState:        remoteaccessv1.RemoteAccessState_REMOTE_ACCESS_STATE_ENABLED,
				Instance:            instance1,
				ExpirationTimestamp: uint64(time.Now().Add(time.Hour * 25).Unix()),
			},
			valid:  false,
			errMsg: "remote access cannot be granted for more than",
		},
		"CreateBadResource_expiration_is_missing": {
			in: &remoteaccessv1.RemoteAccessConfiguration{
				DesiredState: remoteaccessv1.RemoteAccessState_REMOTE_ACCESS_STATE_ENABLED,
				Instance:     instance1,
			},
			valid:  false,
			errMsg: "expiration_timestamp cannot be 0",
		},
		"CreateBadResource_InvalidResourceId": {
			// This tests case verifies that create requests with an invalid resource ID are rejected.
			in: &remoteaccessv1.RemoteAccessConfiguration{
				ResourceId: "invalid-id",
				Instance:   instance2,
			},
			valid: false,
		},
		"CreateBadResource_WithNotExistingInstance": {
			in: &remoteaccessv1.RemoteAccessConfiguration{
				Instance:     &computev1.InstanceResource{ResourceId: "inst-aabbccdd"},
				DesiredState: remoteaccessv1.RemoteAccessState_REMOTE_ACCESS_STATE_ENABLED,
			},
			valid: false,
		},
		"CreateBadResource_WithAResourceIdSet": {
			// This tests case verifies that create requests with a resource ID
			// already set are rejected.
			in: &remoteaccessv1.RemoteAccessConfiguration{
				Instance: instance2,
			},
			valid: false,
		},
	}

	for tcname, tc := range testcases {
		t.Run(tcname, func(t *testing.T) {
			createReq := &inv_v1.Resource{
				Resource: &inv_v1.Resource_RemoteAccess{RemoteAccess: tc.in},
			}

			ctx, cancel := context.WithTimeout(context.Background(), time.Hour)
			defer cancel()

			createReqResp, err := inv_testing.TestClients[inv_testing.APIClient].Create(ctx, createReq)
			var remoteaccessResID string

			if err != nil {
				if tc.valid {
					require.NoError(t, err, "Create RemoteAccessConfiguration() failed")
				} else {
					require.ErrorContains(t, err, tc.errMsg)
				}
			} else {
				remoteaccessResID = inv_testing.GetResourceIDOrFail(t, createReqResp)
				tc.in.ResourceId = remoteaccessResID
				tc.in.CreatedAt = createReqResp.GetRemoteAccess().GetCreatedAt()
				tc.in.UpdatedAt = createReqResp.GetRemoteAccess().GetUpdatedAt()
				assertSameResource(t, createReq, createReqResp, nil)
				if !tc.valid {
					t.Errorf("Create RemoteAccessConfiguration() succeeded but should have failed")
				}
			}

			// only get/delete if valid test and hasn't failed otherwise may segfault
			if !t.Failed() && tc.valid {
				getResp, err := inv_testing.TestClients[inv_testing.APIClient].Get(ctx, remoteaccessResID)
				require.NoError(t, err, "Cannot get RemoteAccessConfiguration")

				// verify data
				if eq, diff := inv_testing.ProtoEqualOrDiff(
					tc.in.Instance,
					getResp.GetResource().GetRemoteAccess().GetInstance(),
				); !eq {
					t.Errorf("Get RemoteAccessConfiguration() data not equal: %v", diff)
				}

				// soft delete
				if _, err = inv_testing.TestClients[inv_testing.APIClient].Delete(ctx, remoteaccessResID); err != nil {
					t.Errorf("Soft Delete RemoteAccessConfiguration failed %s", err)
				}

				rsp, err := inv_testing.TestClients[inv_testing.APIClient].Get(ctx, remoteaccessResID)
				require.NoError(t, err, "Get RemoteAccessConfiguration() failed")
				require.NotNil(t, getResp.GetResource(), "unexpected response")
				require.NotNil(t, getResp.GetResource().GetRemoteAccess(), "unexpected response")
				require.Equal(t, remoteaccessv1.RemoteAccessState_REMOTE_ACCESS_STATE_DELETED,
					rsp.GetResource().GetRemoteAccess().GetDesiredState(), "unexpected desiredState")

				// report current state
				fm := &fieldmaskpb.FieldMask{Paths: []string{remoteaccessconfiguration.FieldCurrentState}}

				_, err = inv_testing.TestClients[inv_testing.RMClient].Update(
					context.TODO(),
					remoteaccessResID,
					fm,
					&inv_v1.Resource{
						Resource: &inv_v1.Resource_RemoteAccess{
							RemoteAccess: &remoteaccessv1.RemoteAccessConfiguration{
								CurrentState: remoteaccessv1.RemoteAccessState_REMOTE_ACCESS_STATE_DELETED,
								Instance:     getResp.GetResource().GetRemoteAccess().GetInstance(),
							},
						},
					})

				assert.NoError(t, err)

				_, err = inv_testing.TestClients[inv_testing.APIClient].Get(ctx, remoteaccessResID)
				require.Error(t, err, "resource shall not be returned since it has been deleted in previous step")
				require.Equal(t, codes.NotFound, status.Code(err))
			}
		})
	}
}

func Test_Get_RemoteAccessConfiguration_By_Host(t *testing.T) {
	racRes := inv_testing.CreateRemoteAccessConfiguration(t, func(r *remoteaccessv1.RemoteAccessConfiguration) {
		r.DesiredState = remoteaccessv1.RemoteAccessState_REMOTE_ACCESS_STATE_ENABLED
		r.ExpirationTimestamp = uint64(time.Now().Add(time.Minute * 15).Unix())
	})

	racResID := racRes.GetResourceId()

	rac, err := inv_testing.TestClients[inv_testing.APIClient].Get(context.TODO(), racResID)
	require.NoError(t, err, "Cannot get RemoteAccessConfiguration")

	instanceID := rac.GetResource().GetRemoteAccess().GetInstance().GetResourceId()
	instance, err := inv_testing.TestClients[inv_testing.APIClient].Get(context.TODO(), instanceID)
	require.NoError(t, err, "Cannot get Host")

	filter := &inv_v1.ResourceFilter{
		Filter: fmt.Sprintf(`%s.%s.%s = %q`,
			remoteaccessconfiguration.EdgeInstance,
			instanceresource.EdgeHost,
			hostresource.FieldResourceID,
			instance.GetResource().GetInstance().GetHost().GetResourceId()),
		Resource: &inv_v1.Resource{Resource: &inv_v1.Resource_RemoteAccess{}},
	}

	findQueryRes, err := inv_testing.TestClients[inv_testing.APIClient].Find(context.TODO(), filter)
	require.NoError(t, err, "find request has rejected")
	require.Len(t, findQueryRes.GetResources(), 1,
		"find request has returned unexpected number of resources")
	inv_testing.SortHasResourceIDAndTenantID(findQueryRes.Resources)
	expResp := &client.ResourceTenantIDCarrier{
		TenantId:   racRes.GetTenantId(),
		ResourceId: racRes.GetResourceId(),
	}
	assert.Equal(t, expResp, findQueryRes.GetResources()[0],
		"returned resource is different that expected")

	listQueryRsp, err := inv_testing.TestClients[inv_testing.APIClient].List(context.TODO(), filter)
	require.NoError(t, err, "list query has been rejected")
	require.Len(t, listQueryRsp.GetResources(), 1, "list request has returned unexpected number of resources")
	returned := listQueryRsp.GetResources()[0].GetResource().GetRemoteAccess().GetResourceId()
	require.Equal(t, racResID, returned)
}

func Test_Update_RemoteAccessConfiguration(t *testing.T) {
	racRes := inv_testing.CreateRemoteAccessConfiguration(t)
	racResID := racRes.GetResourceId()

	randomPort := uint32(inv_testing.GenerateRandInt(1024, 65535))
	randomUserName := strconv.Itoa(rand.Int())
	requestedConfigurationStatusMsg := "foobar"
	requestedConfigurationStatusIndicator := statusv1.StatusIndication_STATUS_INDICATION_IN_PROGRESS
	updated, err := inv_testing.TestClients[inv_testing.APIClient].Update(
		context.TODO(),
		racResID,
		&fieldmaskpb.FieldMask{},
		&inv_v1.Resource{Resource: &inv_v1.Resource_RemoteAccess{RemoteAccess: &remoteaccessv1.RemoteAccessConfiguration{
			DesiredState:                 remoteaccessv1.RemoteAccessState_REMOTE_ACCESS_STATE_ENABLED,
			User:                         randomUserName,
			LocalPort:                    randomPort,
			ConfigurationStatus:          requestedConfigurationStatusMsg,
			ConfigurationStatusIndicator: requestedConfigurationStatusIndicator,
			ConfigurationStatusTimestamp: uint64(time.Now().Unix()),
		}}})

	require.NoError(t, err, updated)

	getResp, err := inv_testing.TestClients[inv_testing.APIClient].Get(context.TODO(), racResID)
	require.NoError(t, err)
	require.NotNil(t, getResp)
	require.NotNil(t, getResp.GetResource())
	require.NotNil(t, getResp.GetResource().GetRemoteAccess())
	require.Equal(t, randomPort, getResp.GetResource().GetRemoteAccess().LocalPort, "unexpected port")
	require.Equal(t, randomUserName, getResp.GetResource().GetRemoteAccess().User, "unexpected user")
	require.Equal(t, requestedConfigurationStatusMsg, getResp.GetResource().GetRemoteAccess().ConfigurationStatus,
		"unexpected ConfigurationStatus")
	require.Equal(t, requestedConfigurationStatusIndicator, getResp.GetResource().GetRemoteAccess().ConfigurationStatusIndicator,
		"unexpected ConfigurationStatusIndicator")
	require.Greater(t, getResp.GetResource().GetRemoteAccess().GetConfigurationStatusTimestamp(), uint64(0))

	fm, err := fieldmaskpb.New(&remoteaccessv1.RemoteAccessConfiguration{}, "local_port", "user")
	require.NoError(t, err)

	_, err = inv_testing.TestClients[inv_testing.APIClient].Update(
		context.TODO(),
		racResID,
		fm,
		&inv_v1.Resource{Resource: &inv_v1.Resource_RemoteAccess{RemoteAccess: &remoteaccessv1.RemoteAccessConfiguration{}}})

	require.NoError(t, err)

	getResp2, err := inv_testing.TestClients[inv_testing.APIClient].Get(context.TODO(), racResID)
	require.NoError(t, err)
	require.NotNil(t, getResp2)
	require.NotNil(t, getResp2.GetResource())
	require.NotNil(t, getResp2.GetResource().GetRemoteAccess())
	assert.Zero(t, getResp2.GetResource().GetRemoteAccess().LocalPort, "unexpected port")
	assert.Empty(t, getResp2.GetResource().GetRemoteAccess().User, "unexpected user")
}

func Test_ExpirationTimestampIsImmutable(t *testing.T) {
	racRes := inv_testing.CreateRemoteAccessConfiguration(t)
	racResID := racRes.GetResourceId()

	fm, err := fieldmaskpb.New(&remoteaccessv1.RemoteAccessConfiguration{}, "expiration_timestamp")
	require.NoError(t, err)

	updated, err := inv_testing.TestClients[inv_testing.APIClient].Update(
		context.TODO(),
		racResID,
		fm,
		&inv_v1.Resource{Resource: &inv_v1.Resource_RemoteAccess{RemoteAccess: &remoteaccessv1.RemoteAccessConfiguration{
			ExpirationTimestamp: uint64(time.Now().Add(time.Minute * 15).Unix()),
		}}})

	assert.Error(t, err)
	require.ErrorContains(t, err, "is immutable")
	require.Equal(t, codes.InvalidArgument, status.Code(err), "unexpected error code")
	assert.Nil(t, updated)
}

func Test_FindByUser(t *testing.T) {
	testUser := strconv.Itoa(int(time.Now().Unix()))
	rac1 := inv_testing.CreateRemoteAccessConfiguration(t, func(r *remoteaccessv1.RemoteAccessConfiguration) {
		r.User = testUser
	})
	rac2 := inv_testing.CreateRemoteAccessConfiguration(t, func(r *remoteaccessv1.RemoteAccessConfiguration) {
		r.User = testUser
	})

	filter := &inv_v1.ResourceFilter{
		Filter:   fmt.Sprintf(`%s = %q`, remoteaccessconfiguration.FieldUser, testUser),
		Resource: &inv_v1.Resource{Resource: &inv_v1.Resource_RemoteAccess{}},
	}
	res, err := inv_testing.TestClients[inv_testing.APIClient].Find(context.TODO(), filter)
	assert.NoError(t, err)
	assert.Len(t, res.GetResources(), 2)
	expRes := []*client.ResourceTenantIDCarrier{
		{TenantId: client.FakeTenantID, ResourceId: rac1.GetResourceId()},
		{TenantId: client.FakeTenantID, ResourceId: rac2.GetResourceId()},
	}
	assert.ElementsMatch(t, res.GetResources(), expRes)
}

func Test_StrongRelations_On_Delete_RemoteAccessConfiguration(t *testing.T) {
	racRes := inv_testing.CreateRemoteAccessConfiguration(t)
	racResID := racRes.GetResourceId()

	rac, err := inv_testing.TestClients[inv_testing.APIClient].Get(context.TODO(), racResID)
	require.NoError(t, err, "Cannot get RemoteAccessConfiguration")

	instanceID := rac.GetResource().GetRemoteAccess().GetInstance().GetResourceId()

	err = inv_testing.HardDeleteInstanceAndReturnError(t, instanceID)
	assertStrongRelationError(t, err, "violates foreign key constraint")
}

func TestRemoteAccessConfigurationMTSanity(t *testing.T) {
	dao := inv_testing.NewInvResourceDAOOrFail(t)
	suite.Run(t, &struct{ mt }{
		mt: mt{
			createResource: func(tenantID string) (string, *inv_v1.Resource) {
				rac := dao.CreateRemoteAccessConfiguration(t, tenantID)
				res, err := util.WrapResource(rac)
				require.NoError(t, err)
				return rac.GetResourceId(), res
			},
		},
	})
}
