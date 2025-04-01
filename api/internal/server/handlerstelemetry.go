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

// (GET /telemetry/groups/logs).
func (r *restHandlers) GetTelemetryGroupsLogs(ctx echo.Context, query api.GetTelemetryGroupsLogsParams) error {
	log.Debug().Msg("GetTelemetryGroupsLogs")

	err := ValidateQuery(ctx.QueryParams(), query)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, err.Error())
	}

	res, err := r.CreateAndExecuteJob(ctx, types.List, types.TelemetryLogsGroup, nil, query, nil)
	if err != nil {
		return err
	}
	return ctx.JSON(res.Status, res.Payload.Data)
}

// (POST /telemetry/groups/logs).
func (r *restHandlers) PostTelemetryGroupsLogs(ctx echo.Context) error {
	log.Debug().Msg("PostTelemetryGroupsLogs")

	err := ValidateQuery(ctx.QueryParams(), nil)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, err.Error())
	}

	// cast the body to a type, as it is known which type we are receiving
	body := new(api.TelemetryLogsGroup)
	res, err := r.CreateAndExecuteJob(ctx, types.Post, types.TelemetryLogsGroup, body, nil, nil)
	if err != nil {
		return err
	}
	return ctx.JSON(res.Status, res.Payload.Data)
}

// (DELETE /telemetry/groups/logs/{telemetryLogsGroupId}).
//
//nolint:revive,stylecheck // auto-generated from OpenAPI
func (r *restHandlers) DeleteTelemetryGroupsLogsTelemetryLogsGroupId(ctx echo.Context, telemetryLogsGroupId string) error {
	log.Debug().Msg("DeleteTelemetryGroupsLogsTelemetryLogsGroupId")

	err := ValidateQuery(ctx.QueryParams(), nil)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, err.Error())
	}

	params := handlers.TelemetryLogsGroupURLParams{
		TelemetryLogsGroupID: telemetryLogsGroupId,
	}
	res, err := r.CreateAndExecuteJob(ctx, types.Delete, types.TelemetryLogsGroup, nil, nil, params)
	if err != nil {
		return err
	}
	return ctx.JSON(res.Status, res.Payload.Data)
}

// (GET /telemetry/groups/logs/{telemetryLogsGroupId}).
//
//nolint:revive,stylecheck // auto-generated from OpenAPI
func (r *restHandlers) GetTelemetryGroupsLogsTelemetryLogsGroupId(ctx echo.Context, telemetryLogsGroupId string) error {
	log.Debug().Msg("GetTelemetryGroupsLogsTelemetryLogsGroupId")

	err := ValidateQuery(ctx.QueryParams(), nil)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, err.Error())
	}

	params := handlers.TelemetryLogsGroupURLParams{
		TelemetryLogsGroupID: telemetryLogsGroupId,
	}
	res, err := r.CreateAndExecuteJob(ctx, types.Get, types.TelemetryLogsGroup, nil, nil, params)
	if err != nil {
		return err
	}
	return ctx.JSON(res.Status, res.Payload.Data)
}

// (GET /telemetry/groups/metrics).
func (r *restHandlers) GetTelemetryGroupsMetrics(ctx echo.Context, query api.GetTelemetryGroupsMetricsParams) error {
	log.Debug().Msg("GetTelemetryGroupsMetrics")

	err := ValidateQuery(ctx.QueryParams(), query)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, err.Error())
	}

	res, err := r.CreateAndExecuteJob(ctx, types.List, types.TelemetryMetricsGroup, nil, query, nil)
	if err != nil {
		return err
	}
	return ctx.JSON(res.Status, res.Payload.Data)
}

// (POST /telemetry/groups/metrics).
func (r *restHandlers) PostTelemetryGroupsMetrics(ctx echo.Context) error {
	log.Debug().Msg("PostTelemetryGroupsMetrics")

	err := ValidateQuery(ctx.QueryParams(), nil)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, err.Error())
	}

	// cast the body to a type, as it is known which type we are receiving
	body := new(api.TelemetryMetricsGroup)
	res, err := r.CreateAndExecuteJob(ctx, types.Post, types.TelemetryMetricsGroup, body, nil, nil)
	if err != nil {
		return err
	}
	return ctx.JSON(res.Status, res.Payload.Data)
}

