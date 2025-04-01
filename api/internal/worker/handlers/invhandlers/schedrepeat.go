// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package invhandlers

import (
	"golang.org/x/exp/maps"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	"github.com/open-edge-platform/infra-core/api/internal/types"
	"github.com/open-edge-platform/infra-core/api/internal/worker/clients"
	"github.com/open-edge-platform/infra-core/api/pkg/api/v0"
	computev1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/compute/v1"
	inventory "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/inventory/v1"
	locationv1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/location/v1"
	schedv1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/schedule/v1"
	schedule_cache "github.com/open-edge-platform/infra-core/inventory/v2/pkg/client/cache/schedule"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/errors"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/tenant"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/util"
	invcollections "github.com/open-edge-platform/infra-core/inventory/v2/pkg/util/collections"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/util/function"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/validator"
)

// OpenAPIRepeatedSchedToProto maps OpenAPI fields name to Proto fields name.
// The key is derived from the json property respectively of the
// structs SchedTemplate defined in edge-infrastructure-manager-openapi-types.gen.go.
var OpenAPIRepeatedSchedToProto = map[string]string{
	"name":            schedv1.RepeatedScheduleResourceFieldName,
	"targetSiteId":    schedv1.RepeatedScheduleResourceEdgeTargetSite,
	"targetHostId":    schedv1.RepeatedScheduleResourceEdgeTargetHost,
	"targetRegionId":  schedv1.RepeatedScheduleResourceEdgeTargetRegion,
	"durationSeconds": schedv1.RepeatedScheduleResourceFieldDurationSeconds,
	"cronMinutes":     schedv1.RepeatedScheduleResourceFieldCronMinutes,
	"cronHours":       schedv1.RepeatedScheduleResourceFieldCronHours,
	"cronDayMonth":    schedv1.RepeatedScheduleResourceFieldCronDayMonth,
	"cronMonth":       schedv1.RepeatedScheduleResourceFieldCronMonth,
	"cronDayWeek":     schedv1.RepeatedScheduleResourceFieldCronDayWeek,
	"scheduleStatus":  schedv1.RepeatedScheduleResourceFieldScheduleStatus,
}

// OpenAPIRepeatedSchedToProtoExcluded defines exclusion rules as there are some fields
// defined in the OpenAPI spec that are not currently mapped to the proto
// fields.
var OpenAPIRepeatedSchedToProtoExcluded = map[string]struct{}{
	"repeatedScheduleID": {}, // repeatedScheduleID must not be set from the API
	"resourceId":         {}, // resourceId must not be set from the API
	"targetHost":         {}, // targetHost must not be set from the API
	"targetSite":         {}, // targetSite must not be set from the API
	"targetRegion":       {}, // targetRegion must not be set from the API
	"timestamps":         {}, // read-only field
}

func NewRepeatedSchedHandler(
	invClient *clients.InventoryClientHandler,
	hScheduleCache *schedule_cache.HScheduleCacheClient,
) InventoryResource {
	return &repeatedSchedHandler{
		invClient:      invClient,
		hScheduleCache: hScheduleCache,
	}
}

type repeatedSchedHandler struct {
	invClient      *clients.InventoryClientHandler
	hScheduleCache *schedule_cache.HScheduleCacheClient
}

func (h *repeatedSchedHandler) Create(job *types.Job) (*types.Payload, error) {
	tenantID, exists := tenant.GetTenantIDFromContext(job.Context)
	if !exists {
		// This should never happen! Interceptor should either fail or set it!
		err := errors.Errorfc(codes.Unauthenticated, "Tenant ID is not present in context")
		log.InfraSec().InfraErr(err).Msg("Create repeated schedule is not authenticated")
		return nil, err
	}

	body, err := castRepeatedSched(&job.Payload)
	if err != nil {
		return nil, err
	}

	repeatedSched, err := openapiToGrpcRepeatedSched(body)
	if err != nil {
		return nil, err
	}

	req := &inventory.Resource{
		Resource: &inventory.Resource_Repeatedschedule{
			Repeatedschedule: repeatedSched,
		},
	}

	invResp, err := h.invClient.InvClient.Create(job.Context, req)
	if err != nil {
		return nil, err
	}

	cratedRSched := invResp.GetRepeatedschedule()
	h.hScheduleCache.InvalidateCache(
		tenantID, cratedRSched.GetResourceId(), inventory.SubscribeEventsResponse_EVENT_KIND_CREATED)

	obj := grpcToOpenAPIRepeatedSched(cratedRSched, nil)

	return &types.Payload{Data: obj}, err
}

