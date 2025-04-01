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

// (GET /schedules).
func (r *restHandlers) GetSchedules(ctx echo.Context, query api.GetSchedulesParams) error {
	log.Debug().Msg("GetSchedules")

	err := ValidateQuery(ctx.QueryParams(), query)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, err.Error())
	}

	repeatedSchedResponse, err := r.CreateAndExecuteJob(
		ctx,
		types.List,
		types.RepeatedSched,
		nil,
		api.GetSchedulesRepeatedParams(query),
		nil,
	)
	if err != nil {
		return err
	}

	singleSchedResponse, err := r.CreateAndExecuteJob(
		ctx,
		types.List,
		types.SingleSched,
		nil,
		api.GetSchedulesSingleParams(query),
		nil,
	)
	if err != nil {
		return err
	}

	singleScheds, ok := singleSchedResponse.Payload.Data.(api.SingleSchedulesList)
	if !ok {
		log.Debug().Msg("error in handle list single schedule job response")
	}

	repeatedScheds, ok := repeatedSchedResponse.Payload.Data.(api.RepeatedSchedulesList)
	if !ok {
		log.Debug().Msg("error in handle list repeated schedule job response")
	}

	schedules := prepareSchedulesResponse(singleScheds, repeatedScheds)
	return ctx.JSON(http.StatusOK, schedules)
}

func prepareSchedulesResponse(
	singleScheds api.SingleSchedulesList,
	repeatedScheds api.RepeatedSchedulesList,
) api.SchedulesListJoin {
	schedules := api.SchedulesListJoin{}
	if singleScheds.SingleSchedules != nil {
		schedules.SingleSchedules = singleScheds.SingleSchedules
	}
	if repeatedScheds.RepeatedSchedules != nil {
		schedules.RepeatedSchedules = repeatedScheds.RepeatedSchedules
	}
	var hasNext bool
	if singleScheds.HasNext != nil {
		hasNext = *singleScheds.HasNext
	}
	if repeatedScheds.HasNext != nil {
		hasNext = hasNext || *repeatedScheds.HasNext
	}
	schedules.HasNext = &hasNext
	return schedules
}

// (GET /schedules/repeated).
func (r *restHandlers) GetSchedulesRepeated(
	ctx echo.Context,
	query api.GetSchedulesRepeatedParams,
) error {
	log.Debug().Msg("GetSchedulesRepeated")

	err := ValidateQuery(ctx.QueryParams(), query)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, err.Error())
	}
	res, err := r.CreateAndExecuteJob(ctx, types.List, types.RepeatedSched, nil, query, nil)
	if err != nil {
		return err
	}
	return ctx.JSON(res.Status, res.Payload.Data)
}

// (POST /schedules/repeated).
func (r *restHandlers) PostSchedulesRepeated(ctx echo.Context) error {
	log.Debug().Msg("PostSchedulesRepeated")

	err := ValidateQuery(ctx.QueryParams(), nil)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, err.Error())
	}

	// cast the body to a type, as it is known which type we are receiving
	body := new(api.RepeatedSchedule)
	res, err := r.CreateAndExecuteJob(ctx, types.Post, types.RepeatedSched, body, nil, nil)
	if err != nil {
		return err
	}
	return ctx.JSON(res.Status, res.Payload.Data)
}

// (DELETE /schedules/repeated/{repeatedScheduleID}).
func (r *restHandlers) DeleteSchedulesRepeatedRepeatedScheduleID(
	ctx echo.Context,
	repeatedScheduleID string,
) error {
	log.Debug().Msg("DeleteSchedulesRepeatedRepeatedScheduleID")

	err := ValidateQuery(ctx.QueryParams(), nil)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, err.Error())
	}

	params := handlers.RepeatedSchedURLParams{
		RepeatedSchedID: repeatedScheduleID,
	}
	res, err := r.CreateAndExecuteJob(ctx, types.Delete, types.RepeatedSched, nil, nil, params)
	if err != nil {
		return err
	}
	return ctx.JSON(res.Status, res.Payload.Data)
}

// (GET /schedules/repeated/{repeatedScheduleID}).
func (r *restHandlers) GetSchedulesRepeatedRepeatedScheduleID(
	ctx echo.Context,
	repeatedScheduleID string,
) error {
	log.Debug().Msg("GetSchedulesRepeatedRepeatedScheduleID")

	err := ValidateQuery(ctx.QueryParams(), nil)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, err.Error())
	}

	params := handlers.RepeatedSchedURLParams{
		RepeatedSchedID: repeatedScheduleID,
	}
	res, err := r.CreateAndExecuteJob(ctx, types.Get, types.RepeatedSched, nil, nil, params)
	if err != nil {
		return err
	}
	return ctx.JSON(res.Status, res.Payload.Data)
}

// (PATCH /schedules/repeated/{repeatedScheduleID}).
func (r *restHandlers) PatchSchedulesRepeatedRepeatedScheduleID(
	ctx echo.Context,
	repeatedScheduleID string,
) error {
	log.Debug().Msg("PatchSchedulesRepeatedRepeatedScheduleID")

	err := ValidateQuery(ctx.QueryParams(), nil)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, err.Error())
	}

	params := handlers.RepeatedSchedURLParams{
		RepeatedSchedID: repeatedScheduleID,
	}

	// cast the body to a type, as it is known which type we are receiving
	body := new(api.RepeatedSchedule)
	res, err := r.CreateAndExecuteJob(ctx, types.Patch, types.RepeatedSched, body, nil, params)
	if err != nil {
		return err
	}
	return ctx.JSON(res.Status, res.Payload.Data)
}