// (DELETE /telemetry/groups/metrics/{telemetryMetricsGroupId}).
//
//nolint:revive,stylecheck // auto-generated from OpenAPI
func (r *restHandlers) DeleteTelemetryGroupsMetricsTelemetryMetricsGroupId(
	ctx echo.Context, telemetryMetricsGroupId string,
) error {
	log.Debug().Msg("DeleteTelemetryGroupsMetricsTelemetryMetricsGroupId")

	err := ValidateQuery(ctx.QueryParams(), nil)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, err.Error())
	}

	params := handlers.TelemetryMetricsGroupURLParams{
		TelemetryMetricsGroupID: telemetryMetricsGroupId,
	}
	res, err := r.CreateAndExecuteJob(ctx, types.Delete, types.TelemetryMetricsGroup, nil, nil, params)
	if err != nil {
		return err
	}
	return ctx.JSON(res.Status, res.Payload.Data)
}

// (GET /telemetry/groups/metrics/{telemetryMetricsGroupId}).
//
//nolint:revive,stylecheck // auto-generated from OpenAPI
func (r *restHandlers) GetTelemetryGroupsMetricsTelemetryMetricsGroupId(ctx echo.Context, telemetryMetricsGroupId string) error {
	log.Debug().Msg("GetTelemetryGroupsMetricsTelemetryMetricsGroupId")

	err := ValidateQuery(ctx.QueryParams(), nil)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, err.Error())
	}

	params := handlers.TelemetryMetricsGroupURLParams{
		TelemetryMetricsGroupID: telemetryMetricsGroupId,
	}
	res, err := r.CreateAndExecuteJob(ctx, types.Get, types.TelemetryMetricsGroup, nil, nil, params)
	if err != nil {
		return err
	}
	return ctx.JSON(res.Status, res.Payload.Data)
}

// (GET /telemetry/profiles/logs).
func (r *restHandlers) GetTelemetryProfilesLogs(ctx echo.Context, query api.GetTelemetryProfilesLogsParams) error {
	log.Debug().Msg("GetTelemetryProfilesLogs")

	err := ValidateQuery(ctx.QueryParams(), query)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, err.Error())
	}

	res, err := r.CreateAndExecuteJob(ctx, types.List, types.TelemetryLogsProfile, nil, query, nil)
	if err != nil {
		return err
	}
	return ctx.JSON(res.Status, res.Payload.Data)
}

// (POST /telemetry/profiles/logs).
func (r *restHandlers) PostTelemetryProfilesLogs(ctx echo.Context) error {
	log.Debug().Msg("PostTelemetryProfilesLogs")

	err := ValidateQuery(ctx.QueryParams(), nil)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, err.Error())
	}

	// cast the body to a type, as it is known which type we are receiving
	body := new(api.TelemetryLogsProfile)
	res, err := r.CreateAndExecuteJob(ctx, types.Post, types.TelemetryLogsProfile, body, nil, nil)
	if err != nil {
		return err
	}
	return ctx.JSON(res.Status, res.Payload.Data)
}

// (DELETE /telemetry/profiles/logs/{telemetryLogsProfileId}).
//
//nolint:revive,stylecheck // auto-generated from OpenAPI
func (r *restHandlers) DeleteTelemetryProfilesLogsTelemetryLogsProfileId(ctx echo.Context, telemetryLogsProfileId string) error {
	log.Debug().Msg("DeleteTelemetryProfilesLogsTelemetryLogsProfileId")

	err := ValidateQuery(ctx.QueryParams(), nil)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, err.Error())
	}

	params := handlers.TelemetryLogsProfileURLParams{
		TelemetryLogsProfileID: telemetryLogsProfileId,
	}
	res, err := r.CreateAndExecuteJob(ctx, types.Delete, types.TelemetryLogsProfile, nil, nil, params)
	if err != nil {
		return err
	}
	return ctx.JSON(res.Status, res.Payload.Data)
}

