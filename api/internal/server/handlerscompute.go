// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/open-edge-platform/infra-core/api/internal/types"
	api "github.com/open-edge-platform/infra-core/api/pkg/api/v0"
)

// (GET /hardware/compute).
func (r *restHandlers) GetCompute(
	ctx echo.Context,
	query api.GetComputeParams,
) error {
	log.Debug().Msg("GetCompute")

	err := ValidateQuery(ctx.QueryParams(), query)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, err.Error())
	}

	hostListResponse, err := r.CreateAndExecuteJob(ctx, types.List, types.Host, nil, api.GetComputeHostsParams(query), nil)
	if err != nil {
		return err
	}

	hostsList, ok := hostListResponse.Payload.Data.(api.HostsList)
	if !ok {
		log.Debug().Msg("error in parsing list host job response")
	}

	return ctx.JSON(http.StatusOK, hostsList)
}
