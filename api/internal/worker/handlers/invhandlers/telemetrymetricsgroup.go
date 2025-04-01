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

// OpenAPITelemetryMetricsGroupResourceToProto maps OpenAPI fields name to Proto fields name.
// The key is derived from the json property respectively of the
// structs TelemetryGroupResource defined in edge-infrastructure-manager-openapi-types.gen.go.
var OpenAPITelemetryMetricsGroupResourceToProto = map[string]string{
	"name":          telemetryv1.TelemetryGroupResourceFieldName,
	"collectorKind": telemetryv1.TelemetryGroupResourceFieldCollectorKind,
}

// OpenAPITelemetryMetricsGroupToProtoExcluded defines exclusion rules as there are some fields
// defined in the OpenAPI spec that are not currently mapped to the proto
// fields especially when doing translation from OAPI to proto.
var OpenAPITelemetryMetricsGroupToProtoExcluded = map[string]struct{}{
	"telemetryMetricsGroupId": {}, // telemetryMetricsGroupId should never be used when translating from OAPI to proto.
	"groups":                  {}, // groups should never be used when translating from OAPI to proto.
	"timestamps":              {}, // read-only field
}

type telemetryMetricsGroupHandler struct {
	invClient *clients.InventoryClientHandler
}

func NewTelemetryMetricsGroupHandler(invClient *clients.InventoryClientHandler) InventoryResource {
	return &telemetryMetricsGroupHandler{invClient: invClient}
}

func (t telemetryMetricsGroupHandler) Create(job *types.Job) (*types.Payload, error) {
	body, err := castTelemetryMetricsGroupAPI(&job.Payload)
	if err != nil {
		return nil, err
	}

	metricsGroup, err := openapiToGrpcTelemetryMetricsGroup(body)
	if err != nil {
		return nil, err
	}

	req := &inventory.Resource{
		Resource: &inventory.Resource_TelemetryGroup{
			TelemetryGroup: metricsGroup,
		},
	}

	invResp, err := t.invClient.InvClient.Create(job.Context, req)
	if err != nil {
		return nil, err
	}

	createdTGMetrics := invResp.GetTelemetryGroup()
	obj := grpcToOpenAPITelemetryMetricsGroup(createdTGMetrics)

	return &types.Payload{Data: obj}, err
}

func (t telemetryMetricsGroupHandler) Get(job *types.Job) (*types.Payload, error) {
	req, err := telemetryMetricsGroupID(&job.Payload)
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

	obj := grpcToOpenAPITelemetryMetricsGroup(telemetryGroup)

	return &types.Payload{Data: obj}, nil
}

func (t telemetryMetricsGroupHandler) Update(job *types.Job) (*types.Payload, error) {
	err := errors.Errorfc(codes.Unimplemented, "%s operation not supported", job.Operation)
	return nil, err
}

func (t telemetryMetricsGroupHandler) Delete(job *types.Job) error {
	req, err := telemetryMetricsGroupID(&job.Payload)
	if err != nil {
		return err
	}

	_, err = t.invClient.InvClient.Delete(job.Context, req)
	if err != nil {
		return err
	}

	return nil
}

func (t telemetryMetricsGroupHandler) List(job *types.Job) (*types.Payload, error) {
	filter, err := telemetryMetricsGroupFilter(&job.Payload)
	if err != nil {
		return nil, err
	}

	resp, err := t.invClient.InvClient.List(job.Context, filter)
	if err != nil {
		return nil, err
	}

	metricsGroups := make([]api.TelemetryMetricsGroup, 0, len(resp.GetResources())) // pre-allocate proper length
	for _, res := range resp.GetResources() {
		group, err := castToTelemetryGroupResource(res)
		if err != nil {
			return nil, err
		}
		obj := grpcToOpenAPITelemetryMetricsGroup(group)
		metricsGroups = append(metricsGroups, *obj)
	}

	hasNext := resp.GetHasNext()
	totalElems := int(resp.GetTotalElements())
	telemetryMetricsGroupList := api.TelemetryMetricsGroupList{
		TelemetryMetricsGroups: &metricsGroups,
		HasNext:                &hasNext,
		TotalElements:          &totalElems,
	}

	payload := &types.Payload{Data: telemetryMetricsGroupList}
	return payload, nil
}

