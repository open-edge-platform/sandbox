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

// (GET /workloads).
func (r *restHandlers) GetWorkloads(ctx echo.Context, params api.GetWorkloadsParams) error {
	log.Debug().Msg("GetWorkloads")

	err := ValidateQuery(ctx.QueryParams(), params)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, err.Error())
	}

	res, err := r.CreateAndExecuteJob(ctx, types.List, types.Workload, nil, params, nil)
	if err != nil {
		return err
	}
	return ctx.JSON(res.Status, res.Payload.Data)
}

// (GET /workloads/{workloadID}).
func (r *restHandlers) GetWorkloadsWorkloadID(ctx echo.Context, workloadID string) error {
	log.Debug().Msg("GetWorkloadsWorkloadID")

	err := ValidateQuery(ctx.QueryParams(), nil)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, err.Error())
	}

	params := handlers.WorkloadURLParams{
		WorkloadID: workloadID,
	}
	res, err := r.CreateAndExecuteJob(ctx, types.Get, types.Workload, nil, nil, params)
	if err != nil {
		return err
	}
	return ctx.JSON(res.Status, res.Payload.Data)
}

// (POST /workloads).
func (r *restHandlers) PostWorkloads(ctx echo.Context) error {
	log.Debug().Msg("PostWorkloads")

	err := ValidateQuery(ctx.QueryParams(), nil)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, err.Error())
	}

	// cast the body to a type, as it is known which type we are receiving
	body := new(api.Workload)
	res, err := r.CreateAndExecuteJob(ctx, types.Post, types.Workload, body, nil, nil)
	if err != nil {
		return err
	}
	return ctx.JSON(res.Status, res.Payload.Data)
}

// (DELETE /workloads/{workloadID}).
func (r *restHandlers) DeleteWorkloadsWorkloadID(ctx echo.Context, workloadID string) error {
	log.Debug().Msg("DeleteWorkloadsWorkloadID")

	err := ValidateQuery(ctx.QueryParams(), nil)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, err.Error())
	}

	params := handlers.WorkloadURLParams{
		WorkloadID: workloadID,
	}
	res, err := r.CreateAndExecuteJob(ctx, types.Delete, types.Workload, nil, nil, params)
	if err != nil {
		return err
	}
	return ctx.JSON(res.Status, res.Payload.Data)
}

// (PUT /workloads/{workloadID}).
func (r *restHandlers) PutWorkloadsWorkloadID(ctx echo.Context, workloadID string) error {
	log.Debug().Msg("PutWorkloadsWorkloadID")

	err := ValidateQuery(ctx.QueryParams(), nil)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, err.Error())
	}

	params := handlers.WorkloadURLParams{
		WorkloadID: workloadID,
	}

	// cast the body to a type, as it is known which type we are receiving
	body := new(api.Workload)
	res, err := r.CreateAndExecuteJob(ctx, types.Put, types.Workload, body, nil, params)
	if err != nil {
		return err
	}
	return ctx.JSON(res.Status, res.Payload.Data)
}

// (PATCH /workloads/{workloadID}).
func (r *restHandlers) PatchWorkloadsWorkloadID(ctx echo.Context, workloadID string) error {
	log.Debug().Msg("PatchWorkloadsWorkloadID")

	err := ValidateQuery(ctx.QueryParams(), nil)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, err.Error())
	}

	params := handlers.WorkloadURLParams{
		WorkloadID: workloadID,
	}

	// cast the body to a type, as it is known which type we are receiving
	body := new(api.Workload)
	res, err := r.CreateAndExecuteJob(ctx, types.Patch, types.Workload, body, nil, params)
	if err != nil {
		return err
	}
	return ctx.JSON(res.Status, res.Payload.Data)
}

// (GET /workloadMembers).
func (r *restHandlers) GetWorkloadMembers(ctx echo.Context, params api.GetWorkloadMembersParams) error {
	log.Debug().Msg("GetWorkloadMembers")

	err := ValidateQuery(ctx.QueryParams(), params)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, err.Error())
	}

	res, err := r.CreateAndExecuteJob(ctx, types.List, types.WorkloadMember, nil, params, nil)
	if err != nil {
		return err
	}
	return ctx.JSON(res.Status, res.Payload.Data)
}

// (POST /workloadMembers).
func (r *restHandlers) PostWorkloadMembers(ctx echo.Context) error {
	log.Debug().Msg("PostWorkloadMembers")

	err := ValidateQuery(ctx.QueryParams(), nil)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, err.Error())
	}

	// cast the body to a type, as it is known which type we are receiving
	body := new(api.WorkloadMember)
	res, err := r.CreateAndExecuteJob(ctx, types.Post, types.WorkloadMember, body, nil, nil)
	if err != nil {
		return err
	}
	return ctx.JSON(res.Status, res.Payload.Data)
}

// (GET /workloadMembers/{workloadMemberID}).
func (r *restHandlers) GetWorkloadMembersWorkloadMemberID(ctx echo.Context, workloadMemberID string) error {
	log.Debug().Msg("GetWorkloadMembersWorkloadMemberID")

	err := ValidateQuery(ctx.QueryParams(), nil)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, err.Error())
	}

	params := handlers.WorkloadMemberURLParams{
		WorkloadMemberID: workloadMemberID,
	}
	res, err := r.CreateAndExecuteJob(ctx, types.Get, types.WorkloadMember, nil, nil, params)
	if err != nil {
		return err
	}
	return ctx.JSON(res.Status, res.Payload.Data)
}

// (DELETE /workloadMembers/{workloadMemberID}).
func (r *restHandlers) DeleteWorkloadMembersWorkloadMemberID(ctx echo.Context, workloadMemberID string) error {
	log.Debug().Msg("DeleteWorkloadMembersWorkloadMemberID")

	err := ValidateQuery(ctx.QueryParams(), nil)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, err.Error())
	}

	params := handlers.WorkloadMemberURLParams{
		WorkloadMemberID: workloadMemberID,
	}
	res, err := r.CreateAndExecuteJob(ctx, types.Delete, types.WorkloadMember, nil, nil, params)
	if err != nil {
		return err
	}
	return ctx.JSON(res.Status, res.Payload.Data)
}
