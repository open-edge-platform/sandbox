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

var pageSize = 100

// (GET /compute/hosts).
func (r *restHandlers) GetComputeHosts(
	ctx echo.Context,
	query api.GetComputeHostsParams,
) error {
	log.Debug().Msg("GetComputeHost")

	err := ValidateQuery(ctx.QueryParams(), query)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, err.Error())
	}

	res, err := r.CreateAndExecuteJob(ctx, types.List, types.Host, nil, query, nil)
	if err != nil {
		return err
	}
	return ctx.JSON(res.Status, res.Payload.Data)
}

// (POST /compute/hosts).
func (r *restHandlers) PostComputeHosts(ctx echo.Context) error {
	log.Debug().Msg("PostComputeHost")

	err := ValidateQuery(ctx.QueryParams(), nil)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, err.Error())
	}

	params := handlers.HostURLParams{
		Action: types.ActionUnspecified,
	}

	// cast the body to a type, as it is known which type we are receiving
	body := new(api.Host)
	res, err := r.CreateAndExecuteJob(ctx, types.Post, types.Host, body, nil, params)
	if err != nil {
		return err
	}
	return ctx.JSON(res.Status, res.Payload.Data)
}

// (GET /compute/hosts/{hostID}).
func (r *restHandlers) GetComputeHostsHostID(ctx echo.Context, hostID string) error {
	log.Debug().Msg("GetComputeHostID")

	err := ValidateQuery(ctx.QueryParams(), nil)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, err.Error())
	}

	params := handlers.HostURLParams{
		HostID: hostID,
	}
	res, err := r.CreateAndExecuteJob(ctx, types.Get, types.Host, nil, nil, params)
	if err != nil {
		return err
	}
	return ctx.JSON(res.Status, res.Payload.Data)
}

// (DELETE /compute/hosts/{hostID}).
func (r *restHandlers) DeleteComputeHostsHostID(ctx echo.Context, hostID string) error {
	log.Debug().Msg("DeleteComputeHostID")

	err := ValidateQuery(ctx.QueryParams(), nil)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, err.Error())
	}

	params := handlers.HostURLParams{
		HostID: hostID,
	}
	body := new(api.HostOperationWithNote)
	res, err := r.CreateAndExecuteJob(ctx, types.Delete, types.Host, body, nil, params)
	if err != nil {
		return err
	}
	return ctx.JSON(res.Status, res.Payload.Data)
}

// (PUT /compute/hosts/{hostID}).
func (r *restHandlers) PutComputeHostsHostID(ctx echo.Context, hostID string) error {
	log.Debug().Msg("PutComputeHostsHostID")

	err := ValidateQuery(ctx.QueryParams(), nil)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, err.Error())
	}

	params := handlers.HostURLParams{
		HostID: hostID,
	}

	// cast the body to a type, as it is known which type we are receiving
	body := new(api.Host)
	res, err := r.CreateAndExecuteJob(ctx, types.Put, types.Host, body, nil, params)
	if err != nil {
		return err
	}
	return ctx.JSON(res.Status, res.Payload.Data)
}

// (PATCH /compute/hosts/{hostID}).
func (r *restHandlers) PatchComputeHostsHostID(ctx echo.Context, hostID string) error {
	log.Debug().Msg("PatchComputeHostsHostID")

	err := ValidateQuery(ctx.QueryParams(), nil)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, err.Error())
	}

	params := handlers.HostURLParams{
		HostID: hostID,
	}

	// cast the body to a type, as it is known which type we are receiving
	body := new(api.Host)
	res, err := r.CreateAndExecuteJob(ctx, types.Patch, types.Host, body, nil, params)
	if err != nil {
		return err
	}
	return ctx.JSON(res.Status, res.Payload.Data)
}

// (PUT /compute/hosts/{hostID}/invalidate).
func (r *restHandlers) PutComputeHostsHostIDInvalidate(ctx echo.Context, hostID string) error {
	log.Debug().Msg("PutComputeHostsHostIDInvalidate")

	err := ValidateQuery(ctx.QueryParams(), nil)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, err.Error())
	}

	params := handlers.HostURLParams{
		HostID: hostID,
		Action: types.HostActionInvalidate,
	}

	// cast the body to a type, as it is known which type we are receiving
	body := new(api.HostOperationWithNote)
	res, err := r.CreateAndExecuteJob(ctx, types.Put, types.Host, body, nil, params)
	if err != nil {
		return err
	}
	return ctx.JSON(res.Status, res.Payload.Data)
}