func telemetryMetricsGroupID(payload *types.Payload) (string, error) {
	params, ok := payload.Params.(TelemetryMetricsGroupURLParams)
	if !ok {
		err := errors.Errorfc(codes.InvalidArgument, "TelemetryMetricsGroupURLParams incorrectly formatted: %T",
			payload.Data)
		log.InfraErr(err).Msg("could not parse job payload params")
		return "", err
	}
	return params.TelemetryMetricsGroupID, nil
}

func openapiToGrpcTelemetryMetricsGroup(body *api.TelemetryMetricsGroup) (*telemetryv1.TelemetryGroupResource, error) {
	telemetry := &telemetryv1.TelemetryGroupResource{
		Name:          body.Name,
		Groups:        body.Groups,
		Kind:          telemetryv1.TelemetryResourceKind_TELEMETRY_RESOURCE_KIND_METRICS,
		CollectorKind: telemetryCollectorKindAPItoGRPC(body.CollectorKind),
	}

	err := validator.ValidateMessage(telemetry)
	if err != nil {
		log.InfraErr(err).Msg("could not validate inventory resource")
		return nil, errors.Wrap(err)
	}

	return telemetry, nil
}

func grpcToOpenAPITelemetryMetricsGroup(
	telemetryGroup *telemetryv1.TelemetryGroupResource,
) *api.TelemetryMetricsGroup {
	resID := telemetryGroup.GetResourceId()
	resName := telemetryGroup.GetName()
	resGroups := telemetryGroup.GetGroups()

	obj := api.TelemetryMetricsGroup{
		TelemetryMetricsGroupId: &resID,
		Groups:                  resGroups,
		Name:                    resName,
		CollectorKind:           telemetryCollectorKindGRPCtoAPI(telemetryGroup.GetCollectorKind()),
		Timestamps:              GrpcToOpenAPITimestamps(telemetryGroup),
	}

	return &obj
}

func castTelemetryMetricsGroupAPI(payload *types.Payload) (*api.TelemetryMetricsGroup, error) {
	body, ok := payload.Data.(*api.TelemetryMetricsGroup)
	if !ok {
		err := errors.Errorfc(codes.InvalidArgument,
			"body format is not TelemetryMetricsGroup: %T",
			payload.Data,
		)
		log.InfraErr(err).Msgf("")
		return nil, err
	}
	return body, nil
}

func telemetryMetricsGroupFilter(payload *types.Payload) (*inventory.ResourceFilter, error) {
	req := &inventory.ResourceFilter{
		Resource: &inventory.Resource{
			Resource: &inventory.Resource_TelemetryGroup{
				TelemetryGroup: &telemetryv1.TelemetryGroupResource{},
			},
		},
	}
	if payload.Data != nil {
		query, ok := payload.Data.(api.GetTelemetryGroupsMetricsParams)
		if !ok {
			err := errors.Errorfc(codes.InvalidArgument,
				"GetTelemetryGroupsMetricsParams incorrectly formatted: %T",
				payload.Data,
			)
			log.InfraErr(err).Msg("list operation")
			return nil, err
		}
		err := castTelemetryMetricsGroupQueryList(&query, req)
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

func castTelemetryMetricsGroupQueryList(
	query *api.GetTelemetryGroupsMetricsParams,
	req *inventory.ResourceFilter,
) error {
	req.Filter = fmt.Sprintf("%s = %s", telemetryv1.TelemetryGroupResourceFieldKind,
		telemetryv1.TelemetryResourceKind_name[int32(telemetryv1.TelemetryResourceKind_TELEMETRY_RESOURCE_KIND_METRICS)])

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
