// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package invhandlers

import (
	"context"
	"fmt"

	"golang.org/x/exp/maps"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	"github.com/open-edge-platform/infra-core/api/internal/types"
	"github.com/open-edge-platform/infra-core/api/internal/worker/clients"
	"github.com/open-edge-platform/infra-core/api/pkg/api/v0"
	inventory "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/inventory/v1"
	telemetryv1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/telemetry/v1"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/errors"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/validator"
)

// OpenAPITelemetryLogsProfileToProto maps OpenAPI fields name to Proto fields name.
// The key is derived from the json property respectively of the
// structs TelemetryLogsProfile defined in edge-infrastructure-manager-openapi-types.gen.go.
var OpenAPITelemetryLogsProfileToProto = map[string]string{
	"targetRegion":   telemetryv1.TelemetryProfileEdgeRegion,
	"targetSite":     telemetryv1.TelemetryProfileEdgeSite,
	"targetInstance": telemetryv1.TelemetryProfileEdgeInstance,
	"logLevel":       telemetryv1.TelemetryProfileFieldLogLevel,
	"logsGroupId":    telemetryv1.TelemetryProfileEdgeGroup,
}

// OpenAPITelemetryLogsProfileToProtoExcluded defines exclusion rules as there are some fields
// defined in the OpenAPI spec that are not currently mapped to the proto
// fields.
var OpenAPITelemetryLogsProfileToProtoExcluded = map[string]struct{}{
	"profileId":  {}, // profileId must not be set from the API
	"logsGroup":  {}, // logsGroup must not be set from the API
	"timestamps": {}, // read-only field
}

type telemetryLogsProfileHandler struct {
	invClient *clients.InventoryClientHandler
}

func NewTelemetryLogsProfileHandler(invClient *clients.InventoryClientHandler) InventoryResource {
	return &telemetryLogsProfileHandler{invClient: invClient}
}

func (t telemetryLogsProfileHandler) Create(job *types.Job) (*types.Payload, error) {
	body, err := castTelemetryLogsProfileAPI(&job.Payload)
	if err != nil {
		return nil, err
	}

	profile, err := openapiToGrpcTelemetryLogsProfile(body)
	if err != nil {
		return nil, err
	}

	req := &inventory.Resource{
		Resource: &inventory.Resource_TelemetryProfile{
			TelemetryProfile: profile,
		},
	}

	invResp, err := t.invClient.InvClient.Create(job.Context, req)
	if err != nil {
		return nil, err
	}

	createdTPLogs := invResp.GetTelemetryProfile()
	obj := grpcToOpenAPITelemetryLogsProfile(createdTPLogs)

	return &types.Payload{Data: obj}, err
}

func (t telemetryLogsProfileHandler) Get(job *types.Job) (*types.Payload, error) {
	req, err := telemetryLogsProfileID(&job.Payload)
	if err != nil {
		return nil, err
	}

	invResp, err := t.invClient.InvClient.Get(job.Context, req)
	if err != nil {
		return nil, err
	}

	telemetryProfile, err := castToTelemetryProfile(invResp)
	if err != nil {
		return nil, err
	}

	obj := grpcToOpenAPITelemetryLogsProfile(telemetryProfile)

	return &types.Payload{Data: obj}, nil
}

func (t telemetryLogsProfileHandler) Update(job *types.Job) (*types.Payload, error) {
	resID, err := telemetryLogsProfileID(&job.Payload)
	if err != nil {
		return nil, err
	}

	fieldmask, err := telemetryLogsProfileFieldMask(&job.Payload, job.Operation)
	if err != nil {
		return nil, err
	}

	res, err := telemetryLogsProfileResource(&job.Payload)
	if err != nil {
		return nil, err
	}

	invResp, err := t.invClient.InvClient.Update(job.Context, resID, fieldmask, res)
	if err != nil {
		return nil, err
	}

	updatedTelemetryLogsProfile := invResp.GetTelemetryProfile()
	obj := grpcToOpenAPITelemetryLogsProfile(updatedTelemetryLogsProfile)
	obj.ProfileId = &resID
	return &types.Payload{Data: obj}, nil
}

func (t telemetryLogsProfileHandler) Delete(job *types.Job) error {
	req, err := telemetryLogsProfileID(&job.Payload)
	if err != nil {
		return err
	}

	_, err = t.invClient.InvClient.Delete(job.Context, req)
	if err != nil {
		return err
	}

	return nil
}

func isInheritedTelemetryLogsRequest(query *api.GetTelemetryProfilesLogsParams) bool {
	return query != nil && query.ShowInherited != nil && *query.ShowInherited
}