//nolint:cyclop // high cyclomatic complexity due to complex status checking
func parseHostsSummary(hosts []api.Host) *api.HostsSummary {
	total := 0
	errorState := 0
	runningState := 0
	unallocatedState := 0

	// isFailedHostInstanceStatus verifies the status of a Host in case it has an Instance.
	isFailedHostInstanceStatus := func(host api.Host) bool {
		if host.Instance == nil {
			return false
		}
		return *host.Instance.InstanceStatusIndicator == api.STATUSINDICATIONERROR ||
			*host.Instance.ProvisioningStatusIndicator == api.STATUSINDICATIONERROR ||
			*host.Instance.UpdateStatusIndicator == api.STATUSINDICATIONERROR ||
			*host.Instance.TrustedAttestationStatusIndicator == api.STATUSINDICATIONERROR
	}

	isFailedHostStatus := func(host api.Host) bool {
		return *host.HostStatusIndicator == api.STATUSINDICATIONERROR ||
			*host.OnboardingStatusIndicator == api.STATUSINDICATIONERROR ||
			*host.RegistrationStatusIndicator == api.STATUSINDICATIONERROR ||
			isFailedHostInstanceStatus(host)
	}

	for _, hostAPI := range hosts {
		if hostAPI.Site == nil {
			unallocatedState++
		}
		if hostAPI.Site != nil && *hostAPI.Site.SiteID == "" {
			unallocatedState++
		}

		if isFailedHostStatus(hostAPI) {
			errorState++
		}

		// Since IDLE status can be used for multiple status (e.g., Powered off or Invalidated),
		// we use Instance's current state as a source of Running state.
		// To avoid counting hosts in both Running and Error states
		// current state must be RUNNING, but Host/Instance status cannot be a failure status.
		if hostAPI.Instance != nil && *hostAPI.Instance.CurrentState == api.INSTANCESTATERUNNING &&
			!isFailedHostStatus(hostAPI) {
			runningState++
		}
	}
	total = len(hosts)

	// Notice, error and running numbers come from Provider.State
	hostsSummary := &api.HostsSummary{
		Total:       &total,
		Error:       &errorState,
		Running:     &runningState,
		Unallocated: &unallocatedState,
	}
	return hostsSummary
}

// (GET /compute/hosts/summary).
func (r *restHandlers) GetComputeHostsSummary(
	ctx echo.Context,
	query api.GetComputeHostsSummaryParams,
) error {
	log.Debug().Msg("GetComputeHostsSummary")

	err := ValidateQuery(ctx.QueryParams(), query)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, err.Error())
	}

	offset := 0
	jobQuery := api.GetComputeHostsParams{
		SiteID:   query.SiteID,
		Filter:   query.Filter,
		PageSize: &pageSize,
		Offset:   &offset,
	}

	// Pre-allocate the first page
	hosts := make([]api.Host, 0, pageSize)
	hasNext := true
	for hasNext {
		hostListResponse, err := r.CreateAndExecuteJob(ctx, types.List, types.Host, nil, jobQuery, nil)
		if err != nil {
			return err
		}

		hostsList, ok := hostListResponse.Payload.Data.(api.HostsList)
		if !ok {
			log.Debug().Msg("error in handle list host job response for summary")
			var zeroCount int
			res := &api.HostsSummary{
				Error:       &zeroCount,
				Running:     &zeroCount,
				Total:       &zeroCount,
				Unallocated: &zeroCount,
			}
			return ctx.JSON(http.StatusOK, res)
		}

		if hostsList.Hosts != nil {
			hosts = append(hosts, *hostsList.Hosts...)
			hasNext = *hostsList.HasNext
			offset += len(*hostsList.Hosts)
		} else {
			log.Debug().Msg("no hosts available in list host job response for summary")
			hasNext = false
		}
	}
	res := parseHostsSummary(hosts)
	return ctx.JSON(http.StatusOK, res)
}

// (POST /compute/hosts/register).
func (r *restHandlers) PostComputeHostsRegister(ctx echo.Context) error {
	log.Debug().Msg("PostComputeHostsRegister")

	err := ValidateQuery(ctx.QueryParams(), nil)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, err.Error())
	}

	params := handlers.HostURLParams{
		Action: types.HostActionRegister,
	}

	body := new(api.HostRegisterInfo)
	res, err := r.CreateAndExecuteJob(ctx, types.Post, types.Host, body, nil, params)
	if err != nil {
		return err
	}
	return ctx.JSON(res.Status, res.Payload.Data)
}

// (PATCH /compute/hosts/{hostID}/register).
func (r *restHandlers) PatchComputeHostsHostIDRegister(ctx echo.Context, hostID string) error {
	log.Debug().Msg("PatchComputeHostsHostIDRegister")

	err := ValidateQuery(ctx.QueryParams(), nil)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, err.Error())
	}

	params := handlers.HostURLParams{
		HostID: hostID,
		Action: types.HostActionRegister,
	}

	body := new(api.HostRegisterInfo)
	res, err := r.CreateAndExecuteJob(ctx, types.Patch, types.Host, body, nil, params)
	if err != nil {
		return err
	}
	return ctx.JSON(res.Status, res.Payload.Data)
}

// (PATCH /compute/hosts/{hostID}/onboard).
func (r *restHandlers) PatchComputeHostsHostIDOnboard(ctx echo.Context, hostID string) error {
	log.Debug().Msg("PatchComputeHostsHostIDOnboard")

	err := ValidateQuery(ctx.QueryParams(), nil)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, err.Error())
	}

	params := handlers.HostURLParams{
		HostID: hostID,
		Action: types.HostActionOnboard,
	}

	res, err := r.CreateAndExecuteJob(ctx, types.Patch, types.Host, nil, nil, params)
	if err != nil {
		return err
	}
	return ctx.JSON(res.Status, res.Payload.Data)
}
