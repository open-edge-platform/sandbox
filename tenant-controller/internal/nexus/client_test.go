// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package nexus_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/errors"
	"github.com/open-edge-platform/infra-core/tenant-controller/internal/configuration"
	"github.com/open-edge-platform/infra-core/tenant-controller/internal/nexus"
	baseprojectactivewatcherinfrahostcomv1 "github.com/open-edge-platform/orch-utils/tenancy-datamodel/build/apis/projectactivewatcher.edge-orchestrator.intel.com/v1"
	baseruntimeinfrahostcomv1 "github.com/open-edge-platform/orch-utils/tenancy-datamodel/build/apis/runtime.edge-orchestrator.intel.com/v1"
	baseruntimefolderinfrahostcomv1 "github.com/open-edge-platform/orch-utils/tenancy-datamodel/build/apis/runtimefolder.edge-orchestrator.intel.com/v1"
	baseruntimeorginfrahostcomv1 "github.com/open-edge-platform/orch-utils/tenancy-datamodel/build/apis/runtimeorg.edge-orchestrator.intel.com/v1"
	baseruntimeprojectinfrahostcomv1 "github.com/open-edge-platform/orch-utils/tenancy-datamodel/build/apis/runtimeproject.edge-orchestrator.intel.com/v1"
	tenancyv1 "github.com/open-edge-platform/orch-utils/tenancy-datamodel/build/apis/tenancy.edge-orchestrator.intel.com/v1"
	nexus_client "github.com/open-edge-platform/orch-utils/tenancy-datamodel/build/nexus-client"
)

func TestClient_NewClient(t *testing.T) {
	nxc := nexus.NewClient(nexus_client.NewFakeClient())
	require.NotNil(t, nxc)
}

func TestClient_GetRuntimeProjectByUID_HappyPath(t *testing.T) {
	nxc := nexus.NewClient(nexus_client.NewFakeClient())
	require.NotNil(t, nxc)

	projectID := uuid.NewString()

	ctx := context.Background()

	// create mt
	tenancyClient, err := nxc.AddTenancyMultiTenancy(
		ctx, &tenancyv1.MultiTenancy{ObjectMeta: metav1.ObjectMeta{Name: "default", ResourceVersion: "1"}},
	)
	require.NoError(t, err)

	runtimeClient, err := tenancyClient.AddRuntime(
		ctx,
		&baseruntimeinfrahostcomv1.Runtime{ObjectMeta: metav1.ObjectMeta{Name: "default", ResourceVersion: "1"}},
	)
	require.NoError(t, err)

	orgClient, err := runtimeClient.AddOrgs(
		ctx,
		&baseruntimeorginfrahostcomv1.RuntimeOrg{ObjectMeta: metav1.ObjectMeta{Name: "default", ResourceVersion: "1"}})
	require.NoError(t, err)

	folderClient, err := orgClient.AddFolders(
		ctx,
		&baseruntimefolderinfrahostcomv1.RuntimeFolder{ObjectMeta: metav1.ObjectMeta{Name: "default", ResourceVersion: "1"}},
	)
	require.NoError(t, err)

	projectClient, err := folderClient.AddProjects(ctx, &baseruntimeprojectinfrahostcomv1.RuntimeProject{
		ObjectMeta: metav1.ObjectMeta{Name: "default", ResourceVersion: "1", UID: types.UID(projectID)},
	})
	require.NoError(t, err)

	rp, err := nxc.GetRuntimeProjectByUID(ctx, projectID)
	require.NoError(t, err)
	require.Equal(t, projectClient, rp)
}

func TestClient_GetRuntimeProjectByUID_RequestedProjectDoesNotExist(t *testing.T) {
	nxc := nexus.NewClient(nexus_client.NewFakeClient())
	require.NotNil(t, nxc)

	projectID := uuid.NewString()

	ctx := context.Background()

	// create mt
	tenancyClient, err := nxc.AddTenancyMultiTenancy(
		ctx, &tenancyv1.MultiTenancy{ObjectMeta: metav1.ObjectMeta{Name: "default", ResourceVersion: "1"}},
	)
	require.NoError(t, err)

	runtimeClient, err := tenancyClient.AddRuntime(
		ctx,
		&baseruntimeinfrahostcomv1.Runtime{ObjectMeta: metav1.ObjectMeta{Name: "default", ResourceVersion: "1"}},
	)
	require.NoError(t, err)

	orgClient, err := runtimeClient.AddOrgs(
		ctx,
		&baseruntimeorginfrahostcomv1.RuntimeOrg{ObjectMeta: metav1.ObjectMeta{Name: "default", ResourceVersion: "1"}})
	require.NoError(t, err)

	_, err = orgClient.AddFolders(
		ctx,
		&baseruntimefolderinfrahostcomv1.RuntimeFolder{ObjectMeta: metav1.ObjectMeta{Name: "default", ResourceVersion: "1"}},
	)
	require.NoError(t, err)

	rp, err := nxc.GetRuntimeProjectByUID(ctx, projectID)
	require.Error(t, err)
	require.True(t, errors.IsNotFound(err))
	require.Nil(t, rp)
}