func (h *repeatedSchedHandler) Get(job *types.Job) (*types.Payload, error) {
	tenantID, exists := tenant.GetTenantIDFromContext(job.Context)
	if !exists {
		// This should never happen! Interceptor should either fail or set it!
		err := errors.Errorfc(codes.Unauthenticated, "Tenant ID is not present in context")
		log.InfraSec().InfraErr(err).Msg("Get repeated schedule is not authenticated")
		return nil, err
	}

	req, err := repeatedSchedResourceID(&job.Payload)
	if err != nil {
		return nil, err
	}

	repeatedSched, err := h.hScheduleCache.GetRepeatedSchedule(tenantID, req)
	if err != nil {
		return nil, err
	}

	obj := grpcToOpenAPIRepeatedSched(repeatedSched, nil)
	return &types.Payload{Data: obj}, nil
}

func (h *repeatedSchedHandler) Update(job *types.Job) (*types.Payload, error) {
	tenantID, exists := tenant.GetTenantIDFromContext(job.Context)
	if !exists {
		// This should never happen! Interceptor should either fail or set it!
		err := errors.Errorfc(codes.Unauthenticated, "Tenant ID is not present in context")
		log.InfraSec().InfraErr(err).Msg("Update repeated schedule is not authenticated")
		return nil, err
	}

	resID, err := repeatedSchedResourceID(&job.Payload)
	if err != nil {
		return nil, err
	}

	fm, err := repeatedSchedFieldMask(&job.Payload, job.Operation)
	if err != nil {
		return nil, err
	}

	res, err := repeatedSchedResource(&job.Payload)
	if err != nil {
		return nil, err
	}

	invResp, err := h.invClient.InvClient.Update(job.Context, resID, fm, res)
	if err != nil {
		return nil, err
	}

	h.hScheduleCache.InvalidateCache(tenantID, resID, inventory.SubscribeEventsResponse_EVENT_KIND_UPDATED)

	updatedRepeatedSched := invResp.GetRepeatedschedule()
	obj := grpcToOpenAPIRepeatedSched(updatedRepeatedSched, nil)
	obj.RepeatedScheduleID = &resID // to be removed
	obj.ResourceId = &resID
	return &types.Payload{Data: obj}, nil
}

func (h *repeatedSchedHandler) Delete(job *types.Job) error {
	tenantID, exists := tenant.GetTenantIDFromContext(job.Context)
	if !exists {
		// This should never happen! Interceptor should either fail or set it!
		err := errors.Errorfc(codes.Unauthenticated, "Tenant ID is not present in context")
		log.InfraSec().InfraErr(err).Msg("Delete repeated schedule is not authenticated")
		return err
	}

	req, err := repeatedSchedResourceID(&job.Payload)
	if err != nil {
		return err
	}

	_, err = h.invClient.InvClient.Delete(job.Context, req)
	if err != nil {
		return err
	}

	h.hScheduleCache.InvalidateCache(tenantID, req, inventory.SubscribeEventsResponse_EVENT_KIND_DELETED)

	return nil
}

func (h *repeatedSchedHandler) List(job *types.Job) (*types.Payload, error) {
	tenantID, exists := tenant.GetTenantIDFromContext(job.Context)
	if !exists {
		// This should never happen! Interceptor should either fail or set it!
		err := errors.Errorfc(codes.Unauthenticated, "Tenant ID is not present in context")
		log.InfraSec().InfraErr(err).Msg("List repeated schedule is not authenticated")
		return nil, err
	}

	filters, ulimit, uoffset, err := repeatedSchedFilter(&job.Payload)
	if err != nil {
		return nil, err
	}

	var offset, limit int
	offset, err = util.Uint32ToInt(uoffset)
	if err != nil {
		return nil, err
	}
	limit, err = util.Uint32ToInt(ulimit)
	if err != nil {
		return nil, err
	}

	repeatedSchedules, hasNext, totalElems, err := h.hScheduleCache.GetRepeatedSchedules(
		job.Context, tenantID, offset, limit, filters)
	if err != nil {
		return nil, err
	}

	schedulesAPI := make([]api.RepeatedSchedule, 0, len(repeatedSchedules))
	for _, rSched := range repeatedSchedules {
		obj := grpcToOpenAPIRepeatedSched(rSched, &inventory.GetResourceResponse_ResourceMetadata{})
		schedulesAPI = append(schedulesAPI, *obj)
	}

	schedsList := api.RepeatedSchedulesList{
		RepeatedSchedules: &schedulesAPI,
		HasNext:           &hasNext,
		TotalElements:     &totalElems,
	}

	payload := &types.Payload{Data: schedsList}
	return payload, nil
}