func (t telemetryLogsProfileHandler) List(job *types.Job) (*types.Payload, error) {
	var query *api.GetTelemetryProfilesLogsParams
	if job.Payload.Data != nil {
		params, ok := job.Payload.Data.(api.GetTelemetryProfilesLogsParams)
		if !ok {
			err := errors.Errorfc(codes.InvalidArgument,
				"GetTelemetryProfilesLogsParams incorrectly formatted: %T",
				job.Payload.Data,
			)
			log.InfraErr(err).Msg("list operation")
			return nil, err
		}
		query = &params
	}

	hasNext := false
	totalElems := 0
	var logsProfiles []api.TelemetryLogsProfile
	if !isInheritedTelemetryLogsRequest(query) {
		filter, err := t.telemetryLogsProfileFilter(query)
		if err != nil {
			return nil, err
		}

		resp, err := t.invClient.InvClient.List(job.Context, filter)
		if err != nil {
			return nil, err
		}

		logsProfiles = make([]api.TelemetryLogsProfile, len(resp.GetResources())) // pre-allocate proper length
		for i, res := range resp.GetResources() {
			profile, err := castToTelemetryProfile(res)
			if err != nil {
				return nil, err
			}
			logsProfiles[i] = *grpcToOpenAPITelemetryLogsProfile(profile)
		}
		hasNext = resp.GetHasNext()
		totalElems = int(resp.GetTotalElements())
	} else {
		var telProfiles []*telemetryv1.TelemetryProfile
		err := error(nil)
		telProfiles, totalElems, hasNext, err = t.listInheritedTelemetryLogs(job.Context, query)
		if err != nil {
			return nil, err
		}
		logsProfiles = make([]api.TelemetryLogsProfile, len(telProfiles))
		for i, res := range telProfiles {
			logsProfiles[i] = *grpcToOpenAPITelemetryLogsProfile(res)
		}
	}

	list := api.TelemetryLogsProfileList{
		TelemetryLogsProfiles: &logsProfiles,
		HasNext:               &hasNext,
		TotalElements:         &totalElems,
	}
	payload := &types.Payload{Data: list}
	return payload, nil
}

func castTelemetryLogsProfileAPI(payload *types.Payload) (*api.TelemetryLogsProfile, error) {
	body, ok := payload.Data.(*api.TelemetryLogsProfile)
	if !ok {
		err := errors.Errorfc(codes.InvalidArgument,
			"body format is not TelemetryLogsProfile: %T",
			payload.Data,
		)
		log.InfraErr(err).Msgf("")
		return nil, err
	}
	return body, nil
}

func telemetryLogsProfileID(payload *types.Payload) (string, error) {
	params, ok := payload.Params.(TelemetryLogsProfileURLParams)
	if !ok {
		err := errors.Errorfc(codes.InvalidArgument, "TelemetryLogsProfileURLParams incorrectly formatted: %T",
			payload.Data)
		log.InfraErr(err).Msg("could not parse job payload params")
		return "", err
	}
	return params.TelemetryLogsProfileID, nil
}

func telemetryLogsProfileFieldMask(
	payload *types.Payload,
	operation types.Operation,
) (*fieldmaskpb.FieldMask, error) {
	body, ok := payload.Data.(*api.TelemetryLogsProfile)
	if !ok {
		err := errors.Errorfc(codes.InvalidArgument,
			"body format is not TelemetryLogsProfile: %T",
			payload.Data,
		)
		log.InfraErr(err).Msgf("")
		return nil, err
	}

	telemetryLogsProfileRes, err := telemetryLogsProfileResource(payload)
	if err != nil {
		return nil, err
	}
	var fieldmask *fieldmaskpb.FieldMask
	if operation == types.Patch {
		fieldmask, err = getTelemetryLogsProfileFieldmask(body)
	} else {
		fieldmask, err = fieldmaskpb.New(telemetryLogsProfileRes.GetTelemetryProfile(),
			maps.Values(OpenAPITelemetryLogsProfileToProto)...)
	}
	if err != nil {
		log.InfraErr(err).Msgf("could not create fieldmask")
		return nil, errors.Wrap(err)
	}

	return fieldmask, nil
}

func getTelemetryLogsProfileFieldmask(body *api.TelemetryLogsProfile) (*fieldmaskpb.FieldMask, error) {
	fieldList := getProtoFieldListFromOpenapiValue(*body, OpenAPITelemetryLogsProfileToProto)
	log.Debug().Msgf("Proto Valid Fields: %s", fieldList)
	return fieldmaskpb.New(&telemetryv1.TelemetryProfile{}, fieldList...)
}

