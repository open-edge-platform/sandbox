// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package invhandlers

import (
	"fmt"

	"google.golang.org/grpc/codes"

	"github.com/open-edge-platform/infra-core/api/internal/types"
	"github.com/open-edge-platform/infra-core/api/internal/worker/clients"
	"github.com/open-edge-platform/infra-core/api/pkg/api/v0"
	computev1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/compute/v1"
	inventory "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/inventory/v1"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/errors"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/validator"
)

// OpenAPIWorkloadMemberToProto maps OpenAPI fields name to Proto fields name.
// The key is derived from the json property respectively of the
// structs WorkloadMember defined in edge-infrastructure-manager-openapi-types.gen.go.
var OpenAPIWorkloadMemberToProto = map[string]string{
	"kind":       computev1.WorkloadMemberFieldKind,
	"instanceId": computev1.WorkloadMemberEdgeInstance, // instance_id is carried via the HostResource.ResourceID
	"workloadId": computev1.WorkloadMemberEdgeWorkload, // workload_id is carried via the Workload.ResourceID
}

// OpenAPIWorkloadMemberToProtoExcluded defines exclusion rules as there are some fields
// defined in the OpenAPI spec that are not currently mapped to the proto
// fields especially when doing translation from OAPI to proto.
var OpenAPIWorkloadMemberToProtoExcluded = map[string]struct{}{
	"member":           {}, // member should never be used when translating from OAPI to proto.
	"workloadMemberId": {}, // workloadMemberId should never be used when translating from OAPI to proto.
	"resourceId":       {}, // resourceId should never be used when translating from OAPI to proto.
	"instance":         {}, // instance should never be used when translating from OAPI to proto.
	"workload":         {}, // workload should never be used when translating from OAPI to proto.
	"timestamps":       {}, // read-only field
}

var (
	workloadMemberOpenAPIWorkloadMemberKindTogrpcKind = map[api.WorkloadMemberKind]computev1.WorkloadMemberKind{
		api.WORKLOADMEMBERKINDUNSPECIFIED: computev1.WorkloadMemberKind_WORKLOAD_MEMBER_KIND_UNSPECIFIED,
		api.WORKLOADMEMBERKINDCLUSTERNODE: computev1.WorkloadMemberKind_WORKLOAD_MEMBER_KIND_CLUSTER_NODE,
	}
	workloadMemberGrpcKindToOpenAPIKind = map[computev1.WorkloadMemberKind]api.WorkloadMemberKind{
		computev1.WorkloadMemberKind_WORKLOAD_MEMBER_KIND_UNSPECIFIED:  api.WORKLOADMEMBERKINDUNSPECIFIED,
		computev1.WorkloadMemberKind_WORKLOAD_MEMBER_KIND_CLUSTER_NODE: api.WORKLOADMEMBERKINDCLUSTERNODE,
	}
)

func NewWorkloadMemberHandler(invClient *clients.InventoryClientHandler) InventoryResource {
	return &workloadMemberHandler{invClient: invClient}
}

type workloadMemberHandler struct {
	invClient *clients.InventoryClientHandler
}

func (w workloadMemberHandler) List(job *types.Job) (*types.Payload, error) {
	filter, err := workloadMemberFilter(&job.Payload)
	if err != nil {
		return nil, err
	}

	resp, err := w.invClient.InvClient.List(job.Context, filter)
	if err != nil {
		return nil, err
	}

	members := make([]api.WorkloadMember, 0, len(resp.GetResources()))
	for _, workloadResp := range resp.GetResources() {
		workload, err := castToWorkloadMember(workloadResp)
		if err != nil {
			return nil, err
		}
		obj := grpcToOpenAPIWorkloadMember(workload)
		members = append(members, *obj)
	}

	hasNext := resp.GetHasNext()
	totalElems := int(resp.GetTotalElements())
	memberList := api.WorkloadMemberList{
		WorkloadMembers: &members,
		HasNext:         &hasNext,
		TotalElements:   &totalElems,
	}

	payload := &types.Payload{Data: memberList}
	return payload, nil
}

func (w workloadMemberHandler) Create(job *types.Job) (*types.Payload, error) {
	body, err := castWorkloadMemberAPI(&job.Payload)
	if err != nil {
		return nil, err
	}

	member := openapiToGrpcWorkloadMember(body)

	req := &inventory.Resource{
		Resource: &inventory.Resource_WorkloadMember{
			WorkloadMember: member,
		},
	}

	invResp, err := w.invClient.InvClient.Create(job.Context, req)
	if err != nil {
		return nil, err
	}

	createdWLoadMember := invResp.GetWorkloadMember()
	obj := grpcToOpenAPIWorkloadMember(createdWLoadMember)

	return &types.Payload{Data: obj}, err
}

