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

// (GET /ous).
func (r *restHandlers) GetOus(ctx echo.Context, query api.GetOusParams) error {
	log.Debug().Msg("GetOUs")

	err := ValidateQuery(ctx.QueryParams(), query)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, err.Error())
	}
	res, err := r.CreateAndExecuteJob(ctx, types.List, types.OU, nil, query, nil)
	if err != nil {
		return err
	}
	return ctx.JSON(res.Status, res.Payload.Data)
}

// (POST /ous).
func (r *restHandlers) PostOus(ctx echo.Context) error {
	log.Debug().Msg("PostOUs")

	err := ValidateQuery(ctx.QueryParams(), nil)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, err.Error())
	}

	// cast the body to a type, as it is known which type we are receiving
	body := new(api.OU)
	res, err := r.CreateAndExecuteJob(ctx, types.Post, types.OU, body, nil, nil)
	if err != nil {
		return err
	}
	return ctx.JSON(res.Status, res.Payload.Data)
}

// (DELETE /ous/{ouID}).
func (r *restHandlers) DeleteOusOuID(ctx echo.Context, ouID string) error {
	log.Debug().Msg("DeleteOUsOUID")

	err := ValidateQuery(ctx.QueryParams(), nil)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, err.Error())
	}

	params := handlers.OUURLParams{
		OUID: ouID,
	}
	res, err := r.CreateAndExecuteJob(ctx, types.Delete, types.OU, nil, nil, params)
	if err != nil {
		return err
	}
	return ctx.JSON(res.Status, res.Payload.Data)
}

// (GET /ous/{ouID}).
func (r *restHandlers) GetOusOuID(ctx echo.Context, ouID string) error {
	log.Debug().Msg("GetOUsOUID")

	err := ValidateQuery(ctx.QueryParams(), nil)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, err.Error())
	}

	params := handlers.OUURLParams{
		OUID: ouID,
	}
	res, err := r.CreateAndExecuteJob(ctx, types.Get, types.OU, nil, nil, params)
	if err != nil {
		return err
	}
	return ctx.JSON(res.Status, res.Payload.Data)
}

// (PUT /ous/{ouID}).
func (r *restHandlers) PutOusOuID(ctx echo.Context, ouID string) error {
	log.Debug().Msg("PutOUsOUID")

	err := ValidateQuery(ctx.QueryParams(), nil)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, err.Error())
	}

	params := handlers.OUURLParams{
		OUID: ouID,
	}

	// cast the body to a type, as it is known which type we are receiving
	body := new(api.OU)
	res, err := r.CreateAndExecuteJob(ctx, types.Put, types.OU, body, nil, params)
	if err != nil {
		return err
	}
	return ctx.JSON(res.Status, res.Payload.Data)
}

// (PATCH /ous/{ouID}).
func (r *restHandlers) PatchOusOuID(ctx echo.Context, ouID string) error {
	log.Debug().Msg("PatchOUsOUID")

	err := ValidateQuery(ctx.QueryParams(), nil)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, err.Error())
	}

	params := handlers.OUURLParams{
		OUID: ouID,
	}

	// cast the body to a type, as it is known which type we are receiving
	body := new(api.OU)
	res, err := r.CreateAndExecuteJob(ctx, types.Patch, types.OU, body, nil, params)
	if err != nil {
		return err
	}
	return ctx.JSON(res.Status, res.Payload.Data)
}
