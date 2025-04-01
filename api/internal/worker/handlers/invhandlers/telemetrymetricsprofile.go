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
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/util"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/validator"
)

// OpenAPITelemetryMetricsProfileToProto maps OpenAPI fields name to Proto fields name.
// The key is derived from the json property respectively of the
// structs TelemetryLogsProfile defined in edge-infrastructure-manager-openapi-types.gen.go.
var OpenAPITelemetryMetricsProfileToProto = map[string]string{
	"targetRegion":    telemetryv1.TelemetryProfileEdgeRegion,
	"targetSite":      telemetryv1.TelemetryProfileEdgeSite,
	"targetInstance":  telemetryv1.TelemetryProfileEdgeInstance,
	"metricsInterval": telemetryv1.TelemetryProfileFieldMetricsInterval,
	"metricsGroupId":  telemetryv1.TelemetryProfileEdgeGroup,
}

// OpenAPITelemetryMetricsProfileToProtoExcluded defines exclusion rules as there are some fields
// defined in the OpenAPI spec that are not currently mapped to the proto
// fields.
var OpenAPITelemetryMetricsProfileToProtoExcluded = map[string]struct{}{
	"profileId":    {}, // profileId must not be set from the API
	"metricsGroup": {}, // metricsGroup must not be set from the API
	"timestamps":   {}, // read-only field
}

type telemetryMetricsProfileHandler struct {
	invClient *clients.InventoryClientHandler
}

func NewTelemetryMetricsProfileHandler(invClient *clients.InventoryClientHandler) InventoryResource {
	return &telemetryMetricsProfileHandler{invClient: invClient}
}

func (t telemetryMetricsProfileHandler) Create(job *types.Job) (*types.Payload, error) {
	body, err := castTelemetryMetricsProfileAPI(&job.Payload)
	if err != nil {
		return nil, err
	}

	profile, err := openapiToGrpcTelemetryMetricsProfile(body)
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

	createdTPMetrics := invResp.GetTelemetryProfile()
	obj := grpcToOpenAPITelemetryMetricsProfile(createdTPMetrics)

	return &types.Payload{Data: obj}, nil
}

func (t telemetryMetricsProfileHandler) Get(job *types.Job) (*types.Payload, error) {
	req, err := telemetryMetricsProfileID(&job.Payload)
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

	obj := grpcToOpenAPITelemetryMetricsProfile(telemetryProfile)

	return &types.Payload{Data: obj}, nil
}

func (t telemetryMetricsProfileHandler) Update(job *types.Job) (*types.Payload, error) {
	resID, err := telemetryMetricsProfileID(&job.Payload)
	if err != nil {
		return nil, err
	}

	fieldmask, err := telemetryMetricsProfileFieldMask(&job.Payload, job.Operation)
	if err != nil {
		return nil, err
	}

	res, err := telemetryMetricsProfileResource(&job.Payload)
	if err != nil {
		return nil, err
	}

	invResp, err := t.invClient.InvClient.Update(job.Context, resID, fieldmask, res)
	if err != nil {
		return nil, err
	}

	updatedTelemetryMetricsProfile := invResp.GetTelemetryProfile()
	obj := grpcToOpenAPITelemetryMetricsProfile(updatedTelemetryMetricsProfile)
	obj.ProfileId = &resID
	return &types.Payload{Data: obj}, nil
}

func (t telemetryMetricsProfileHandler) Delete(job *types.Job) error {
	req, err := telemetryMetricsProfileID(&job.Payload)
	if err != nil {
		return err
	}

	_, err = t.invClient.InvClient.Delete(job.Context, req)
	if err != nil {
		return err
	}

	return nil
}

func isInheritedTelemetryMetricsRequest(query *api.GetTelemetryProfilesMetricsParams) bool {
	return query != nil && query.ShowInherited != nil && *query.ShowInherited
}

