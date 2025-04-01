// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package invhandlers

import (
	"fmt"

	"google.golang.org/grpc/codes"

	"github.com/open-edge-platform/infra-core/api/internal/types"
	"github.com/open-edge-platform/infra-core/api/internal/worker/clients"
	"github.com/open-edge-platform/infra-core/api/pkg/api/v0"
	inventory "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/inventory/v1"
	telemetryv1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/telemetry/v1"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/errors"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/validator"
)

// OpenAPITelemetryLogsGroupResourceToProto maps OpenAPI fields name to Proto fields name.
// The key is derived from the json property respectively of the
// structs TelemetryGroupResource defined in edge-infrastructure-manager-openapi-types.gen.go.
var OpenAPITelemetryLogsGroupResourceToProto = map[string]string{
	"name":          telemetryv1.TelemetryGroupResourceFieldName,
	"collectorKind": telemetryv1.TelemetryGroupResourceFieldCollectorKind,
}

// OpenAPITelemetryLogsGroupToProtoExcluded defines exclusion rules as there are some fields
// defined in the OpenAPI spec that are not currently mapped to the proto
// fields especially when doing translation from OAPI to proto.
var OpenAPITelemetryLogsGroupToProtoExcluded = map[string]struct{}{
	"telemetryLogsGroupId": {}, // telemetryLogsGroupId should never be used when translating from OAPI to proto.
	"groups":               {}, // groups should never be used when translating from OAPI to proto.
	"timestamps":           {}, // read-only field
}

type telemetryLogsGroupHandler struct {
	invClient *clients.InventoryClientHandler
}

func NewTelemetryLogsGroupHandler(invClient *clients.InventoryClientHandler) InventoryResource {
	return &telemetryLogsGroupHandler{invClient: invClient}
}

func (t telemetryLogsGroupHandler) Create(job *types.Job) (*types.Payload, error) {
	body, err := castTelemetryLogsGroupAPI(&job.Payload)
	if err != nil {
		return nil, err
	}

	telemetryGroup, err := openapiToGrpcTelemetryLogsGroup(body)
	if err != nil {
		return nil, err
	}

	req := &inventory.Resource{
		Resource: &inventory.Resource_TelemetryGroup{
			TelemetryGroup: telemetryGroup,
		},
	}

	invResp, err := t.invClient.InvClient.Create(job.Context, req)
	if err != nil {
		return nil, err
	}

	createdTGLogs := invResp.GetTelemetryGroup()
	obj := grpcToOpenAPITelemetryLogsGroup(createdTGLogs)

	return &types.Payload{Data: obj}, err
}

func (t telemetryLogsGroupHandler) Get(job *types.Job) (*types.Payload, error) {
	req, err := telemetryLogsGroupID(&job.Payload)
	if err != nil {
		return nil, err
	}

	invResp, err := t.invClient.InvClient.Get(job.Context, req)
	if err != nil {
		return nil, err
	}

	telemetryGroup, err := castToTelemetryGroupResource(invResp)
	if err != nil {
		return nil, err
	}

	obj := grpcToOpenAPITelemetryLogsGroup(telemetryGroup)

	return &types.Payload{Data: obj}, nil
}

func (t telemetryLogsGroupHandler) Update(job *types.Job) (*types.Payload, error) {
	err := errors.Errorfc(codes.Unimplemented, "%s operation not supported", job.Operation)
	return nil, err
}

func (t telemetryLogsGroupHandler) Delete(job *types.Job) error {
	req, err := telemetryLogsGroupID(&job.Payload)
	if err != nil {
		return err
	}

	_, err = t.invClient.InvClient.Delete(job.Context, req)
	if err != nil {
		return err
	}

	return nil
}

func (t telemetryLogsGroupHandler) List(job *types.Job) (*types.Payload, error) {
	filter, err := telemetryLogsGroupFilter(&job.Payload)
	if err != nil {
		return nil, err
	}

	resp, err := t.invClient.InvClient.List(job.Context, filter)
	if err != nil {
		return nil, err
	}

	logsGroups := make([]api.TelemetryLogsGroup, 0, len(resp.GetResources())) // pre-allocate proper length
	for _, res := range resp.GetResources() {
		group, err := castToTelemetryGroupResource(res)
		if err != nil {
			return nil, err
		}
		obj := grpcToOpenAPITelemetryLogsGroup(group)
		logsGroups = append(logsGroups, *obj)
	}

	hasNext := resp.GetHasNext()
	totalElems := int(resp.GetTotalElements())
	telemetryLogsGroupList := api.TelemetryLogsGroupList{
		TelemetryLogsGroups: &logsGroups,
		HasNext:             &hasNext,
		TotalElements:       &totalElems,
	}

	payload := &types.Payload{Data: telemetryLogsGroupList}
	return payload, nil
}

func castTelemetryLogsGroupAPI(payload *types.Payload) (*api.TelemetryLogsGroup, error) {
	body, ok := payload.Data.(*api.TelemetryLogsGroup)
	if !ok {
		err := errors.Errorfc(codes.InvalidArgument,
			"body format is not TelemetryLogsGroup: %T",
			payload.Data,
		)
		log.InfraErr(err).Msgf("")
		return nil, err
	}
	return body, nil
}

