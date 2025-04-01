// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package invhandlers

import (
	"fmt"

	"golang.org/x/exp/maps"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	"github.com/open-edge-platform/infra-core/api/internal/types"
	"github.com/open-edge-platform/infra-core/api/internal/worker/clients"
	"github.com/open-edge-platform/infra-core/api/pkg/api/v0"
	computev1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/compute/v1"
	inventory "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/inventory/v1"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/errors"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/validator"
)

// OpenAPIWorkloadToProto maps OpenAPI fields name to Proto fields name.
// The key is derived from the json property respectively of the
// structs Workload defined in edge-infrastructure-manager-openapi-types.gen.go.
var OpenAPIWorkloadToProto = map[string]string{
	"kind":       computev1.WorkloadResourceFieldKind,
	"name":       computev1.WorkloadResourceFieldName,
	"status":     computev1.WorkloadResourceFieldStatus,
	"externalId": computev1.WorkloadResourceFieldExternalId,
}

// OpenAPIWorkloadToProtoExcluded defines exclusion rules as there are some fields
// defined in the OpenAPI spec that are not currently mapped to the proto
// fields especially when doing translation from OAPI to proto.
var OpenAPIWorkloadToProtoExcluded = map[string]struct{}{
	"workloadId": {}, // workloadId should never be used when translating from OAPI to proto.
	"members":    {}, // members should never be used when translating from OAPI to proto.
	"resourceId": {}, // resourceId should never be used when translating from OAPI to proto.
	"timestamps": {}, // read-only field
}

var (
	workloadOpenAPIWorkloadKindTogrpcKind = map[api.WorkloadKind]computev1.WorkloadKind{
		api.WORKLOADKINDUNSPECIFIED: computev1.WorkloadKind_WORKLOAD_KIND_UNSPECIFIED,
		api.WORKLOADKINDCLUSTER:     computev1.WorkloadKind_WORKLOAD_KIND_CLUSTER,
	}
	workloadGrpcKindToOpenAPIKind = map[computev1.WorkloadKind]api.WorkloadKind{
		computev1.WorkloadKind_WORKLOAD_KIND_UNSPECIFIED: api.WORKLOADKINDUNSPECIFIED,
		computev1.WorkloadKind_WORKLOAD_KIND_CLUSTER:     api.WORKLOADKINDCLUSTER,
	}
)

func NewWorkloadHandler(invClient *clients.InventoryClientHandler) InventoryResource {
	return &workloadHandler{invClient: invClient}
}

type workloadHandler struct {
	invClient *clients.InventoryClientHandler
}

func (w workloadHandler) List(job *types.Job) (*types.Payload, error) {
	filter, err := workloadFilter(&job.Payload)
	if err != nil {
		return nil, err
	}

	resp, err := w.invClient.InvClient.List(job.Context, filter)
	if err != nil {
		return nil, err
	}

	workloads := make([]api.Workload, 0, len(resp.GetResources()))
	for _, workloadResp := range resp.GetResources() {
		workload, err := castToWorkload(workloadResp)
		if err != nil {
			return nil, err
		}
		obj := grpcToOpenAPIWorkload(workload)
		workloads = append(workloads, *obj)
	}

	hasNext := resp.GetHasNext()
	totalElems := int(resp.GetTotalElements())
	workloadsList := api.WorkloadList{
		Workloads:     &workloads,
		HasNext:       &hasNext,
		TotalElements: &totalElems,
	}

	payload := &types.Payload{Data: workloadsList}
	return payload, nil
}

func (w workloadHandler) Create(job *types.Job) (*types.Payload, error) {
	body, err := castWorkloadAPI(&job.Payload)
	if err != nil {
		return nil, err
	}

	workload, err := openapiToGrpcWorkload(body)
	if err != nil {
		return nil, err
	}

	req := &inventory.Resource{
		Resource: &inventory.Resource_Workload{
			Workload: workload,
		},
	}

	invResp, err := w.invClient.InvClient.Create(job.Context, req)
	if err != nil {
		return nil, err
	}

	createdWLoad := invResp.GetWorkload()
	obj := grpcToOpenAPIWorkload(createdWLoad)

	return &types.Payload{Data: obj}, err
}

func (w workloadHandler) Get(job *types.Job) (*types.Payload, error) {
	req, err := workloadResourceID(&job.Payload)
	if err != nil {
		return nil, err
	}

	invResp, err := w.invClient.InvClient.Get(job.Context, req)
	if err != nil {
		return nil, err
	}

	workload, err := castToWorkload(invResp)
	if err != nil {
		return nil, err
	}

	obj := grpcToOpenAPIWorkload(workload)
	return &types.Payload{Data: obj}, nil
}

func (w workloadHandler) Update(job *types.Job) (*types.Payload, error) {
	resID, err := workloadResourceID(&job.Payload)
	if err != nil {
		return nil, err
	}

	oapiWorkload, err := castWorkloadAPI(&job.Payload)
	if err != nil {
		return nil, err
	}

	protoWorkload, err := openapiToGrpcWorkload(oapiWorkload)
	if err != nil {
		return nil, err
	}

	fm, err := workloadFieldMask(oapiWorkload, protoWorkload, job.Operation)
	if err != nil {
		return nil, err
	}

	req := &inventory.Resource{
		Resource: &inventory.Resource_Workload{
			Workload: protoWorkload,
		},
	}

	invResp, err := w.invClient.InvClient.Update(job.Context, resID, fm, req)
	if err != nil {
		return nil, err
	}

	updatedWorkload := invResp.GetWorkload()
	obj := grpcToOpenAPIWorkload(updatedWorkload)
	obj.WorkloadId = &resID // to be removed
	obj.ResourceId = &resID

	return &types.Payload{Data: obj}, nil
}