// (PUT /schedules/repeated/{repeatedScheduleID}).
func (r *restHandlers) PutSchedulesRepeatedRepeatedScheduleID(
	ctx echo.Context,
	repeatedScheduleID string,
) error {
	log.Debug().Msg("PutSchedulesRepeatedRepeatedScheduleID")

	err := ValidateQuery(ctx.QueryParams(), nil)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, err.Error())
	}

	params := handlers.RepeatedSchedURLParams{
		RepeatedSchedID: repeatedScheduleID,
	}

	// cast the body to a type, as it is known which type we are receiving
	body := new(api.RepeatedSchedule)
	res, err := r.CreateAndExecuteJob(ctx, types.Put, types.RepeatedSched, body, nil, params)
	if err != nil {
		return err
	}
	return ctx.JSON(res.Status, res.Payload.Data)
}

// (GET /schedules/single).
func (r *restHandlers) GetSchedulesSingle(
	ctx echo.Context,
	query api.GetSchedulesSingleParams,
) error {
	log.Debug().Msg("GetSchedulesSingle")

	err := ValidateQuery(ctx.QueryParams(), query)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, err.Error())
	}
	res, err := r.CreateAndExecuteJob(ctx, types.List, types.SingleSched, nil, query, nil)
	if err != nil {
		return err
	}
	return ctx.JSON(res.Status, res.Payload.Data)
}

// (POST /schedules/single).
func (r *restHandlers) PostSchedulesSingle(ctx echo.Context) error {
	log.Debug().Msg("PostSchedulesSingle")

	err := ValidateQuery(ctx.QueryParams(), nil)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, err.Error())
	}

	// cast the body to a type, as it is known which type we are receiving
	body := new(api.SingleSchedule)
	res, err := r.CreateAndExecuteJob(ctx, types.Post, types.SingleSched, body, nil, nil)
	if err != nil {
		return err
	}
	return ctx.JSON(res.Status, res.Payload.Data)
}

// (DELETE /schedules/single/{singleScheduleID}).
func (r *restHandlers) DeleteSchedulesSingleSingleScheduleID(
	ctx echo.Context,
	singleScheduleID string,
) error {
	log.Debug().Msg("DeleteSchedulesSingleSingleScheduleID")

	err := ValidateQuery(ctx.QueryParams(), nil)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, err.Error())
	}

	params := handlers.SingleSchedURLParams{
		SingleSchedID: singleScheduleID,
	}
	res, err := r.CreateAndExecuteJob(ctx, types.Delete, types.SingleSched, nil, nil, params)
	if err != nil {
		return err
	}
	return ctx.JSON(res.Status, res.Payload.Data)
}

// (GET /schedules/single/{singleScheduleID}).
func (r *restHandlers) GetSchedulesSingleSingleScheduleID(
	ctx echo.Context,
	singleScheduleID string,
) error {
	log.Debug().Msg("GetSchedulesSingleSingleScheduleID")

	err := ValidateQuery(ctx.QueryParams(), nil)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, err.Error())
	}

	params := handlers.SingleSchedURLParams{
		SingleSchedID: singleScheduleID,
	}
	res, err := r.CreateAndExecuteJob(ctx, types.Get, types.SingleSched, nil, nil, params)
	if err != nil {
		return err
	}
	return ctx.JSON(res.Status, res.Payload.Data)
}

// (PATCH /schedules/single/{singleScheduleID}).
func (r *restHandlers) PatchSchedulesSingleSingleScheduleID(
	ctx echo.Context,
	singleScheduleID string,
) error {
	log.Debug().Msg("PatchSchedulesSingleSingleScheduleID")

	err := ValidateQuery(ctx.QueryParams(), nil)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, err.Error())
	}

	params := handlers.SingleSchedURLParams{
		SingleSchedID: singleScheduleID,
	}

	// cast the body to a type, as it is known which type we are receiving
	body := new(api.SingleSchedule)
	res, err := r.CreateAndExecuteJob(ctx, types.Patch, types.SingleSched, body, nil, params)
	if err != nil {
		return err
	}
	return ctx.JSON(res.Status, res.Payload.Data)
}

// (PUT /schedules/single/{singleScheduleID}).
func (r *restHandlers) PutSchedulesSingleSingleScheduleID(
	ctx echo.Context,
	singleScheduleID string,
) error {
	log.Debug().Msg("PutSchedulesSingleSingleScheduleID")

	err := ValidateQuery(ctx.QueryParams(), nil)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, err.Error())
	}

	params := handlers.SingleSchedURLParams{
		SingleSchedID: singleScheduleID,
	}

	// cast the body to a type, as it is known which type we are receiving
	body := new(api.SingleSchedule)
	res, err := r.CreateAndExecuteJob(ctx, types.Put, types.SingleSched, body, nil, params)
	if err != nil {
		return err
	}
	return ctx.JSON(res.Status, res.Payload.Data)
}
