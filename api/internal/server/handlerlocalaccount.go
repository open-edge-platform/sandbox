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

// (GET /localaccounts).
func (r *restHandlers) GetLocalAccounts(ctx echo.Context, query api.GetLocalAccountsParams) error {
	log.Debug().Msg("GetLocalAccounts")

	err := ValidateQuery(ctx.QueryParams(), query)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, err.Error())
	}

	res, err := r.CreateAndExecuteJob(ctx, types.List, types.LocalAccount, nil, query, nil)
	if err != nil {
		return err
	}
	return ctx.JSON(res.Status, res.Payload.Data)
}

// (POST /localaccount).
func (r *restHandlers) PostLocalAccounts(ctx echo.Context) error {
	log.Debug().Msg("PostLocalAccount")

	err := ValidateQuery(ctx.QueryParams(), nil)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, err.Error())
	}

	// cast the body to a type, as it is known which type we are receiving
	body := new(api.LocalAccount)
	res, err := r.CreateAndExecuteJob(ctx, types.Post, types.LocalAccount, body, nil, nil)
	if err != nil {
		return err
	}
	return ctx.JSON(res.Status, res.Payload.Data)
}

// GET /localaccount/{localAccountId} - Get a local account by ID.
func (r *restHandlers) GetLocalAccountsLocalAccountID(ctx echo.Context, localAccountID string) error {
	log.Debug().Msg("GetLocalAccountsLocalAccountID")

	err := ValidateQuery(ctx.QueryParams(), nil)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, err.Error())
	}

	params := handlers.LocalAccountURLParams{
		LocalAccountID: localAccountID,
	}
	res, err := r.CreateAndExecuteJob(ctx, types.Get, types.LocalAccount, nil, nil, params)
	if err != nil {
		return err
	}
	return ctx.JSON(res.Status, res.Payload.Data)
}

// (DELETE /localaccount/{localAccountId}).
func (r *restHandlers) DeleteLocalAccountsLocalAccountID(ctx echo.Context, localAccountID string) error {
	log.Debug().Msg("DeleteLocalAccountLocalAccountID")

	err := ValidateQuery(ctx.QueryParams(), nil)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, err.Error())
	}

	params := handlers.LocalAccountURLParams{
		LocalAccountID: localAccountID,
	}
	res, err := r.CreateAndExecuteJob(ctx, types.Delete, types.LocalAccount, nil, nil, params)
	if err != nil {
		return err
	}
	return ctx.JSON(res.Status, res.Payload.Data)
}
