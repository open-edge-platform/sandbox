// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package handlers

import (
	"net/http"

	"google.golang.org/grpc/codes"

	"github.com/open-edge-platform/infra-core/api/internal/types"
	"github.com/open-edge-platform/infra-core/api/internal/worker/clients"
	inv_handlers "github.com/open-edge-platform/infra-core/api/internal/worker/handlers/invhandlers"
	schedule_cache "github.com/open-edge-platform/infra-core/inventory/v2/pkg/client/cache/schedule"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/errors"
)

// NewInventoryHandler instantiates an inventory handler given the required clients.
// All inventory resource handlers are instantiated given the supported types.
func NewInventoryHandler(client *clients.InventoryClientHandler,
	invHCacheClientHandler *schedule_cache.HScheduleCacheClient,
) Handler {
	resources := map[types.Resource]inv_handlers.InventoryResource{}

	for resType := range inv_handlers.SupportedResourceTypes {
		invResHandler, err := inv_handlers.NewInventoryResourceHandler(resType, client, invHCacheClientHandler)
		if err != nil {
			log.Warn().Msgf("Unsupported inventory resource type %s", resType)
		}
		resources[resType] = invResHandler
	}

	invHandler := &inventoryHandler{
		resources: resources,
	}
	return invHandler
}

type inventoryHandler struct {
	resources map[types.Resource]inv_handlers.InventoryResource
}

// CanHandle returns if inventory resource handlers provides support for a resource type.
func (h *inventoryHandler) CanHandle(resType types.Resource) bool {
	_, ok := inv_handlers.SupportedResourceTypes[resType]
	return ok
}

func (h *inventoryHandler) getResourceHandler(jobType types.Resource) (inv_handlers.InventoryResource, error) {
	resourceHandler, ok := h.resources[jobType]
	if !ok {
		log.InfraError("handle unsupported resource type %s", jobType).Msgf("")
		return nil, errors.Errorfc(codes.Unimplemented, "handle unsupported resource type %s", jobType)
	}
	return resourceHandler, nil
}

// Handle implements the execution of the job by inventory resource handlers.
func (h *inventoryHandler) Handle(job *types.Job) (*types.Response, error) {
	log.Info().Msgf("handle new job op %s resource %s", job.Operation, job.Resource)

	resourceHandler, err := h.getResourceHandler(job.Resource)
	if err != nil {
		return nil, err
	}

	switch job.Operation {
	case types.Post:
		return h.create(job, resourceHandler)
	case types.Get:
		return h.get(job, resourceHandler)
	case types.Put:
		return h.update(job, resourceHandler)
	case types.Patch:
		return h.update(job, resourceHandler)
	case types.List:
		return h.list(job, resourceHandler)
	case types.Delete:
		return h.delete(job, resourceHandler)
	default:
		err := errors.Errorfc(codes.Unimplemented, "unsupported operation %s for resource %s",
			job.Operation,
			job.Resource)
		log.InfraErr(err).Msg("")
		return nil, err
	}
}

func (h *inventoryHandler) create(job *types.Job, handler inv_handlers.InventoryResource) (*types.Response, error) {
	payload, err := handler.Create(job)
	if err != nil {
		return nil, err
	}

	return &types.Response{
		Payload: *payload,
		Status:  http.StatusCreated,
		ID:      job.ID,
	}, nil
}

func (h *inventoryHandler) list(job *types.Job, handler inv_handlers.InventoryResource) (*types.Response, error) {
	payload, err := handler.List(job)
	if err != nil {
		return nil, err
	}

	return &types.Response{
		Payload: *payload,
		Status:  http.StatusOK,
		ID:      job.ID,
	}, nil
}

func (h *inventoryHandler) get(job *types.Job, handler inv_handlers.InventoryResource) (*types.Response, error) {
	payload, err := handler.Get(job)
	if err != nil {
		return nil, err
	}

	return &types.Response{
		Payload: *payload,
		Status:  http.StatusOK,
		ID:      job.ID,
	}, nil
}

func (h *inventoryHandler) update(job *types.Job, handler inv_handlers.InventoryResource) (*types.Response, error) {
	payload, err := handler.Update(job)
	if err != nil {
		return nil, err
	}

	return &types.Response{
		Payload: *payload,
		Status:  http.StatusOK,
		ID:      job.ID,
	}, nil
}

func (h *inventoryHandler) delete(job *types.Job, handler inv_handlers.InventoryResource) (*types.Response, error) {
	err := handler.Delete(job)
	if err != nil {
		return nil, err
	}

	return &types.Response{
		Payload: types.Payload{Data: nil},
		Status:  http.StatusNoContent,
		ID:      job.ID,
	}, nil
}