func telemetryLogsProfileResource(payload *types.Payload) (*inventory.Resource, error) {
	body, err := castTelemetryLogsProfileAPI(payload)
	if err != nil {
		return nil, err
	}

	log.Debug().Msgf("TelemetryLogsProfile: %v", body)

	logsProfile, err := openapiToGrpcTelemetryLogsProfile(body)
	if err != nil {
		return nil, err
	}

	req := &inventory.Resource{
		Resource: &inventory.Resource_TelemetryProfile{
			TelemetryProfile: logsProfile,
		},
	}
	return req, nil
}

func openapiToGrpcTelemetryLogsProfile(body *api.TelemetryLogsProfile) (*telemetryv1.TelemetryProfile, error) {
	profile := &telemetryv1.TelemetryProfile{
		Kind: telemetryv1.TelemetryResourceKind_TELEMETRY_RESOURCE_KIND_LOGS,
		Group: &telemetryv1.TelemetryGroupResource{
			ResourceId: body.LogsGroupId,
		},
		LogLevel: telemetryLogLevelAPItoGRPC(body.LogLevel),
	}

	err := validateAndSetTelemetryProfileRelations(profile,
		body.TargetInstance, body.TargetSite, body.TargetRegion)
	if err != nil {
		return nil, err
	}

	err = validator.ValidateMessage(profile)
	if err != nil {
		log.InfraErr(err).Msg("could not validate inventory resource")
		return nil, errors.Wrap(err)
	}

	return profile, nil
}

func grpcToOpenAPITelemetryLogsProfile(profile *telemetryv1.TelemetryProfile) *api.TelemetryLogsProfile {
	profileID := profile.GetResourceId()

	obj := api.TelemetryLogsProfile{
		LogLevel:   telemetryLogLevelGRPCtoAPI(profile.GetLogLevel()),
		ProfileId:  &profileID,
		Timestamps: GrpcToOpenAPITimestamps(profile),
	}

	if profile.GetInstance() != nil {
		resInstID := profile.GetInstance().GetResourceId()
		obj.TargetInstance = &resInstID
	}

	if profile.GetSite() != nil {
		resSiteID := profile.GetSite().GetResourceId()
		obj.TargetSite = &resSiteID
	}

	if profile.GetRegion() != nil {
		resRegionID := profile.GetRegion().GetResourceId()
		obj.TargetRegion = &resRegionID
	}

	if profile.GetGroup() != nil {
		obj.LogsGroup = grpcToOpenAPITelemetryLogsGroup(profile.GetGroup())
		obj.LogsGroupId = profile.GetGroup().GetResourceId()
	}

	return &obj
}

func telemetryLogLevelAPItoGRPC(apiLevel api.TelemetrySeverityLevel) telemetryv1.SeverityLevel {
	levelMap := map[api.TelemetrySeverityLevel]telemetryv1.SeverityLevel{
		api.TELEMETRYSEVERITYLEVELCRITICAL: telemetryv1.SeverityLevel_SEVERITY_LEVEL_CRITICAL,
		api.TELEMETRYSEVERITYLEVELERROR:    telemetryv1.SeverityLevel_SEVERITY_LEVEL_ERROR,
		api.TELEMETRYSEVERITYLEVELINFO:     telemetryv1.SeverityLevel_SEVERITY_LEVEL_INFO,
		api.TELEMETRYSEVERITYLEVELDEBUG:    telemetryv1.SeverityLevel_SEVERITY_LEVEL_DEBUG,
		api.TELEMETRYSEVERITYLEVELWARN:     telemetryv1.SeverityLevel_SEVERITY_LEVEL_WARN,
	}

	grpcLevel, ok := levelMap[apiLevel]
	if !ok {
		return telemetryv1.SeverityLevel_SEVERITY_LEVEL_UNSPECIFIED
	}
	return grpcLevel
}

func telemetryLogLevelGRPCtoAPI(grpcLevel telemetryv1.SeverityLevel) api.TelemetrySeverityLevel {
	levelMap := map[telemetryv1.SeverityLevel]api.TelemetrySeverityLevel{
		telemetryv1.SeverityLevel_SEVERITY_LEVEL_CRITICAL: api.TELEMETRYSEVERITYLEVELCRITICAL,
		telemetryv1.SeverityLevel_SEVERITY_LEVEL_ERROR:    api.TELEMETRYSEVERITYLEVELERROR,
		telemetryv1.SeverityLevel_SEVERITY_LEVEL_INFO:     api.TELEMETRYSEVERITYLEVELINFO,
		telemetryv1.SeverityLevel_SEVERITY_LEVEL_DEBUG:    api.TELEMETRYSEVERITYLEVELDEBUG,
		telemetryv1.SeverityLevel_SEVERITY_LEVEL_WARN:     api.TELEMETRYSEVERITYLEVELWARN,
	}

	apiLevel, ok := levelMap[grpcLevel]
	if !ok {
		return api.TELEMETRYSEVERITYLEVELCRITICAL
	}
	return apiLevel
}