func (w workloadHandler) Delete(job *types.Job) error {
	resID, err := workloadResourceID(&job.Payload)
	if err != nil {
		return err
	}

	_, err = w.invClient.InvClient.Delete(job.Context, resID)
	if err != nil {
		return err
	}
	return nil
}

func castWorkloadAPI(payload *types.Payload) (*api.Workload, error) {
	body, ok := payload.Data.(*api.Workload)
	if !ok {
		err := errors.Errorfc(codes.InvalidArgument,
			"body format is not Workload: %T",
			payload.Data,
		)
		log.InfraErr(err).Msgf("")
		return nil, err
	}
	return body, nil
}

func openapiToGrpcWorkload(body *api.Workload) (*computev1.WorkloadResource, error) {
	// This is only used for POST/PUT/PATCH
	workload := &computev1.WorkloadResource{
		Kind: workloadOpenAPIWorkloadKindTogrpcKind[body.Kind],
		// We directly set the desired state since it's not being exposed to the outside yet
		DesiredState: computev1.WorkloadState_WORKLOAD_STATE_PROVISIONED,
	}
	if body.Name != nil {
		workload.Name = *body.Name
	}
	if body.Status != nil {
		workload.Status = *body.Status
	}
	if body.ExternalId != nil {
		workload.ExternalId = *body.ExternalId
	}

	err := validator.ValidateMessage(workload)
	if err != nil {
		log.InfraSec().InfraErr(err).Msg("could not validate inventory resource")
		return nil, errors.Wrap(err)
	}

	return workload, nil
}

func getWorkloadFieldmask(body *api.Workload) (*fieldmaskpb.FieldMask, error) {
	var fieldList []string
	fieldList = append(
		fieldList,
		getProtoFieldListFromOpenapiPointer(body, OpenAPIWorkloadToProto)...)
	log.Debug().Msgf("Proto Valid Fields: %s", fieldList)
	return fieldmaskpb.New(&computev1.WorkloadResource{}, fieldList...)
}

func workloadResourceID(payload *types.Payload) (string, error) {
	params, ok := payload.Params.(WorkloadURLParams)
	if !ok {
		err := errors.Errorfc(codes.InvalidArgument, "WorkloadURLParams incorrectly formatted: %T",
			payload.Data)
		log.InfraErr(err).Msg("could not parse job payload params")
		return "", err
	}
	return params.WorkloadID, nil
}

// helpers method to convert between API formats.
func castToWorkload(resp *inventory.GetResourceResponse) (
	*computev1.WorkloadResource, error,
) {
	if resp.GetResource().GetWorkload() != nil {
		return resp.GetResource().GetWorkload(), nil
	}
	err := errors.Errorfc(codes.Internal, "%s is not a WorkloadResource", resp.GetResource())
	log.InfraErr(err).Msgf("could not cast inventory resource")
	return nil, err
}

func grpcToOpenAPIWorkload(workload *computev1.WorkloadResource) *api.Workload {
	wKind := workloadGrpcKindToOpenAPIKind[workload.GetKind()]
	wName := workload.GetName()
	wResourceID := workload.GetResourceId()
	workloadMembers := workload.GetMembers()
	externalID := workload.GetExternalId()

	obj := api.Workload{
		WorkloadId: &wResourceID,
		Kind:       wKind,
		Name:       &wName,
		Status:     &workload.Status,
		ExternalId: &externalID,
		ResourceId: &wResourceID,
		Timestamps: GrpcToOpenAPITimestamps(workload),
	}
	if workloadMembers != nil {
		var members []api.WorkloadMember
		for _, m := range workloadMembers {
			members = append(members, *grpcToOpenAPIWorkloadMember(m))
		}
		obj.Members = &members
	}

	return &obj
}

func workloadFieldMask(
	oapiWorkload *api.Workload,
	protoWorkload *computev1.WorkloadResource,
	operation types.Operation,
) (*fieldmaskpb.FieldMask, error) {
	var fieldmask *fieldmaskpb.FieldMask
	err := error(nil)
	if operation == types.Patch {
		fieldmask, err = getWorkloadFieldmask(oapiWorkload)
	} else {
		fieldmask, err = fieldmaskpb.New(protoWorkload, maps.Values(OpenAPIWorkloadToProto)...)
	}
	if err != nil {
		log.InfraErr(err).Msgf("could not create fieldmask")
		return nil, errors.Wrap(err)
	}

	return fieldmask, nil
}

func workloadFilter(payload *types.Payload) (*inventory.ResourceFilter, error) {
	req := &inventory.ResourceFilter{
		Resource: &inventory.Resource{Resource: &inventory.Resource_Workload{Workload: &computev1.WorkloadResource{}}},
	}
	if payload.Data != nil {
		query, ok := payload.Data.(api.GetWorkloadsParams)
		if !ok {
			err := errors.Errorfc(codes.InvalidArgument,
				"GetWorkloadsParams incorrectly formatted: %T",
				payload.Data,
			)
			log.InfraErr(err).Msg("list operation")
			return nil, err
		}
		err := castWorkloadQueryList(&query, req)
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

func castWorkloadQueryList(
	query *api.GetWorkloadsParams,
	req *inventory.ResourceFilter,
) error {
	workload := &computev1.WorkloadResource{}

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

	if query.Filter != nil {
		req.Filter = *query.Filter
	} else if query.Kind != nil {
		req.Filter = fmt.Sprintf("%s = %s", computev1.WorkloadResourceFieldKind,
			workloadOpenAPIWorkloadKindTogrpcKind[*query.Kind])
	}
	req.Resource.Resource = &inventory.Resource_Workload{
		Workload: workload,
	}
	return nil
}
