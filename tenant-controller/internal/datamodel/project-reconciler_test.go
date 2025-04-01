// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package datamodel

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/errors"
	"github.com/open-edge-platform/infra-core/tenant-controller/internal/controller"
	recv2 "github.com/open-edge-platform/orch-library/go/pkg/controller/v2"
	baseruntimeprojectinfrahostcomv1 "github.com/open-edge-platform/orch-utils/tenancy-datamodel/build/apis/runtimeproject.edge-orchestrator.intel.com/v1"
	nexus_client "github.com/open-edge-platform/orch-utils/tenancy-datamodel/build/nexus-client"
)

func Test_reconcileProjectTermination_failOnGetRuntimeProjectByUID(t *testing.T) {
	nxc := new(nexusClientMock)
	nxc.On("GetRuntimeProjectByUID", mock.Anything, mock.Anything).
		Return(nil, fmt.Errorf("cannot communicate with datamodel"))
	terminationHandlerMock := new(tenantTerminationHandlerMock)

	sut := reconcileProjectTermination(nxc, terminationHandlerMock, true)
	resp := sut(context.TODO(), recv2.Request[ProjectID]{ID: "any"})
	require.IsType(t, new(recv2.RetryWith[ProjectID]), resp)
}

func Test_reconcileProjectTermination_failOnTerminateTenant(t *testing.T) {
	rp := &nexus_client.RuntimeprojectRuntimeProject{
		RuntimeProject: &baseruntimeprojectinfrahostcomv1.RuntimeProject{
			ObjectMeta: metav1.ObjectMeta{
				UID: types.UID(uuid.NewString()),
			},
		},
	}

	nxc := new(nexusClientMock)
	nxc.On("GetRuntimeProjectByUID", mock.Anything, mock.Anything).
		Return(rp, nil)
	terminationHandlerMock := new(tenantTerminationHandlerMock)
	terminationHandlerMock.On("TerminateTenant", mock.Anything, mock.Anything).
		Return(fmt.Errorf("cannot termimnate tenant"))

	sut := reconcileProjectTermination(nxc, terminationHandlerMock, true)
	resp := sut(context.TODO(), recv2.Request[ProjectID]{ID: "any"})
	require.IsType(t, new(recv2.RetryWith[ProjectID]), resp)
}

func Test_reconcileProjectCreation_failOnGetRuntimeProjectByUID(t *testing.T) {
	nxc := new(nexusClientMock)
	nxc.On("GetRuntimeProjectByUID", mock.Anything, mock.Anything).
		Return(nil, fmt.Errorf("cannot communicate with datamodel"))

	initializationHandlerMock := new(tenantInitializationHandlerMock)

	sut := reconcileProjectCreation(nxc, initializationHandlerMock, false)
	resp := sut(context.TODO(), recv2.Request[ProjectID]{ID: "any"})
	require.IsType(t, new(recv2.Fail[ProjectID]), resp)
}

func Test_reconcileProjectCreation_failOnRegisterActiveWatcher(t *testing.T) {
	rp := &nexus_client.RuntimeprojectRuntimeProject{
		RuntimeProject: &baseruntimeprojectinfrahostcomv1.RuntimeProject{
			ObjectMeta: metav1.ObjectMeta{
				UID: types.UID(uuid.NewString()),
			},
		},
	}

	nxc := new(nexusClientMock)
	nxc.On("GetRuntimeProjectByUID", mock.Anything, mock.Anything).
		Return(rp, nil)
	nxc.On("RegisterActiveWatcher", mock.Anything).
		Return(nil, fmt.Errorf("cannot register active watcher"))

	initializationHandlerMock := new(tenantInitializationHandlerMock)

	sut := reconcileProjectCreation(nxc, initializationHandlerMock, false)
	resp := sut(context.TODO(), recv2.Request[ProjectID]{ID: "any"})
	require.IsType(t, new(recv2.RetryWith[ProjectID]), resp)
}

