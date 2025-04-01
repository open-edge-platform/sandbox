// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package handlers

import (
	"google.golang.org/grpc/codes"

	"github.com/open-edge-platform/infra-core/api/internal/types"
	"github.com/open-edge-platform/infra-core/api/internal/worker/clients"
	"github.com/open-edge-platform/infra-core/api/pkg/utils"
	schedule_cache "github.com/open-edge-platform/infra-core/inventory/v2/pkg/client/cache/schedule"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/errors"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/logging"
)

var log = logging.GetLogger("handlers")

// Handlers define the methods that a worker
// handler must implement.
type Handlers interface {
	// Do performs the execution of a job
	// returning the output response of it,
	// and an error when existent.
	Do(*types.Job) (*types.Response, error)
}

// Handler defines the interface that
// must be satisfied by a handler to become
// part of the set of Handlers available to
// Do a job.
type Handler interface {
	// CanHandle returns if a Handler has support
	// to implement a method that handles a resource type.
	CanHandle(types.Resource) bool
	// Handle performs the execution of the job
	// by the Handler.
	Handle(*types.Job) (*types.Response, error)
}

// NewHandlers instantiate all the handlers available to Do jobs for a worker.
func NewHandlers(
	invClientHandler *clients.InventoryClientHandler,
	invHCacheClientHandler *schedule_cache.HScheduleCacheClient,
) Handlers {
	invHandler := NewInventoryHandler(invClientHandler, invHCacheClientHandler)

	return &workerHandlers{
		invHandler: invHandler,
	}
}

type workerHandlers struct {
	invHandler Handler
}

// getHandler selects which Handler can handle a specific resource type.
// Notice the order of preference done by the switch statement.
func (h workerHandlers) getHandler(resType types.Resource) (Handler, error) {
	switch {
	case h.invHandler.CanHandle(resType):
		return h.invHandler, nil
	default:
		log.InfraError("unsupported resource type %s", resType).Msgf("")
		return nil, errors.Errorfc(codes.Unimplemented, "unsupported resource type %s", resType)
	}
}

// Do performs the execution of a job by a handler
// that implements the support for the resource type of the job.
func (h workerHandlers) Do(job *types.Job) (*types.Response, error) {
	handler, err := h.getHandler(job.Resource)
	if err != nil {
		return nil, err
	}

	job.Context, err = utils.AppendJWTtoContext(job.Context)
	if err != nil {
		return nil, err
	}

	resp, err := handler.Handle(job)
	return resp, err
}