func (w workloadMemberHandler) Get(job *types.Job) (*types.Payload, error) {
	memberID, err := workloadMemberResourceID(&job.Payload)
	if err != nil {
		return nil, err
	}

	invResp, err := w.invClient.InvClient.Get(job.Context, memberID)
	if err != nil {
		return nil, err
	}

	workload, err := castToWorkloadMember(invResp)
	if err != nil {
		return nil, err
	}

	obj := grpcToOpenAPIWorkloadMember(workload)
	return &types.Payload{Data: obj}, nil
}

func (w workloadMemberHandler) Update(_ *types.Job) (*types.Payload, error) {
	// Unsupported, we should never reach this point
	err := errors.Errorfc(codes.Unimplemented, "you cannot update a workload member, you can delete and create "+
		"a workload member instead")
	log.InfraSec().InfraErr(err).Msg("PATCH and PUT are unsupported operation for workload member")
	return nil, err
}

func (w workloadMemberHandler) Delete(job *types.Job) error {
	memberID, err := workloadMemberResourceID(&job.Payload)
	if err != nil {
		return err
	}

	_, err = w.invClient.InvClient.Delete(job.Context, memberID)
	if err != nil {
		return err
	}
	return nil
}

// helpers method to convert between API formats.
func castToWorkloadMember(resp *inventory.GetResourceResponse) (
	*computev1.WorkloadMember, error,
) {
	if resp.GetResource().GetWorkloadMember() != nil {
		return resp.GetResource().GetWorkloadMember(), nil
	}
	err := errors.Errorfc(codes.Internal, "%s is not a WorkloadMemberResource", resp.GetResource())
	log.InfraErr(err).Msgf("could not cast inventory resource")
	return nil, err
}

func workloadMemberFilter(payload *types.Payload) (*inventory.ResourceFilter, error) {
	req := &inventory.ResourceFilter{
		Resource: &inventory.Resource{
			Resource: &inventory.Resource_WorkloadMember{
				WorkloadMember: &computev1.WorkloadMember{},
			},
		},
	}
	if payload.Data != nil {
		query, ok := payload.Data.(api.GetWorkloadMembersParams)
		if !ok {
			err := errors.Errorfc(codes.InvalidArgument,
				"GetWorkloadMembersParams incorrectly formatted: %T",
				payload.Data,
			)
			log.InfraErr(err).Msg("list operation")
			return nil, err
		}
		err := castWorkloadMemberQueryList(&query, req)
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

func castWorkloadMemberQueryList(
	query *api.GetWorkloadMembersParams,
	req *inventory.ResourceFilter,
) error {
	workloadMember := &computev1.WorkloadMember{}

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
	} else if query.WorkloadId != nil {
		req.Filter = fmt.Sprintf("%s.%s = %s", computev1.WorkloadMemberEdgeWorkload,
			computev1.WorkloadResourceFieldResourceId, *query.WorkloadId)
	}
	req.Resource.Resource = &inventory.Resource_WorkloadMember{
		WorkloadMember: workloadMember,
	}
	return nil
}

func castWorkloadMemberAPI(payload *types.Payload) (*api.WorkloadMember, error) {
	body, ok := payload.Data.(*api.WorkloadMember)
	if !ok {
		err := errors.Errorfc(codes.InvalidArgument,
			"body format is not WorkloadMember: %T",
			payload.Data,
		)
		log.InfraErr(err).Msgf("")
		return nil, err
	}
	return body, nil
}

func openapiToGrpcWorkloadMember(body *api.WorkloadMember) *computev1.WorkloadMember {
	// This is only used for POST
	wMember := &computev1.WorkloadMember{
		Kind: workloadMemberOpenAPIWorkloadMemberKindTogrpcKind[body.Kind],
		Workload: &computev1.WorkloadResource{
			ResourceId: *body.WorkloadId,
		},
	}
	if body.InstanceId != nil {
		wMember.Instance = &computev1.InstanceResource{
			ResourceId: *body.InstanceId,
		}
	}
	return wMember
}

func grpcToOpenAPIWorkloadMember(member *computev1.WorkloadMember) *api.WorkloadMember {
	mKind := workloadMemberGrpcKindToOpenAPIKind[member.GetKind()]
	mResourceID := member.GetResourceId()
	obj := api.WorkloadMember{
		Kind:             mKind,
		WorkloadMemberId: &mResourceID,
		ResourceId:       &mResourceID,
		Timestamps:       GrpcToOpenAPITimestamps(member),
	}
	if w := member.GetWorkload(); w != nil {
		obj.Workload = grpcToOpenAPIWorkload(w)
	}
	if h := member.GetInstance(); h != nil {
		obj.Member = GrpcToOpenAPIInstance(h)
		obj.Instance = GrpcToOpenAPIInstance(h)
	}
	return &obj
}

func workloadMemberResourceID(payload *types.Payload) (string, error) {
	params, ok := payload.Params.(WorkloadMemberURLParams)
	if !ok {
		err := errors.Errorfc(codes.InvalidArgument, "WorkloadMemberURLParams incorrectly formatted: %T",
			payload.Data)
		log.InfraErr(err).Msg("could not parse job payload params")
		return "", err
	}
	return params.WorkloadMemberID, nil
}
