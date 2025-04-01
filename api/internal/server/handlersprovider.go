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

// (GET /Providers).
func (r *restHandlers) GetProviders(ctx echo.Context, query api.GetProvidersParams) error {
	log.Debug().Msg("GetProviders")

	err := ValidateQuery(ctx.QueryParams(), query)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, err.Error())
	}

	res, err := r.CreateAndExecuteJob(ctx, types.List, types.Provider, nil, query, nil)
	if err != nil {
		return err
	}
	return ctx.JSON(res.Status, res.Payload.Data)
}

// (POST /Providers).
func (r *restHandlers) PostProviders(ctx echo.Context) error {
	log.Debug().Msg("PostProviders")

	err := ValidateQuery(ctx.QueryParams(), nil)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, err.Error())
	}

	// cast the body to a type, as it is known which type we are receiving
	body := new(api.Provider)
	res, err := r.CreateAndExecuteJob(ctx, types.Post, types.Provider, body, nil, nil)
	if err != nil {
		return err
	}
	return ctx.JSON(res.Status, res.Payload.Data)
}

// (DELETE /Providers/{providerId}).
func (r *restHandlers) DeleteProvidersProviderID(ctx echo.Context, providerID string) error {
	log.Debug().Msg("DeleteProvidersProviderID")

	err := ValidateQuery(ctx.QueryParams(), nil)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, err.Error())
	}

	params := handlers.ProviderURLParams{
		ProviderID: providerID,
	}
	res, err := r.CreateAndExecuteJob(ctx, types.Delete, types.Provider, nil, nil, params)
	if err != nil {
		return err
	}
	return ctx.JSON(res.Status, res.Payload.Data)
}

// (GET /Providers/{providerId}).
func (r *restHandlers) GetProvidersProviderID(ctx echo.Context, providerID string) error {
	log.Debug().Msg("GetProvidersProviderID")

	err := ValidateQuery(ctx.QueryParams(), nil)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, err.Error())
	}

	params := handlers.ProviderURLParams{
		ProviderID: providerID,
	}
	res, err := r.CreateAndExecuteJob(ctx, types.Get, types.Provider, nil, nil, params)
	if err != nil {
		return err
	}
	return ctx.JSON(res.Status, res.Payload.Data)
}