func (t telemetryMetricsProfileHandler) List(job *types.Job) (*types.Payload, error) {
	var query *api.GetTelemetryProfilesMetricsParams
	if job.Payload.Data != nil {
		params, ok := job.Payload.Data.(api.GetTelemetryProfilesMetricsParams)
		if !ok {
			err := errors.Errorfc(codes.InvalidArgument,
				"GetTelemetryProfilesMetricsParams incorrectly formatted: %T",
				job.Payload.Data,
			)
			log.InfraErr(err).Msg("list operation")
			return nil, err
		}
		query = &params
	}
	hasNext := false
	totalElems := 0
	var metricsProfiles []api.TelemetryMetricsProfile
	if !isInheritedTelemetryMetricsRequest(query) {
		filter, err := t.telemetryMetricsProfileFilter(query)
		if err != nil {
			return nil, err
		}

		resp, err := t.invClient.InvClient.List(job.Context, filter)
		if err != nil {
			return nil, err
		}

		metricsProfiles = make([]api.TelemetryMetricsProfile, len(resp.GetResources())) // pre-allocate proper length
		for i, res := range resp.GetResources() {
			profile, err := castToTelemetryProfile(res)
			if err != nil {
				return nil, err
			}
			metricsProfiles[i] = *grpcToOpenAPITelemetryMetricsProfile(profile)
		}
		hasNext = resp.GetHasNext()
		totalElems = int(resp.GetTotalElements())
	} else {
		var telProfiles []*telemetryv1.TelemetryProfile
		err := error(nil)
		telProfiles, totalElems, hasNext, err = t.listInheritedTelemetryMetrics(job.Context, query)
		if err != nil {
			return nil, err
		}
		metricsProfiles = make([]api.TelemetryMetricsProfile, len(telProfiles))
		for i, res := range telProfiles {
			metricsProfiles[i] = *grpcToOpenAPITelemetryMetricsProfile(res)
		}
	}
	list := api.TelemetryMetricsProfileList{
		TelemetryMetricsProfiles: &metricsProfiles,
		HasNext:                  &hasNext,
		TotalElements:            &totalElems,
	}
	payload := &types.Payload{Data: list}
	return payload, nil
}

func telemetryMetricsProfileID(payload *types.Payload) (string, error) {
	params, ok := payload.Params.(TelemetryMetricsProfileURLParams)
	if !ok {
		err := errors.Errorfc(codes.InvalidArgument, "TelemetryMetricsProfileURLParams incorrectly formatted: %T",
			payload.Data)
		log.InfraErr(err).Msg("could not parse job payload params")
		return "", err
	}
	return params.TelemetryMetricsProfileID, nil
}

func telemetryMetricsProfileFieldMask(payload *types.Payload, operation types.Operation) (*fieldmaskpb.FieldMask, error) {
	body, ok := payload.Data.(*api.TelemetryMetricsProfile)
	if !ok {
		err := errors.Errorfc(codes.InvalidArgument,
			"body format is not TelemetryMetricsProfile: %T",
			payload.Data,
		)
		log.InfraErr(err).Msgf("")
		return nil, err
	}

	telemetryMetricsProfileRes, err := telemetryMetricsProfileResource(payload)
	if err != nil {
		return nil, err
	}
	var fieldmask *fieldmaskpb.FieldMask
	if operation == types.Patch {
		fieldmask, err = getTelemetryMetricsProfileFieldmask(body)
	} else {
		fieldmask, err = fieldmaskpb.New(telemetryMetricsProfileRes.GetTelemetryProfile(),
			maps.Values(OpenAPITelemetryMetricsProfileToProto)...)
	}

	if err != nil {
		log.InfraErr(err).Msgf("could not create fieldmask")
		return nil, errors.Wrap(err)
	}

	return fieldmask, nil
}

func getTelemetryMetricsProfileFieldmask(body *api.TelemetryMetricsProfile) (*fieldmaskpb.FieldMask, error) {
	fieldList := getProtoFieldListFromOpenapiValue(*body, OpenAPITelemetryMetricsProfileToProto)
	log.Debug().Msgf("Proto Valid Fields: %s", fieldList)
	return fieldmaskpb.New(&telemetryv1.TelemetryProfile{}, fieldList...)
}