func TestClient_GetRuntimeProjectByUID_RuntimeIsMissing(t *testing.T) {
	nxc := nexus.NewClient(nexus_client.NewFakeClient())
	require.NotNil(t, nxc)

	projectID := uuid.NewString()

	ctx := context.Background()
	rp, err := nxc.GetRuntimeProjectByUID(ctx, projectID)
	require.Nil(t, rp)
	require.Error(t, err)
}

func TestClient_TryToSetActiveWatcherStatusIdle_RuntimeIsMissing(t *testing.T) {
	nxc := nexus.NewClient(nexus_client.NewFakeClient())
	err := nxc.TryToSetActiveWatcherStatusIdle("anyProject")
	require.Error(t, err)
	require.True(t, errors.IsNotFound(err))
}

func TestClient_TryToSetActiveWatcherStatusIdle_ActiveWatcherIsMissing(t *testing.T) {
	nxc := nexus.NewClient(nexus_client.NewFakeClient())
	ctx := context.Background()
	projectID := uuid.NewString()
	tenancyClient, err := nxc.AddTenancyMultiTenancy(
		ctx, &tenancyv1.MultiTenancy{ObjectMeta: metav1.ObjectMeta{Name: "default", ResourceVersion: "1"}},
	)
	require.NoError(t, err)

	runtimeClient, err := tenancyClient.AddRuntime(
		ctx,
		&baseruntimeinfrahostcomv1.Runtime{ObjectMeta: metav1.ObjectMeta{Name: "default", ResourceVersion: "1"}},
	)
	require.NoError(t, err)

	orgClient, err := runtimeClient.AddOrgs(
		ctx,
		&baseruntimeorginfrahostcomv1.RuntimeOrg{ObjectMeta: metav1.ObjectMeta{Name: "default", ResourceVersion: "1"}})
	require.NoError(t, err)

	folderClient, err := orgClient.AddFolders(
		ctx,
		&baseruntimefolderinfrahostcomv1.RuntimeFolder{ObjectMeta: metav1.ObjectMeta{Name: "default", ResourceVersion: "1"}},
	)
	require.NoError(t, err)

	projectClient, err := folderClient.AddProjects(ctx, &baseruntimeprojectinfrahostcomv1.RuntimeProject{
		ObjectMeta: metav1.ObjectMeta{Name: "default", ResourceVersion: "1", UID: types.UID(projectID)},
	})
	require.NoError(t, err)
	require.NotNil(t, projectClient)

	err = nxc.TryToSetActiveWatcherStatusIdle(projectID)
	require.Error(t, err)
	require.True(t, errors.IsNotFound(err))
}

func TestClient_TryToSetActiveWatcherStatusIdle_HappyPath(t *testing.T) {
	nxc := nexus.NewClient(nexus_client.NewFakeClient())
	ctx := context.Background()
	projectID := uuid.NewString()
	tenancyClient, err := nxc.AddTenancyMultiTenancy(
		ctx, &tenancyv1.MultiTenancy{ObjectMeta: metav1.ObjectMeta{Name: "default", ResourceVersion: "1"}},
	)
	require.NoError(t, err)

	runtimeClient, err := tenancyClient.AddRuntime(
		ctx,
		&baseruntimeinfrahostcomv1.Runtime{ObjectMeta: metav1.ObjectMeta{Name: "default", ResourceVersion: "1"}},
	)
	require.NoError(t, err)

	orgClient, err := runtimeClient.AddOrgs(
		ctx,
		&baseruntimeorginfrahostcomv1.RuntimeOrg{ObjectMeta: metav1.ObjectMeta{Name: "default", ResourceVersion: "1"}})
	require.NoError(t, err)

	folderClient, err := orgClient.AddFolders(
		ctx,
		&baseruntimefolderinfrahostcomv1.RuntimeFolder{ObjectMeta: metav1.ObjectMeta{Name: "default", ResourceVersion: "1"}},
	)
	require.NoError(t, err)

	projectClient, err := folderClient.AddProjects(ctx, &baseruntimeprojectinfrahostcomv1.RuntimeProject{
		ObjectMeta: metav1.ObjectMeta{Name: "default", ResourceVersion: "1", UID: types.UID(projectID)},
	})
	require.NoError(t, err)
	require.NotNil(t, projectClient)

	aw, err := nxc.RegisterActiveWatcher(projectClient)
	require.NoError(t, err)
	require.NotNil(t, aw)
	require.Equal(t, configuration.AppName, aw.DisplayName())
	require.Equalf(t,
		baseprojectactivewatcherinfrahostcomv1.StatusIndicationInProgress,
		aw.Spec.StatusIndicator, "registered AW shall be in running state",
	)

	// set AW status running -> idle
	err = nxc.TryToSetActiveWatcherStatusIdle(projectID)
	require.NoError(t, err)

	aw, err = projectClient.GetActiveWatchers(ctx, configuration.AppName)
	require.NoError(t, err)
	require.Equal(t, baseprojectactivewatcherinfrahostcomv1.StatusIndicationIdle, aw.Spec.StatusIndicator,
		"AW shall be in idle state",
	)

	// set idle  again
	err = nxc.TryToSetActiveWatcherStatusIdle(projectID)
	require.NoError(t, err)

	aw, err = projectClient.GetActiveWatchers(ctx, configuration.AppName)
	require.NoError(t, err)
	require.Equal(t, baseprojectactivewatcherinfrahostcomv1.StatusIndicationIdle, aw.Spec.StatusIndicator)
}