func telemetryLogsGroupID(payload *types.Payload) (string, error) {
	params, ok := payload.Params.(TelemetryLogsGroupURLParams)
	if !ok {
		err := errors.Errorfc(codes.InvalidArgument, "TelemetryLogsGroupURLParams incorrectly formatted: %T",
			payload.Data)
		log.InfraErr(err).Msg("could not parse job payload params")
		return "", err
	}
	return params.TelemetryLogsGroupID, nil
}

func telemetryCollectorKindGRPCtoAPI(grpcCollectorKind telemetryv1.CollectorKind) api.TelemetryCollectorKind {
	kindMap := map[telemetryv1.CollectorKind]api.TelemetryCollectorKind{
		telemetryv1.CollectorKind_COLLECTOR_KIND_UNSPECIFIED: api.TELEMETRYCOLLECTORKINDUNSPECIFIED,
		telemetryv1.CollectorKind_COLLECTOR_KIND_HOST:        api.TELEMETRYCOLLECTORKINDHOST,
		telemetryv1.CollectorKind_COLLECTOR_KIND_CLUSTER:     api.TELEMETRYCOLLECTORKINDCLUSTER,
	}

	apiKind, ok := kindMap[grpcCollectorKind]
	if !ok {
		return api.TELEMETRYCOLLECTORKINDUNSPECIFIED
	}

	return apiKind
}

func telemetryCollectorKindAPItoGRPC(apiCollectorKind api.TelemetryCollectorKind) telemetryv1.CollectorKind {
	kindMap := map[api.TelemetryCollectorKind]telemetryv1.CollectorKind{
		api.TELEMETRYCOLLECTORKINDUNSPECIFIED: telemetryv1.CollectorKind_COLLECTOR_KIND_UNSPECIFIED,
		api.TELEMETRYCOLLECTORKINDHOST:        telemetryv1.CollectorKind_COLLECTOR_KIND_HOST,
		api.TELEMETRYCOLLECTORKINDCLUSTER:     telemetryv1.CollectorKind_COLLECTOR_KIND_CLUSTER,
	}

	grpcKind, ok := kindMap[apiCollectorKind]
	if !ok {
		return telemetryv1.CollectorKind_COLLECTOR_KIND_UNSPECIFIED
	}

	return grpcKind
}

func openapiToGrpcTelemetryLogsGroup(body *api.TelemetryLogsGroup) (*telemetryv1.TelemetryGroupResource, error) {
	telemetry := &telemetryv1.TelemetryGroupResource{
		Name:          body.Name,
		Groups:        body.Groups,
		Kind:          telemetryv1.TelemetryResourceKind_TELEMETRY_RESOURCE_KIND_LOGS,
		CollectorKind: telemetryCollectorKindAPItoGRPC(body.CollectorKind),
	}

	err := validator.ValidateMessage(telemetry)
	if err != nil {
		log.InfraErr(err).Msg("could not validate inventory resource")
		return nil, errors.Wrap(err)
	}

	return telemetry, nil
}

func grpcToOpenAPITelemetryLogsGroup(
	telemetryGroup *telemetryv1.TelemetryGroupResource,
) *api.TelemetryLogsGroup {
	resID := telemetryGroup.GetResourceId()
	resName := telemetryGroup.GetName()
	resGroups := telemetryGroup.GetGroups()

	obj := api.TelemetryLogsGroup{
		TelemetryLogsGroupId: &resID,
		Groups:               resGroups,
		Name:                 resName,
		CollectorKind:        telemetryCollectorKindGRPCtoAPI(telemetryGroup.GetCollectorKind()),
		Timestamps:           GrpcToOpenAPITimestamps(telemetryGroup),
	}

	return &obj
}

func telemetryLogsGroupFilter(payload *types.Payload) (*inventory.ResourceFilter, error) {
	req := &inventory.ResourceFilter{
		Resource: &inventory.Resource{
			Resource: &inventory.Resource_TelemetryGroup{},
		},
	}
	if payload.Data != nil {
		query, ok := payload.Data.(api.GetTelemetryGroupsLogsParams)
		if !ok {
			err := errors.Errorfc(codes.InvalidArgument,
				"GetTelemetryGroupsLogsParams incorrectly formatted: %T",
				payload.Data,
			)
			log.InfraErr(err).Msg("list operation")
			return nil, err
		}
		err := castTelemetryLogsGroupQueryList(&query, req)
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

func castTelemetryLogsGroupQueryList(
	query *api.GetTelemetryGroupsLogsParams,
	req *inventory.ResourceFilter,
) error {
	req.Filter = fmt.Sprintf("%s = %s", telemetryv1.TelemetryGroupResourceFieldKind,
		telemetryv1.TelemetryResourceKind_name[int32(telemetryv1.TelemetryResourceKind_TELEMETRY_RESOURCE_KIND_LOGS)])

	err := error(nil)
	req.Limit, req.Offset, err = parsePagination(
		query.PageSize,
		query.Offset,
	)
	if err != nil {
		return err
	}
	if query.OrderBy != nil {
		req.OrderBy = *query.OrderBy
	}
	return nil
}