func Test_reconcileProjectCreation_failOnInitializeTenant(t *testing.T) {
	rp := &nexus_client.RuntimeprojectRuntimeProject{
		RuntimeProject: &baseruntimeprojectinfrahostcomv1.RuntimeProject{
			ObjectMeta: metav1.ObjectMeta{
				UID: types.UID(uuid.NewString()),
			},
		},
	}

	nxc := new(nexusClientMock)
	nxc.On("GetRuntimeProjectByUID", mock.Anything, mock.Anything).
		Return(rp, nil)
	nxc.On("RegisterActiveWatcher", mock.Anything).
		Return(nil, nil)

	initializationHandlerMock := new(tenantInitializationHandlerMock)
	initializationHandlerMock.On("InitializeTenant", mock.Anything, mock.Anything).
		Return(fmt.Errorf("cannot initialize tenant"))

	sut := reconcileProjectCreation(nxc, initializationHandlerMock, false)
	resp := sut(context.TODO(), recv2.Request[ProjectID]{ID: "any"})
	require.IsType(t, new(recv2.RetryWith[ProjectID]), resp)
}

func Test_reconcileProjectCreation_HappyPath(t *testing.T) {
	rp := &nexus_client.RuntimeprojectRuntimeProject{
		RuntimeProject: &baseruntimeprojectinfrahostcomv1.RuntimeProject{
			ObjectMeta: metav1.ObjectMeta{
				UID: types.UID(uuid.NewString()),
			},
		},
	}

	nxc := new(nexusClientMock)
	nxc.On("GetRuntimeProjectByUID", mock.Anything, mock.Anything).
		Return(rp, nil)
	nxc.On("RegisterActiveWatcher", mock.Anything).
		Return(nil, nil)

	initializationHandlerMock := new(tenantInitializationHandlerMock)
	initializationHandlerMock.On("InitializeTenant", mock.Anything, mock.Anything).
		Return(nil)

	sut := reconcileProjectCreation(nxc, initializationHandlerMock, false)
	resp := sut(context.TODO(), recv2.Request[ProjectID]{ID: "any"})
	require.IsType(t, new(recv2.Ack[ProjectID]), resp)
}

type tenantTerminationHandlerMock struct {
	mock.Mock
}

func (t *tenantTerminationHandlerMock) TerminateTenant(ctx context.Context, tenantID string) error {
	args := t.Called(ctx, tenantID)
	return args.Error(0)
}

type tenantInitializationHandlerMock struct {
	mock.Mock
}

func (t *tenantInitializationHandlerMock) InitializeTenant(ctx context.Context, config controller.ProjectConfig) error {
	args := t.Called(ctx, config)
	return args.Error(0)
}

type nexusClientMock struct {
	mock.Mock
}

func (n *nexusClientMock) GetRuntimeProjectByUID(
	ctx context.Context, tenantID string,
) (*nexus_client.RuntimeprojectRuntimeProject, error) {
	args := n.Called(ctx, tenantID)
	if args[0] == nil {
		return nil, args.Error(1)
	}
	runtimeProject, ok := args.Get(0).(*nexus_client.RuntimeprojectRuntimeProject)
	if !ok {
		return nil, errors.Errorf("unexpected type for RuntimeprojectRuntimeProject: %T", args.Get(0))
	}
	return runtimeProject, args.Error(1)
}

func (n *nexusClientMock) RegisterActiveWatcher(
	rp *nexus_client.RuntimeprojectRuntimeProject,
) (*nexus_client.ProjectactivewatcherProjectActiveWatcher, error) {
	args := n.Called(rp)
	if args[0] == nil {
		return nil, args.Error(1)
	}
	projectActiveWatcher, ok := args.Get(0).(*nexus_client.ProjectactivewatcherProjectActiveWatcher)
	if !ok {
		return nil, errors.Errorf("unexpected type for ProjectactivewatcherProjectActiveWatcher: %T", args.Get(0))
	}
	return projectActiveWatcher, args.Error(1)
}

func (n *nexusClientMock) ReportError(aw *nexus_client.ProjectactivewatcherProjectActiveWatcher, msg, tenantID string) error {
	args := n.Called(aw, msg, tenantID)
	return args.Error(0)
}
