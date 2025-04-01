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

// (GET /regions).
func (r *restHandlers) GetRegions(ctx echo.Context, query api.GetRegionsParams) error {
	log.Debug().Msg("GetRegion")

	err := ValidateQuery(ctx.QueryParams(), query)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, err.Error())
	}

	params := handlers.RegionURLParams{}
	res, err := r.CreateAndExecuteJob(ctx, types.List, types.Region, nil, query, params)
	if err != nil {
		return err
	}
	return ctx.JSON(res.Status, res.Payload.Data)
}

// (POST /regions).
func (r *restHandlers) PostRegions(ctx echo.Context) error {
	log.Debug().Msg("PostRegion")

	err := ValidateQuery(ctx.QueryParams(), nil)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, err.Error())
	}

	params := handlers.RegionURLParams{}

	// cast the body to a type, as it is known which type we are receiving
	body := new(api.Region)
	res, err := r.CreateAndExecuteJob(ctx, types.Post, types.Region, body, nil, params)
	if err != nil {
		return err
	}
	return ctx.JSON(res.Status, res.Payload.Data)
}

// (DELETE /regions/{regionID}).
func (r *restHandlers) DeleteRegionsRegionID(ctx echo.Context, regionID string) error {
	log.Debug().Msg("DeleteRegionsRegionID")

	err := ValidateQuery(ctx.QueryParams(), nil)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, err.Error())
	}

	params := handlers.RegionURLParams{
		RegionID: regionID,
	}
	res, err := r.CreateAndExecuteJob(ctx, types.Delete, types.Region, nil, nil, params)
	if err != nil {
		return err
	}
	return ctx.JSON(res.Status, res.Payload.Data)
}

// (GET /regions/{regionID}).
func (r *restHandlers) GetRegionsRegionID(ctx echo.Context, regionID string) error {
	log.Debug().Msg("GetRegionsRegionID")

	err := ValidateQuery(ctx.QueryParams(), nil)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, err.Error())
	}

	params := handlers.RegionURLParams{
		RegionID: regionID,
	}
	res, err := r.CreateAndExecuteJob(ctx, types.Get, types.Region, nil, nil, params)
	if err != nil {
		return err
	}
	return ctx.JSON(res.Status, res.Payload.Data)
}

// (PUT /regions/{regionID}).
func (r *restHandlers) PutRegionsRegionID(ctx echo.Context, regionID string) error {
	log.Debug().Msg("PutRegionsRegionID")

	err := ValidateQuery(ctx.QueryParams(), nil)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, err.Error())
	}

	params := handlers.RegionURLParams{
		RegionID: regionID,
	}

	body := new(api.Region)
	res, err := r.CreateAndExecuteJob(ctx, types.Put, types.Region, body, nil, params)
	if err != nil {
		return err
	}
	return ctx.JSON(res.Status, res.Payload.Data)
}

// (PATCH /regions/{regionID}).
func (r *restHandlers) PatchRegionsRegionID(ctx echo.Context, regionID string) error {
	log.Debug().Msg("PatchRegionsRegionID")

	err := ValidateQuery(ctx.QueryParams(), nil)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, err.Error())
	}

	params := handlers.RegionURLParams{
		RegionID: regionID,
	}

	body := new(api.Region)
	res, err := r.CreateAndExecuteJob(ctx, types.Patch, types.Region, body, nil, params)
	if err != nil {
		return err
	}
	return ctx.JSON(res.Status, res.Payload.Data)
}

// (GET /sites).
func (r *restHandlers) GetSites(ctx echo.Context, query api.GetSitesParams) error {
	log.Debug().Msg("GetSites")

	err := ValidateQuery(ctx.QueryParams(), query)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, err.Error())
	}

	params := handlers.SiteURLParams{}
	res, err := r.CreateAndExecuteJob(ctx, types.List, types.Site, nil, query, params)
	if err != nil {
		return err
	}
	return ctx.JSON(res.Status, res.Payload.Data)
}

// (POST /sites).
func (r *restHandlers) PostSites(ctx echo.Context) error {
	log.Debug().Msg("PostSites")

	err := ValidateQuery(ctx.QueryParams(), nil)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, err.Error())
	}

	params := handlers.SiteURLParams{}

	// cast the body to a type, as it is known which type we are receiving
	body := new(api.Site)
	res, err := r.CreateAndExecuteJob(ctx, types.Post, types.Site, body, nil, params)
	if err != nil {
		return err
	}
	return ctx.JSON(res.Status, res.Payload.Data)
}

// (DELETE /sites/{siteID}).
func (r *restHandlers) DeleteSitesSiteID(
	ctx echo.Context,
	siteID string,
) error {
	log.Debug().Msg("DeleteSitesSiteID")

	err := ValidateQuery(ctx.QueryParams(), nil)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, err.Error())
	}

	params := handlers.SiteURLParams{
		SiteID: siteID,
	}
	res, err := r.CreateAndExecuteJob(ctx, types.Delete, types.Site, nil, nil, params)
	if err != nil {
		return err
	}
	return ctx.JSON(res.Status, res.Payload.Data)
}

// (GET /sites/{siteID}).
func (r *restHandlers) GetSitesSiteID(
	ctx echo.Context,
	siteID string,
) error {
	log.Debug().Msg("GetSitesSiteID")

	err := ValidateQuery(ctx.QueryParams(), nil)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, err.Error())
	}

	params := handlers.SiteURLParams{
		SiteID: siteID,
	}
	res, err := r.CreateAndExecuteJob(ctx, types.Get, types.Site, nil, nil, params)
	if err != nil {
		return err
	}
	return ctx.JSON(res.Status, res.Payload.Data)
}

// (PUT /sites/{siteID}).
func (r *restHandlers) PutSitesSiteID(
	ctx echo.Context,
	siteID string,
) error {
	log.Debug().Msg("PutSitesSiteID")

	err := ValidateQuery(ctx.QueryParams(), nil)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, err.Error())
	}

	params := handlers.SiteURLParams{
		SiteID: siteID,
	}

	// cast the body to a type, as it is known which type we are receiving
	body := new(api.Site)
	res, err := r.CreateAndExecuteJob(ctx, types.Put, types.Site, body, nil, params)
	if err != nil {
		return err
	}
	return ctx.JSON(res.Status, res.Payload.Data)
}

// (PATCH /sites/{siteID}).
func (r *restHandlers) PatchSitesSiteID(
	ctx echo.Context,
	siteID string,
) error {
	log.Debug().Msg("PatchSitesSiteID")

	err := ValidateQuery(ctx.QueryParams(), nil)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, err.Error())
	}

	params := handlers.SiteURLParams{
		SiteID: siteID,
	}

	// cast the body to a type, as it is known which type we are receiving
	body := new(api.Site)
	res, err := r.CreateAndExecuteJob(ctx, types.Patch, types.Site, body, nil, params)
	if err != nil {
		return err
	}
	return ctx.JSON(res.Status, res.Payload.Data)
}

// (GET /locations).
func (r *restHandlers) GetLocations(ctx echo.Context, query api.GetLocationsParams) error {
	log.Debug().Msg("GetLocations")

	err := ValidateQuery(ctx.QueryParams(), query)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, err.Error())
	}

	// cast the body to a type, as it is known which type we are receiving
	res, err := r.CreateAndExecuteJob(ctx, types.List, types.Locations, nil, query, nil)
	if err != nil {
		return err
	}
	return ctx.JSON(res.Status, res.Payload.Data)
}
