// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/open-edge-platform/infra-core/api/internal/types"
	handlers "github.com/open-edge-platform/infra-core/api/internal/worker/handlers/invhandlers"
	api "github.com/open-edge-platform/infra-core/api/pkg/api/v0"
)

// (GET /OSResources).
func (r *restHandlers) GetOSResources(ctx echo.Context, query api.GetOSResourcesParams) error {
	log.Debug().Msg("GetOSResources")

	err := ValidateQuery(ctx.QueryParams(), query)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, err.Error())
	}

	res, err := r.CreateAndExecuteJob(ctx, types.List, types.OSResource, nil, query, nil)
	if err != nil {
		return err
	}
	return ctx.JSON(res.Status, res.Payload.Data)
}

// (POST /OSResources).
func (r *restHandlers) PostOSResources(ctx echo.Context) error {
	log.Debug().Msg("PostOSResources")

	err := ValidateQuery(ctx.QueryParams(), nil)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, err.Error())
	}

	// cast the body to a type, as it is known which type we are receiving
	body := new(api.OperatingSystemResource)
	res, err := r.CreateAndExecuteJob(ctx, types.Post, types.OSResource, body, nil, nil)
	if err != nil {
		return err
	}
	return ctx.JSON(res.Status, res.Payload.Data)
}

// (DELETE /OSResources/{OSResourceID}).
func (r *restHandlers) DeleteOSResourcesOSResourceID(ctx echo.Context, operatingSystemResourceID string) error {
	log.Debug().Msg("DeleteOSResourcesOSResourceID")

	err := ValidateQuery(ctx.QueryParams(), nil)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, err.Error())
	}

	params := handlers.OSResourceURLParams{
		OSResourceID: operatingSystemResourceID,
	}
	res, err := r.CreateAndExecuteJob(ctx, types.Delete, types.OSResource, nil, nil, params)
	if err != nil {
		return err
	}
	return ctx.JSON(res.Status, res.Payload.Data)
}

// (GET /OSResources/{OSResourceID}).
func (r *restHandlers) GetOSResourcesOSResourceID(ctx echo.Context, operatingSystemResourceID string) error {
	log.Debug().Msg("GetOSResourcesOSResourceID")

	err := ValidateQuery(ctx.QueryParams(), nil)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, err.Error())
	}

	params := handlers.OSResourceURLParams{
		OSResourceID: operatingSystemResourceID,
	}
	res, err := r.CreateAndExecuteJob(ctx, types.Get, types.OSResource, nil, nil, params)
	if err != nil {
		return err
	}
	return ctx.JSON(res.Status, res.Payload.Data)
}

// (PATCH /OSResources/{OSResourceID}).
func (r *restHandlers) PatchOSResourcesOSResourceID(ctx echo.Context, operatingSystemResourceID string) error {
	log.Debug().Msg("PatchOSResourcesOSResourceID")

	err := ValidateQuery(ctx.QueryParams(), nil)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, err.Error())
	}

	params := handlers.OSResourceURLParams{
		OSResourceID: operatingSystemResourceID,
	}

	// cast the body to a type, as it is known which type we are receiving
	body := new(api.OperatingSystemResource)
	res, err := r.CreateAndExecuteJob(ctx, types.Patch, types.OSResource, body, nil, params)
	if err != nil {
		return err
	}
	return ctx.JSON(res.Status, res.Payload.Data)
}

// (PUT /OSResources/{OSResourceID}).
func (r *restHandlers) PutOSResourcesOSResourceID(ctx echo.Context, operatingSystemResourceID string) error {
	log.Debug().Msg("PutOSResourcesOSResourceID")

	err := ValidateQuery(ctx.QueryParams(), nil)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, err.Error())
	}

	params := handlers.OSResourceURLParams{
		OSResourceID: operatingSystemResourceID,
	}

	// cast the body to a type, as it is known which type we are receiving
	body := new(api.OperatingSystemResource)
	res, err := r.CreateAndExecuteJob(ctx, types.Put, types.OSResource, body, nil, params)
	if err != nil {
		return err
	}
	return ctx.JSON(res.Status, res.Payload.Data)
}
