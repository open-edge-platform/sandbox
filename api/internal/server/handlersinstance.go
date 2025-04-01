// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/open-edge-platform/infra-core/api/internal/types"
	handlers "github.com/open-edge-platform/infra-core/api/internal/worker/handlers/invhandlers"
	"github.com/open-edge-platform/infra-core/api/pkg/api/v0"
)

// (GET /instances).
func (r *restHandlers) GetInstances(ctx echo.Context, params api.GetInstancesParams) error {
	log.Debug().Msg("GetInstances")

	err := ValidateQuery(ctx.QueryParams(), params)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, err.Error())
	}

	res, err := r.CreateAndExecuteJob(ctx, types.List, types.Instance, nil, params, nil)
	if err != nil {
		return err
	}
	return ctx.JSON(res.Status, res.Payload.Data)
}

// (POST /instances).
func (r *restHandlers) PostInstances(ctx echo.Context) error {
	log.Debug().Msg("PostInstances")

	err := ValidateQuery(ctx.QueryParams(), nil)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, err.Error())
	}

	// cast the body to a type, as it is known which type we are receiving
	body := new(api.Instance)
	res, err := r.CreateAndExecuteJob(ctx, types.Post, types.Instance, body, nil, nil)
	if err != nil {
		return err
	}
	return ctx.JSON(res.Status, res.Payload.Data)
}

// (DELETE /instances/{instanceID}).
func (r *restHandlers) DeleteInstancesInstanceID(ctx echo.Context, instanceID string) error {
	log.Debug().Msg("DeleteInstancesInstanceID")

	err := ValidateQuery(ctx.QueryParams(), nil)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, err.Error())
	}

	params := handlers.InstanceURLParams{
		InstanceID: instanceID,
	}
	res, err := r.CreateAndExecuteJob(ctx, types.Delete, types.Instance, nil, nil, params)
	if err != nil {
		return err
	}
	return ctx.JSON(res.Status, res.Payload.Data)
}

// (GET /instances/{instanceID}).
func (r *restHandlers) GetInstancesInstanceID(ctx echo.Context, instanceID string) error {
	log.Debug().Msg("GetInstancesInstanceID")

	err := ValidateQuery(ctx.QueryParams(), nil)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, err.Error())
	}

	params := handlers.InstanceURLParams{
		InstanceID: instanceID,
	}
	res, err := r.CreateAndExecuteJob(ctx, types.Get, types.Instance, nil, nil, params)
	if err != nil {
		return err
	}
	return ctx.JSON(res.Status, res.Payload.Data)
}

// (PATCH /instances/{instanceID}).
func (r *restHandlers) PatchInstancesInstanceID(ctx echo.Context, instanceID string) error {
	log.Debug().Msg("PatchInstancesInstanceID")

	err := ValidateQuery(ctx.QueryParams(), nil)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, err.Error())
	}

	params := handlers.InstanceURLParams{
		InstanceID: instanceID,
	}

	// cast the body to a type, as it is known which type we are receiving
	body := new(api.Instance)
	res, err := r.CreateAndExecuteJob(ctx, types.Patch, types.Instance, body, nil, params)
	if err != nil {
		return err
	}
	return ctx.JSON(res.Status, res.Payload.Data)
}

// (PUT /instances/{instanceID}/invalidate).
func (r *restHandlers) PutInstancesInstanceIDInvalidate(ctx echo.Context, instanceID string) error {
	log.Debug().Msg("PutInstancesInstanceIDInvalidate")

	err := ValidateQuery(ctx.QueryParams(), nil)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, err.Error())
	}

	params := handlers.InstanceURLParams{
		InstanceID: instanceID,
		Action:     types.InstanceActionInvalidate,
	}

	// cast the body to a type, as it is known which type we are receiving
	body := new(api.Instance)
	res, err := r.CreateAndExecuteJob(ctx, types.Put, types.Instance, body, nil, params)
	if err != nil {
		return err
	}
	return ctx.JSON(res.Status, res.Payload.Data)
}