func castRepeatedSched(payload *types.Payload) (*api.RepeatedSchedule, error) {
	body, ok := payload.Data.(*api.RepeatedSchedule)
	if !ok {
		err := errors.Errorfc(codes.InvalidArgument,
			"body format is not RepeatedSchedule: %T",
			payload.Data,
		)
		log.InfraErr(err).Msgf("")
		return nil, err
	}
	return body, nil
}

func repeatedSchedResource(payload *types.Payload) (*inventory.Resource, error) {
	body, err := castRepeatedSched(payload)
	if err != nil {
		return nil, err
	}

	repeatedSched, err := openapiToGrpcRepeatedSched(body)
	if err != nil {
		return nil, err
	}

	req := &inventory.Resource{
		Resource: &inventory.Resource_Repeatedschedule{
			Repeatedschedule: repeatedSched,
		},
	}
	return req, nil
}

func repeatedSchedResourceID(payload *types.Payload) (string, error) {
	params, ok := payload.Params.(RepeatedSchedURLParams)
	if !ok {
		err := errors.Errorfc(codes.InvalidArgument, "RepeatedSchedURLParams incorrectly formatted: %T",
			payload.Data)
		log.InfraErr(err).Msg("could not parse job payload params")
		return "", err
	}
	return params.RepeatedSchedID, nil
}

func repeatedSchedFilter(
	payload *types.Payload,
) (filters *schedule_cache.Filters, limit, offset uint32, err error) {
	if payload.Data != nil {
		query, ok := payload.Data.(api.GetSchedulesRepeatedParams)
		if !ok {
			errInt := errors.Errorfc(codes.InvalidArgument,
				"GetSchedulesRepeatedParams incorrectly formatted: %T",
				payload.Data,
			)
			log.InfraErr(errInt).Msg("list operation")
			return nil, 0, 0, errInt
		}
		filters, limit, offset, err = castRepeatedSchedQueryList(&query)
		if err != nil {
			log.Debug().Msgf("error parsing query parameters in list operation: %s",
				err.Error())
			return nil, 0, 0, err
		}
		filters.Add(schedule_cache.FilterByTS(query.UnixEpoch))
	}

	return filters, limit, offset, nil
}

func repeatedSchedFieldMask(payload *types.Payload, operation types.Operation) (*fieldmaskpb.FieldMask, error) {
	body, ok := payload.Data.(*api.RepeatedSchedule)
	if !ok {
		errInt := errors.Errorfc(codes.InvalidArgument,
			"body format is not RepeatedSchedule: %T",
			payload.Data,
		)
		log.InfraErr(errInt).Msgf("")
		return nil, errInt
	}

	repeatedSchedRes, err := repeatedSchedResource(payload)
	if err != nil {
		return nil, err
	}

	var fieldmask *fieldmaskpb.FieldMask
	if operation == types.Patch {
		fieldmask, err = getRepeatedSchedFieldmask(body)
	} else {
		fieldmask, err = fieldmaskpb.New(repeatedSchedRes.GetRepeatedschedule(),
			maps.Values(OpenAPIRepeatedSchedToProto)...)
	}
	if err != nil {
		log.InfraErr(err).Msgf("could not create fieldmask")
		return nil, errors.Wrap(err)
	}

	return fieldmask, nil
}