func TestClient_RegisterActiveWatcher_HappyPath(t *testing.T) {
	nxc := nexus.NewClient(nexus_client.NewFakeClient())
	ctx := context.Background()
	projectID := uuid.NewString()
	tenancyClient, err := nxc.AddTenancyMultiTenancy(
		ctx, &tenancyv1.MultiTenancy{ObjectMeta: metav1.ObjectMeta{Name: "default", ResourceVersion: "1"}},
	)
	require.NoError(t, err)

	runtimeClient, err := tenancyClient.AddRuntime(
		ctx,
		&baseruntimeinfrahostcomv1.Runtime{ObjectMeta: metav1.ObjectMeta{Name: "default", ResourceVersion: "1"}},
	)
	require.NoError(t, err)

	orgClient, err := runtimeClient.AddOrgs(
		ctx,
		&baseruntimeorginfrahostcomv1.RuntimeOrg{ObjectMeta: metav1.ObjectMeta{Name: "default", ResourceVersion: "1"}})
	require.NoError(t, err)

	folderClient, err := orgClient.AddFolders(
		ctx,
		&baseruntimefolderinfrahostcomv1.RuntimeFolder{ObjectMeta: metav1.ObjectMeta{Name: "default", ResourceVersion: "1"}},
	)
	require.NoError(t, err)

	projectClient, err := folderClient.AddProjects(ctx, &baseruntimeprojectinfrahostcomv1.RuntimeProject{
		ObjectMeta: metav1.ObjectMeta{Name: "default", ResourceVersion: "1", UID: types.UID(projectID)},
	})
	require.NoError(t, err)
	require.NotNil(t, projectClient)

	// register active watcher
	aw1, err := nxc.RegisterActiveWatcher(projectClient)
	require.NoError(t, err)
	require.NotNil(t, aw1)
	require.Equal(t, configuration.AppName, aw1.DisplayName())

	// assert existing active watchers
	watchers, err := projectClient.GetAllActiveWatchers(ctx)
	require.NoError(t, err)
	require.Len(t, watchers, 1)

	// try to create active watcher again
	aw2, err := nxc.RegisterActiveWatcher(projectClient)
	require.NoError(t, err)
	require.NotNil(t, aw1)
	require.Equal(t, aw1, aw2)

	// assert existing active watchers
	watchers, err = projectClient.GetAllActiveWatchers(ctx)
	require.NoError(t, err)
	require.Len(t, watchers, 1)

	// report error
	require.NoError(t, nxc.ReportError(aw2, "no way", "anyID"))
	aw2, err = projectClient.GetActiveWatchers(ctx, configuration.AppName)
	// assert reported error
	require.NoError(t, err)
	require.Equal(t, baseprojectactivewatcherinfrahostcomv1.StatusIndicationError, aw2.Spec.StatusIndicator)
	require.Equal(t, "no way", aw2.Spec.Message)

	// delete aw
	require.NoError(t, aw1.Delete(ctx))
	// assert deletion
	watchers, err = projectClient.GetAllActiveWatchers(ctx)
	require.NoError(t, err)
	require.Len(t, watchers, 0)
	// use another reference to AW and try to delete it
	err = aw2.Delete(ctx)
	require.Error(t, err)
	require.True(t, nexus_client.IsNotFound(err))
}

func TestClient_SetupWatcherConfig_HappyPath(t *testing.T) {
	nxc := nexus.NewClient(nexus_client.NewFakeClient())

	require.NoError(t, nxc.SetupWatcherConfig())

	watcher, err := nxc.TenancyMultiTenancy().Config().GetProjectWatchers(context.Background(), configuration.AppName)
	require.NoError(t, err)
	require.NotNil(t, watcher)
	require.Equal(t, configuration.AppName, watcher.DisplayName())
}