// Should be called only for request without inherited telemetry profiles.
func (t telemetryLogsProfileHandler) telemetryLogsProfileFilter(
	query *api.GetTelemetryProfilesLogsParams,
) (*inventory.ResourceFilter, error) {
	req := &inventory.ResourceFilter{
		Resource: &inventory.Resource{
			Resource: &inventory.Resource_TelemetryProfile{},
		},
	}
	if query != nil {
		err := t.castTelemetryLogsProfileQueryList(query, req)
		if err != nil {
			log.Debug().Msgf("error parsing query parameters in list operation: %s",
				err.Error())
			return nil, err
		}
	}
	if err := validator.ValidateMessage(req); err != nil {
		log.InfraSec().InfraErr(err).Msg("failed to validate query params")
		return nil, errors.Wrap(err)
	}
	return req, nil
}

func (t telemetryLogsProfileHandler) castTelemetryLogsProfileQueryList(
	query *api.GetTelemetryProfilesLogsParams,
	req *inventory.ResourceFilter,
) error {
	err := error(nil)
	req.Limit, req.Offset, err = parsePagination(
		query.PageSize,
		query.Offset,
	)
	if err != nil {
		return err
	}

	err = validateTelemetryProfileRelations(query.InstanceId, query.SiteId, query.RegionId, false)
	if err != nil {
		return err
	}

	if query.OrderBy != nil {
		req.OrderBy = *query.OrderBy
	}
	req.Filter = telemetryProfileFilter(
		telemetryv1.TelemetryResourceKind_TELEMETRY_RESOURCE_KIND_LOGS,
		query.InstanceId,
		query.SiteId,
		query.RegionId)
	return nil
}

// Should be called only when requesting inherited metadata.
func (t telemetryLogsProfileHandler) listInheritedTelemetryLogs(
	ctx context.Context,
	query *api.GetTelemetryProfilesLogsParams,
) (telProfiles []*telemetryv1.TelemetryProfile, totalElements int, more bool, err error) {
	if query != nil {
		limit, offset, err := parsePagination(
			query.PageSize,
			query.Offset,
		)
		if err != nil {
			return nil, 0, false, err
		}

		err = validateTelemetryProfileRelations(query.InstanceId, query.SiteId, query.RegionId, false)
		if err != nil {
			return nil, 0, false, err
		}
		var inheritBy inventory.ListInheritedTelemetryProfilesRequest_InheritBy
		switch {
		case !isUnset(query.InstanceId):
			inheritBy = inventory.ListInheritedTelemetryProfilesRequest_InheritBy{
				Id: &inventory.ListInheritedTelemetryProfilesRequest_InheritBy_InstanceId{
					InstanceId: *query.InstanceId,
				},
			}
		case !isUnset(query.SiteId):
			inheritBy = inventory.ListInheritedTelemetryProfilesRequest_InheritBy{
				Id: &inventory.ListInheritedTelemetryProfilesRequest_InheritBy_SiteId{
					SiteId: *query.SiteId,
				},
			}
		case !isUnset(query.RegionId):
			inheritBy = inventory.ListInheritedTelemetryProfilesRequest_InheritBy{
				Id: &inventory.ListInheritedTelemetryProfilesRequest_InheritBy_RegionId{
					RegionId: *query.RegionId,
				},
			}
		}

		orderBy := ""
		if query.OrderBy != nil {
			orderBy = *query.OrderBy
		}

		resp, err := t.invClient.InvClient.ListInheritedTelemetryProfiles(
			ctx,
			&inheritBy,
			fmt.Sprintf("%s = %s",
				telemetryv1.TelemetryProfileFieldKind, telemetryv1.TelemetryResourceKind_TELEMETRY_RESOURCE_KIND_LOGS),
			orderBy, limit, offset)
		if err != nil {
			log.Debug().Msgf("error querying inherited logs telemetry profiles: %s", err.Error())
			return nil, 0, false, err
		}
		telProfiles = resp.GetTelemetryProfiles()
		totalElements = int(resp.GetTotalElements())
		// Safe to cast to offset to int since it comes from an int already.
		more = int(offset)+len(resp.GetTelemetryProfiles()) < totalElements
	}
	return telProfiles, totalElements, more, nil
}