func castRepeatedSchedQueryList(
	query *api.GetSchedulesRepeatedParams,
) (schedFilters *schedule_cache.Filters, limit, offset uint32, err error) {
	limit, offset, err = parsePagination(
		query.PageSize,
		query.Offset,
	)
	if err != nil {
		return nil, 0, 0, err
	}

	schedFilters = new(schedule_cache.Filters)
	nonEmptyFilters := 0
	if function.IsEmptyNullCase(query.RegionID) {
		schedFilters.Add(schedule_cache.HasNoRegion())
	} else if function.IsNotEmptyNullCase(query.RegionID) {
		schedFilters.Add(schedule_cache.HasRegionID(query.RegionID))
		nonEmptyFilters++
	}

	if function.IsEmptyNullCase(query.SiteID) {
		schedFilters.Add(schedule_cache.HasNoSite())
	} else if function.IsNotEmptyNullCase(query.SiteID) {
		schedFilters.Add(schedule_cache.HasSiteID(query.SiteID))
		nonEmptyFilters++
	}

	if function.IsEmptyNullCase(query.HostID) {
		schedFilters.Add(schedule_cache.HasNoHost())
	} else if function.IsNotEmptyNullCase(query.HostID) {
		schedFilters.Add(schedule_cache.HasHostID(query.HostID))
		nonEmptyFilters++
	}

	// Cannot apply multiple filters
	if nonEmptyFilters > 1 {
		return nil, 0, 0,
			errors.Errorfc(codes.InvalidArgument, "Only one of TargetHost, TargetSite and TargetRegion can be specified")
	}
	// No filters
	if schedFilters.Size() == 0 {
		schedFilters.Add(schedule_cache.NewStandardFilter(nil, "All"))
	}

	return schedFilters, limit, offset, nil
}

func getRepeatedSchedFieldmask(body *api.RepeatedSchedule) (*fieldmaskpb.FieldMask, error) {
	fieldList := getProtoFieldListFromOpenapiValue(*body, OpenAPIRepeatedSchedToProto)
	log.Debug().Msgf("Proto Valid Fields: %s", fieldList)
	return fieldmaskpb.New(&schedv1.RepeatedScheduleResource{}, fieldList...)
}

func OpenAPISchedStatusTogrpcSchedStatus(s api.ScheduleStatus) schedv1.ScheduleStatus {
	mapStatus := map[api.ScheduleStatus]schedv1.ScheduleStatus{
		api.SCHEDULESTATUSMAINTENANCE: schedv1.ScheduleStatus_SCHEDULE_STATUS_MAINTENANCE,
		api.SCHEDULESTATUSOSUPDATE:    schedv1.ScheduleStatus_SCHEDULE_STATUS_OS_UPDATE,
		api.SCHEDULESTATUSUNSPECIFIED: schedv1.ScheduleStatus_SCHEDULE_STATUS_UNSPECIFIED,
	}

	state, ok := mapStatus[s]
	if !ok {
		return schedv1.ScheduleStatus_SCHEDULE_STATUS_UNSPECIFIED
	}
	return state
}

func openapiToGrpcRepeatedSchedCron(body *api.RepeatedSchedule, sched *schedv1.RepeatedScheduleResource) {
	sched.CronMinutes = body.CronMinutes
	sched.CronHours = body.CronHours
	sched.CronDayWeek = body.CronDayWeek
	sched.CronDayMonth = body.CronDayMonth
	sched.CronMonth = body.CronMonth
}

func openapiToGrpcRepeatedSched(body *api.RepeatedSchedule) (*schedv1.RepeatedScheduleResource, error) {
	status := OpenAPISchedStatusTogrpcSchedStatus(body.ScheduleStatus)

	duration, err := util.IntToUint32(body.DurationSeconds)
	if err != nil {
		return nil, err
	}

	sched := &schedv1.RepeatedScheduleResource{
		ScheduleStatus:  status,
		DurationSeconds: duration,
	}

	if body.Name != nil {
		sched.Name = *body.Name
	}

	openapiToGrpcRepeatedSchedCron(body, sched)

	activeTargets := invcollections.Filter([]*string{body.TargetHostId, body.TargetSiteId, body.TargetRegionId}, isSet)
	if len(activeTargets) > 1 {
		activeTargetsErr := errors.Errorfc(codes.InvalidArgument,
			"only one target (site, host or region) must be provided for schedule resource")
		log.InfraErr(activeTargetsErr).Msg("error in parsing schedule resource")
		return nil, activeTargetsErr
	}

	if isSet(body.TargetRegionId) {
		sched.Relation = createRSRTargetRegion(*body.TargetRegionId)
	}
	if isSet(body.TargetHostId) {
		sched.Relation = createRSRTargetHost(*body.TargetHostId)
	}
	if isSet(body.TargetSiteId) {
		sched.Relation = createRSRTargetSite(*body.TargetSiteId)
	}

	if err = validator.ValidateMessage(sched); err != nil {
		log.InfraSec().InfraErr(err).Msg("could not validate inventory resource")
		return nil, errors.Wrap(err)
	}

	return sched, nil
}