// (GET /telemetry/profiles/logs/{telemetryLogsProfileId}).
//
//nolint:revive,stylecheck // auto-generated from OpenAPI
func (r *restHandlers) GetTelemetryProfilesLogsTelemetryLogsProfileId(
	ctx echo.Context,
	telemetryLogsProfileId string,
) error {
	log.Debug().Msg("GetTelemetryProfilesLogsTelemetryLogsProfileId")

	err := ValidateQuery(ctx.QueryParams(), nil)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, err.Error())
	}

	params := handlers.TelemetryLogsProfileURLParams{
		TelemetryLogsProfileID: telemetryLogsProfileId,
	}

	res, err := r.CreateAndExecuteJob(ctx, types.Get, types.TelemetryLogsProfile, nil, nil, params)
	if err != nil {
		return err
	}
	return ctx.JSON(res.Status, res.Payload.Data)
}

// (PATCH /telemetry/profiles/logs/{telemetryLogsProfileId}).
//
//nolint:revive,stylecheck // auto-generated from OpenAPI
func (r *restHandlers) PatchTelemetryProfilesLogsTelemetryLogsProfileId(
	ctx echo.Context,
	telemetryLogsProfileId string,
) error {
	log.Debug().Msg("PatchTelemetryProfilesLogsTelemetryLogsProfileId")

	err := ValidateQuery(ctx.QueryParams(), nil)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, err.Error())
	}

	params := handlers.TelemetryLogsProfileURLParams{
		TelemetryLogsProfileID: telemetryLogsProfileId,
	}

	body := new(api.TelemetryLogsProfile)
	res, err := r.CreateAndExecuteJob(ctx, types.Patch, types.TelemetryLogsProfile, body, nil, params)
	if err != nil {
		return err
	}
	return ctx.JSON(res.Status, res.Payload.Data)
}

// (PUT /telemetry/profiles/logs/{telemetryLogsProfileId}).
//
//nolint:revive,stylecheck // auto-generated from OpenAPI
func (r *restHandlers) PutTelemetryProfilesLogsTelemetryLogsProfileId(ctx echo.Context, telemetryLogsProfileId string) error {
	log.Debug().Msg("PutTelemetryProfilesLogsTelemetryLogsProfileId")

	err := ValidateQuery(ctx.QueryParams(), nil)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, err.Error())
	}

	params := handlers.TelemetryLogsProfileURLParams{
		TelemetryLogsProfileID: telemetryLogsProfileId,
	}

	body := new(api.TelemetryLogsProfile)
	res, err := r.CreateAndExecuteJob(ctx, types.Put, types.TelemetryLogsProfile, body, nil, params)
	if err != nil {
		return err
	}
	return ctx.JSON(res.Status, res.Payload.Data)
}

// (GET /telemetry/profiles/metrics).
func (r *restHandlers) GetTelemetryProfilesMetrics(ctx echo.Context, query api.GetTelemetryProfilesMetricsParams) error {
	log.Debug().Msg("GetTelemetryProfilesMetrics")

	err := ValidateQuery(ctx.QueryParams(), query)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, err.Error())
	}

	res, err := r.CreateAndExecuteJob(ctx, types.List, types.TelemetryMetricsProfile, nil, query, nil)
	if err != nil {
		return err
	}
	return ctx.JSON(res.Status, res.Payload.Data)
}

// (POST /telemetry/profiles/metrics).
func (r *restHandlers) PostTelemetryProfilesMetrics(ctx echo.Context) error {
	log.Debug().Msg("PostTelemetryProfilesMetrics")

	err := ValidateQuery(ctx.QueryParams(), nil)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, err.Error())
	}

	// cast the body to a type, as it is known which type we are receiving
	body := new(api.TelemetryMetricsProfile)
	res, err := r.CreateAndExecuteJob(ctx, types.Post, types.TelemetryMetricsProfile, body, nil, nil)
	if err != nil {
		return err
	}
	return ctx.JSON(res.Status, res.Payload.Data)
}

