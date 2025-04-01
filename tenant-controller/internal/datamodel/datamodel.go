// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package datamodel

import (
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/logging"
	"github.com/open-edge-platform/infra-core/tenant-controller/internal/nexus"
	nexus_client "github.com/open-edge-platform/orch-utils/tenancy-datamodel/build/nexus-client"
)

var log = logging.GetLogger("data-model")

func NewDataModelController(
	nxc *nexus.Client, trace bool, tenantTerminationHandler tenantTerminationHandler,
	tenantInitializationHandler tenantInitializationHandler,
) *Controller {
	projectsToBeInitialized := make(chan *nexus_client.RuntimeprojectRuntimeProject)
	projectsToBeTerminated := make(chan *nexus_client.RuntimeprojectRuntimeProject)

	return &Controller{
		nxc:                     nxc,
		projectsToBeInitialized: projectsToBeInitialized,
		projectsToBeTerminated:  projectsToBeTerminated,
		projectReconciler: newProjectReconciliationController(
			nxc,
			trace,
			tenantTerminationHandler,
			tenantInitializationHandler,
			projectsToBeInitialized,
			projectsToBeTerminated,
		),
	}
}

type Controller struct {
	nxc                     *nexus.Client
	projectsToBeInitialized chan *nexus_client.RuntimeprojectRuntimeProject
	projectsToBeTerminated  chan *nexus_client.RuntimeprojectRuntimeProject
	projectReconciler       ProjectReconciliationController
}

func (d *Controller) Start(termChan chan bool) error {
	// Start Nexus event handler
	d.nxc.SubscribeAll()
	projects := d.nxc.TenancyMultiTenancy().Runtime().Orgs("*").Folders("*").Projects("*")

	// Setup Project watcher for INFRA-TC
	if err := d.nxc.SetupWatcherConfig(); err != nil {
		log.Err(err).Msgf("Error when setting up watchers")
		return err
	}

	if _, err := projects.RegisterAddCallback(d.onCreate()); err != nil {
		log.InfraSec().Err(err).Msg("failed registering create callback for rPROJECT")
		return err
	}

	if _, err := projects.RegisterUpdateCallback(d.onUpdate()); err != nil {
		log.InfraSec().Err(err).Msg("failed to register update callback for rPROJECT")
		return err
	}

	go d.projectReconciler.ControlLoop(termChan)
	return nil
}

func (d *Controller) onUpdate() func(
	_ *nexus_client.RuntimeprojectRuntimeProject, obj *nexus_client.RuntimeprojectRuntimeProject,
) {
	return func(_, obj *nexus_client.RuntimeprojectRuntimeProject) {
		if obj.Spec.Deleted {
			log.Debug().Msgf("rPROJECT[%s/%s] UpdateCallback - project deletion reported", obj.GetUID(), obj.DisplayName())
			d.projectsToBeTerminated <- obj
		} else {
			log.Debug().Msgf("rPROJECT[%s/%s] UpdateCallback - project update reported", obj.GetUID(), obj.DisplayName())
			d.projectsToBeInitialized <- obj
		}
	}
}

func (d *Controller) onCreate() func(obj *nexus_client.RuntimeprojectRuntimeProject) {
	return func(obj *nexus_client.RuntimeprojectRuntimeProject) {
		log.Debug().Msgf("project[%s/%s] initialization reported", obj.GetUID(), obj.DisplayName())
		if obj.Spec.Deleted {
			log.Debug().Msgf("rPROJECT[%s/%s] AddCallback - project deletion reported", obj.GetUID(), obj.DisplayName())
			d.projectsToBeTerminated <- obj
		} else {
			log.Debug().Msgf("rPROJECT[%s/%s] AddCallback - project creation reported", obj.GetUID(), obj.DisplayName())
			d.projectsToBeInitialized <- obj
		}
	}
}