func grpcSchedStatusToOpenAPISchedStatus(s schedv1.ScheduleStatus) api.ScheduleStatus {
	mapStatus := map[schedv1.ScheduleStatus]api.ScheduleStatus{
		schedv1.ScheduleStatus_SCHEDULE_STATUS_MAINTENANCE: api.SCHEDULESTATUSMAINTENANCE,
		schedv1.ScheduleStatus_SCHEDULE_STATUS_OS_UPDATE:   api.SCHEDULESTATUSOSUPDATE,
		schedv1.ScheduleStatus_SCHEDULE_STATUS_SHIPPING:    api.SCHEDULESTATUSUNSPECIFIED,
		schedv1.ScheduleStatus_SCHEDULE_STATUS_UNSPECIFIED: api.SCHEDULESTATUSUNSPECIFIED,
	}

	state, ok := mapStatus[s]
	if !ok {
		return api.SCHEDULESTATUSUNSPECIFIED
	}
	return state
}

func grpcToOpenAPIRepeatedSched(
	sched *schedv1.RepeatedScheduleResource,
	_ *inventory.GetResourceResponse_ResourceMetadata,
) *api.RepeatedSchedule {
	resID := sched.GetResourceId()
	resName := sched.GetName()
	cronMinutes := sched.GetCronMinutes()
	cronHours := sched.GetCronHours()
	cronDayMonth := sched.GetCronDayMonth()
	cronMonth := sched.GetCronMonth()
	cronDayWeek := sched.GetCronDayWeek()

	duration, err := util.Uint32ToInt(sched.GetDurationSeconds())
	if err != nil {
		log.Debug().Msgf("error in cast Uint32ToInt, duration_seconds field of repeated schedule: %s",
			err.Error())
	}

	status := grpcSchedStatusToOpenAPISchedStatus(sched.GetScheduleStatus())
	obj := api.RepeatedSchedule{
		RepeatedScheduleID: &resID,
		Name:               &resName,
		DurationSeconds:    duration,
		CronMinutes:        cronMinutes,
		CronHours:          cronHours,
		CronDayWeek:        cronDayWeek,
		CronDayMonth:       cronDayMonth,
		CronMonth:          cronMonth,
		ScheduleStatus:     status,
		ResourceId:         &resID,
		Timestamps:         GrpcToOpenAPITimestamps(sched),
	}

	if sched.GetTargetHost() != nil {
		resHostID := sched.GetTargetHost().GetResourceId()
		obj.TargetHostId = &resHostID
		obj.TargetHost = grpcToOpenAPIHostStatus(sched.GetTargetHost(), nil, nil, nil, nil, nil, nil)
	}

	if sched.GetTargetSite() != nil {
		resSiteID := sched.GetTargetSite().GetResourceId()
		obj.TargetSiteId = &resSiteID
		obj.TargetSite = grpcToOpenAPISite(sched.GetTargetSite(), nil)
	}

	if sched.GetTargetRegion() != nil {
		resRegionID := sched.GetTargetRegion().GetResourceId()
		obj.TargetRegionId = &resRegionID
		obj.TargetRegion = grpcToOpenAPIRegion(sched.GetTargetRegion(), nil)
	}

	return &obj
}

func createRSRTargetRegion(targetRegionID string) *schedv1.RepeatedScheduleResource_TargetRegion {
	return &schedv1.RepeatedScheduleResource_TargetRegion{
		TargetRegion: &locationv1.RegionResource{
			ResourceId: targetRegionID,
		},
	}
}

func createRSRTargetHost(targetHostID string) *schedv1.RepeatedScheduleResource_TargetHost {
	return &schedv1.RepeatedScheduleResource_TargetHost{
		TargetHost: &computev1.HostResource{
			ResourceId: targetHostID,
		},
	}
}

func createRSRTargetSite(targetSiteID string) *schedv1.RepeatedScheduleResource_TargetSite {
	return &schedv1.RepeatedScheduleResource_TargetSite{
		TargetSite: &locationv1.SiteResource{
			ResourceId: targetSiteID,
		},
	}
}