// (DELETE /telemetry/profiles/metrics/{telemetryMetricsProfileId}).
//
//nolint:revive,stylecheck // auto-generated from OpenAPI
func (r *restHandlers) DeleteTelemetryProfilesMetricsTelemetryMetricsProfileId(
	ctx echo.Context, telemetryMetricsProfileId string,
) error {
	log.Debug().Msg("DeleteTelemetryProfilesMetricsTelemetryMetricsProfileId")

	err := ValidateQuery(ctx.QueryParams(), nil)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, err.Error())
	}

	params := handlers.TelemetryMetricsProfileURLParams{
		TelemetryMetricsProfileID: telemetryMetricsProfileId,
	}
	res, err := r.CreateAndExecuteJob(ctx, types.Delete, types.TelemetryMetricsProfile, nil, nil, params)
	if err != nil {
		return err
	}
	return ctx.JSON(res.Status, res.Payload.Data)
}

// (GET /telemetry/profiles/metrics/{telemetryMetricsProfileId}).
//
//nolint:revive,stylecheck // auto-generated from OpenAPI
func (r *restHandlers) GetTelemetryProfilesMetricsTelemetryMetricsProfileId(
	ctx echo.Context, telemetryMetricsProfileId string,
) error {
	log.Debug().Msg("GetTelemetryProfilesMetricsTelemetryMetricsProfileId")

	err := ValidateQuery(ctx.QueryParams(), nil)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, err.Error())
	}

	params := handlers.TelemetryMetricsProfileURLParams{
		TelemetryMetricsProfileID: telemetryMetricsProfileId,
	}

	res, err := r.CreateAndExecuteJob(ctx, types.Get, types.TelemetryMetricsProfile, nil, nil, params)
	if err != nil {
		return err
	}
	return ctx.JSON(res.Status, res.Payload.Data)
}

// (PATCH /telemetry/profiles/metrics/{telemetryMetricsProfileId}).
//
//nolint:revive,stylecheck // auto-generated from OpenAPI
func (r *restHandlers) PatchTelemetryProfilesMetricsTelemetryMetricsProfileId(
	ctx echo.Context,
	telemetryMetricsProfileId string,
) error {
	log.Debug().Msg("PatchTelemetryProfilesMetricsTelemetryMetricsProfileId")

	err := ValidateQuery(ctx.QueryParams(), nil)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, err.Error())
	}

	params := handlers.TelemetryMetricsProfileURLParams{
		TelemetryMetricsProfileID: telemetryMetricsProfileId,
	}

	body := new(api.TelemetryMetricsProfile)
	res, err := r.CreateAndExecuteJob(ctx, types.Patch, types.TelemetryMetricsProfile, body, nil, params)
	if err != nil {
		return err
	}
	return ctx.JSON(res.Status, res.Payload.Data)
}

// (PUT /telemetry/profiles/metrics/{telemetryMetricsProfileId}).
//
//nolint:revive,stylecheck // auto-generated from OpenAPI
func (r *restHandlers) PutTelemetryProfilesMetricsTelemetryMetricsProfileId(
	ctx echo.Context,
	telemetryMetricsProfileId string,
) error {
	log.Debug().Msg("PutTelemetryProfilesMetricsTelemetryMetricsProfileId")

	err := ValidateQuery(ctx.QueryParams(), nil)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, err.Error())
	}

	params := handlers.TelemetryMetricsProfileURLParams{
		TelemetryMetricsProfileID: telemetryMetricsProfileId,
	}

	body := new(api.TelemetryMetricsProfile)
	res, err := r.CreateAndExecuteJob(ctx, types.Put, types.TelemetryMetricsProfile, body, nil, params)
	if err != nil {
		return err
	}
	return ctx.JSON(res.Status, res.Payload.Data)
}