func telemetryMetricsProfileResource(payload *types.Payload) (*inventory.Resource, error) {
	body, err := castTelemetryMetricsProfileAPI(payload)
	if err != nil {
		return nil, err
	}

	metricsProfile, err := openapiToGrpcTelemetryMetricsProfile(body)
	if err != nil {
		return nil, err
	}

	req := &inventory.Resource{
		Resource: &inventory.Resource_TelemetryProfile{
			TelemetryProfile: metricsProfile,
		},
	}
	return req, nil
}

func castTelemetryMetricsProfileAPI(payload *types.Payload) (*api.TelemetryMetricsProfile, error) {
	body, ok := payload.Data.(*api.TelemetryMetricsProfile)
	if !ok {
		err := errors.Errorfc(codes.InvalidArgument,
			"body format is not TelemetryMetricsProfile: %T",
			payload.Data,
		)
		log.InfraErr(err).Msgf("")
		return nil, err
	}
	return body, nil
}

func openapiToGrpcTelemetryMetricsProfile(body *api.TelemetryMetricsProfile) (*telemetryv1.TelemetryProfile, error) {
	metricsInterval, err := util.IntToUint32(body.MetricsInterval)
	if err != nil {
		return nil, err
	}

	profile := &telemetryv1.TelemetryProfile{
		Kind: telemetryv1.TelemetryResourceKind_TELEMETRY_RESOURCE_KIND_METRICS,
		Group: &telemetryv1.TelemetryGroupResource{
			ResourceId: body.MetricsGroupId,
		},
		MetricsInterval: metricsInterval,
	}

	err = validateAndSetTelemetryProfileRelations(profile,
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

func grpcToOpenAPITelemetryMetricsProfile(profile *telemetryv1.TelemetryProfile) *api.TelemetryMetricsProfile {
	profileID := profile.GetResourceId()

	obj := api.TelemetryMetricsProfile{
		MetricsInterval: int(profile.GetMetricsInterval()),
		ProfileId:       &profileID,
		Timestamps:      GrpcToOpenAPITimestamps(profile),
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
		obj.MetricsGroup = grpcToOpenAPITelemetryMetricsGroup(profile.GetGroup())
		obj.MetricsGroupId = profile.GetGroup().GetResourceId()
	}

	return &obj
}

// Should be called only for request without inherited telemetry profiles.
func (t telemetryMetricsProfileHandler) telemetryMetricsProfileFilter(
	query *api.GetTelemetryProfilesMetricsParams,
) (*inventory.ResourceFilter, error) {
	req := &inventory.ResourceFilter{
		Resource: &inventory.Resource{
			Resource: &inventory.Resource_TelemetryProfile{},
		},
	}
	if query != nil {
		err := t.castTelemetryMetricsProfileQueryList(query, req)
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

func (t telemetryMetricsProfileHandler) castTelemetryMetricsProfileQueryList(
	query *api.GetTelemetryProfilesMetricsParams,
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
		telemetryv1.TelemetryResourceKind_TELEMETRY_RESOURCE_KIND_METRICS,
		query.InstanceId,
		query.SiteId,
		query.RegionId)
	return nil
}

// Should be called only when requesting inherited metadata.
func (t telemetryMetricsProfileHandler) listInheritedTelemetryMetrics(
	ctx context.Context,
	query *api.GetTelemetryProfilesMetricsParams,
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
				telemetryv1.TelemetryProfileFieldKind, telemetryv1.TelemetryResourceKind_TELEMETRY_RESOURCE_KIND_METRICS),
			orderBy, limit, offset)
		if err != nil {
			log.Debug().Msgf("error querying inherited metrics telemetry profiles: %s", err.Error())
			return nil, 0, false, err
		}
		telProfiles = resp.GetTelemetryProfiles()
		totalElements = int(resp.GetTotalElements())
		// Safe to cast to offset to int since it comes from an int already.
		more = int(offset)+len(resp.GetTelemetryProfiles()) < totalElements
	}
	return telProfiles, totalElements, more, nil
}
