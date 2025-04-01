// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	inv_v1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/inventory/v1"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/client"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/util/collections"
	"github.com/open-edge-platform/infra-core/tenant-controller/internal/configuration"
	"github.com/open-edge-platform/infra-core/tenant-controller/internal/invclient"
	testutils "github.com/open-edge-platform/infra-core/tenant-controller/internal/testing"
)

func TestTenantHardDeletion(t *testing.T) {
	configuration.DefaultBackoff = backoff.WithMaxRetries(backoff.NewConstantBackOff(time.Millisecond), 10)

	tenantID := uuid.NewString()
	ic := new(invClientMock)

	ic.On("GetTenantResource", mock.Anything, tenantID).
		Return("", "", fmt.Errorf("error")).
		Times(1)

	ic.On("GetTenantResource", mock.Anything, tenantID).
		Return(tenantID, "any", nil)

	ic.On("HardDeleteTenantResource", mock.Anything, tenantID, mock.AnythingOfType("string")).
		Return(fmt.Errorf("hard deletion error")).Times(1)

	ic.On("HardDeleteTenantResource", mock.Anything, tenantID, mock.AnythingOfType("string")).
		Return(nil).Times(1)

	tt := NewTenantTerminator(ic, nil, tenantID, []*terminationStep{
		{resourceKind: inv_v1.ResourceKind_RESOURCE_KIND_TENANT, terminationFunc: tenantHardDeletion, watchEvents: true},
	})

	require.NoError(t, tt.Run(context.TODO()))
	ic.AssertExpectations(t)
}

func TestWorkloadsTermination_enforcedHardDeletion(t *testing.T) {
	tenantID := uuid.NewString()

	expectEnforcedHardDeletion := true

	// short time of waiting for soft deletion executed by OM
	workloadSoftDeletionTimeout := time.Millisecond
	configuration.WorkloadSoftDeletionTimeout = &workloadSoftDeletionTimeout

	ic := new(invClientMock)
	// FindAll keep returning existing resources
	ic.On("FindAll", mock.Anything, mock.Anything).
		Return(
			[]*client.ResourceTenantIDCarrier{{TenantId: tenantID, ResourceId: "any"}},
			fmt.Errorf("tc-inv communication problems"),
		)
	// DeleteAllResources(members) call accepted
	ic.On("DeleteAllResources",
		mock.Anything, tenantID,
		inv_v1.ResourceKind_RESOURCE_KIND_WORKLOAD_MEMBER, expectEnforcedHardDeletion).
		Return(nil).
		Times(1)

	// DeleteAllResources(workloads) call accepted
	ic.On("DeleteAllResources",
		mock.Anything, tenantID,
		inv_v1.ResourceKind_RESOURCE_KIND_WORKLOAD, expectEnforcedHardDeletion).
		Return(nil).
		Times(1)

	tt := NewTenantTerminator(ic, nil, tenantID, []*terminationStep{
		{resourceKind: inv_v1.ResourceKind_RESOURCE_KIND_WORKLOAD, terminationFunc: workloadsDeletion, watchEvents: true},
	})

	require.NoError(t, tt.Run(context.TODO()))
	ic.AssertExpectations(t)
}

func TestTenantTerminationCalledMultipleTimes(t *testing.T) {
	ic := testutils.CreateInvClient(t)
	tenantID := uuid.NewString()

	tc := NewTerminationController(ic)

	// emulate ongoing termination
	tc.terminators.PutIfAbsent(tenantID, nil)

	err := tc.TerminateTenant(context.Background(), tenantID)
	require.Error(t, err, "second call of tenant %s deletion shall be rejected", tenantID)
}

// INV client used by TC need to support events required by terminator.
func TestEventsConfigurationForInventoryClients(t *testing.T) {
	expectedResourceKinds := collections.MapSlice[*terminationStep, string](
		collections.Filter[*terminationStep](tenantTerminationSteps, func(ts *terminationStep) bool {
			return ts.watchEvents
		}), func(step *terminationStep) string {
			return step.resourceKind.String()
		},
	)

	t.Run("checking events configured on inv client", func(t *testing.T) {
		configuredResourceKinds := collections.MapSlice[inv_v1.ResourceKind, string](invclient.SupportedEventKinds,
			func(kind inv_v1.ResourceKind) string {
				return kind.String()
			})

		require.ElementsMatch(t, expectedResourceKinds, configuredResourceKinds)
	})

	t.Run("checking events configured on test inv client", func(t *testing.T) {
		configuredResourceKinds := collections.MapSlice[inv_v1.ResourceKind, string](testutils.SupportedEvents,
			func(kind inv_v1.ResourceKind) string {
				return kind.String()
			})

		require.ElementsMatch(t, expectedResourceKinds, configuredResourceKinds)
	})
}

func TestTenantTerminator_Run_terminationStepsReturnError(t *testing.T) {
	expectedError := fmt.Errorf("resource termination error")
	sut := NewTenantTerminator(nil, nil, uuid.NewString(), []*terminationStep{
		{
			resourceKind: 0,
			terminationFunc: func(_ context.Context, _ *TenantTerminator, _ inv_v1.ResourceKind) error {
				return expectedError
			},
			watchEvents: false,
		},
	})

	require.ErrorIs(t, sut.Run(context.TODO()), expectedError)
}
