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
	inventory "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/inventory/v1"
	ouv1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/ou/v1"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/errors"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/validator"
)

// OpenAPIOuToProto maps OpenAPI fields name to Proto fields name.
// The key is derived from the json property respectively of the
// structs OuTemplate defined in edge-infrastructure-manager-openapi-types.gen.go.
var OpenAPIOuToProto = map[string]string{
	"ouKind":   ouv1.OuResourceFieldOuKind,
	"metadata": ouv1.OuResourceFieldMetadata,
	"name":     ouv1.OuResourceFieldName,
	"parentOu": ouv1.OuResourceEdgeParentOu,
}

// OpenAPIOUToProtoExcluded defines exclusion rules as there are some fields
// defined in the OpenAPI spec that are not currently mapped to the proto
// fields.
var OpenAPIOUToProtoExcluded = map[string]struct{}{
	"ouID":              {}, // ouID must not be set from the API
	"resourceId":        {}, // resourceId must not be set from the API
	"inheritedMetadata": {}, // inheritedMetadata must not be set from the API
	"timestamps":        {}, // read-only field
}

func NewOUHandler(invClient *clients.InventoryClientHandler) InventoryResource {
	return &ouHandler{invClient: invClient}
}

type ouHandler struct {
	invClient *clients.InventoryClientHandler
}

func (h *ouHandler) Create(job *types.Job) (*types.Payload, error) {
	body, err := castOUAPI(&job.Payload)
	if err != nil {
		return nil, err
	}

	ou, err := openapiToGrpcOU(body)
	if err != nil {
		return nil, err
	}

	req := &inventory.Resource{
		Resource: &inventory.Resource_Ou{
			Ou: ou,
		},
	}

	invResp, err := h.invClient.InvClient.Create(job.Context, req)
	if err != nil {
		return nil, err
	}

	createdOu := invResp.GetOu()
	obj := grpcToOpenAPIOU(createdOu, nil)

	return &types.Payload{Data: obj}, err
}

func (h *ouHandler) Get(job *types.Job) (*types.Payload, error) {
	req, err := ouResourceID(&job.Payload)
	if err != nil {
		return nil, err
	}

	invResp, err := h.invClient.InvClient.Get(job.Context, req)
	if err != nil {
		return nil, err
	}

	ou, meta, err := castToOU(invResp)
	if err != nil {
		return nil, err
	}

	obj := grpcToOpenAPIOU(ou, meta)
	return &types.Payload{Data: obj}, nil
}

func (h *ouHandler) Update(job *types.Job) (*types.Payload, error) {
	resID, err := ouResourceID(&job.Payload)
	if err != nil {
		return nil, err
	}

	fm, err := ouFieldMask(&job.Payload, job.Operation)
	if err != nil {
		return nil, err
	}

	res, err := ouResource(&job.Payload)
	if err != nil {
		return nil, err
	}

	invResp, err := h.invClient.InvClient.Update(job.Context, resID, fm, res)
	if err != nil {
		return nil, err
	}

	updatedOu := invResp.GetOu()
	obj := grpcToOpenAPIOU(updatedOu, nil)
	obj.OuID = &resID // to be removed
	obj.ResourceId = &resID
	return &types.Payload{Data: obj}, nil
}

func (h *ouHandler) Delete(job *types.Job) error {
	req, err := ouResourceID(&job.Payload)
	if err != nil {
		return err
	}

	_, err = h.invClient.InvClient.Delete(job.Context, req)
	if err != nil {
		return err
	}

	return nil
}

func (h *ouHandler) List(job *types.Job) (*types.Payload, error) {
	filter, err := ouFilter(&job.Payload)
	if err != nil {
		return nil, err
	}

	resp, err := h.invClient.InvClient.List(job.Context, filter)
	if err != nil {
		return nil, err
	}

	ous := make([]api.OU, 0, len(resp.GetResources()))

	for _, res := range resp.GetResources() {
		ou, meta, err := castToOU(res)
		if err != nil {
			return nil, err
		}
		obj := grpcToOpenAPIOU(ou, meta)
		ous = append(ous, *obj)
	}

	hasNext := resp.GetHasNext()
	totalElems := int(resp.GetTotalElements())
	ousList := api.OUsList{
		OUs:           &ous,
		HasNext:       &hasNext,
		TotalElements: &totalElems,
	}

	payload := &types.Payload{Data: ousList}
	return payload, nil
}

func castOUAPI(payload *types.Payload) (*api.OU, error) {
	body, ok := payload.Data.(*api.OU)
	if !ok {
		err := errors.Errorfc(codes.InvalidArgument,
			"body format is not OU: %T",
			payload.Data,
		)
		log.InfraErr(err).Msgf("")
		return nil, err
	}
	return body, nil
}

func ouResource(payload *types.Payload) (*inventory.Resource, error) {
	body, err := castOUAPI(payload)
	if err != nil {
		return nil, err
	}

	ou, err := openapiToGrpcOU(body)
	if err != nil {
		return nil, err
	}

	req := &inventory.Resource{
		Resource: &inventory.Resource_Ou{
			Ou: ou,
		},
	}
	return req, nil
}

func ouFilter(payload *types.Payload) (*inventory.ResourceFilter, error) {
	req := &inventory.ResourceFilter{
		Resource: &inventory.Resource{Resource: &inventory.Resource_Ou{Ou: &ouv1.OuResource{}}},
	}
	if payload.Data != nil {
		query, ok := payload.Data.(api.GetOusParams)
		if !ok {
			err := errors.Errorfc(codes.InvalidArgument,
				"GetOusParams incorrectly formatted: %T",
				payload.Data,
			)
			log.InfraErr(err).Msg("list operation")
			return nil, err
		}
		err := castOUQueryList(&query, req)
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

func ouResourceID(payload *types.Payload) (string, error) {
	params, ok := payload.Params.(OUURLParams)
	if !ok {
		err := errors.Errorfc(codes.InvalidArgument, "OUURLParams incorrectly formatted: %T",
			payload.Data)
		log.InfraErr(err).Msg("could not parse job payload params")
		return "", err
	}
	return params.OUID, nil
}

func ouFieldMask(payload *types.Payload, operation types.Operation) (*fieldmaskpb.FieldMask, error) {
	body, ok := payload.Data.(*api.OU)
	if !ok {
		err := errors.Errorfc(codes.InvalidArgument,
			"body format is not OU: %T",
			payload.Data,
		)
		log.InfraErr(err).Msgf("")
		return nil, err
	}

	ouRes, err := ouResource(payload)
	if err != nil {
		return nil, err
	}

	var fieldmask *fieldmaskpb.FieldMask
	if operation == types.Patch {
		fieldmask, err = getOuFieldmask(*body)
	} else {
		fieldmask, err = fieldmaskpb.New(ouRes.GetOu(), maps.Values(OpenAPIOuToProto)...)
	}
	if err != nil {
		log.InfraErr(err).Msgf("could not create fieldmask")
		return nil, errors.Wrap(err)
	}

	return fieldmask, nil
}

func castOUQueryList(
	query *api.GetOusParams,
	req *inventory.ResourceFilter,
) error {
	ou := &ouv1.OuResource{}

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
	} else if query.Parent != nil {
		if *query.Parent != emptyNullCase {
			req.Filter = fmt.Sprintf("%s.%s = %q", ouv1.OuResourceEdgeParentOu, ouv1.OuResourceFieldResourceId, *query.Parent)
		} else {
			req.Filter = fmt.Sprintf("NOT has(%s)", ouv1.OuResourceEdgeParentOu)
		}
	}
	req.Resource.Resource = &inventory.Resource_Ou{
		Ou: ou,
	}
	return nil
}

// helpers method to convert between API formats.
func castToOU(
	resp *inventory.GetResourceResponse,
) (*ouv1.OuResource, *inventory.GetResourceResponse_ResourceMetadata, error) {
	if resp.GetResource().GetOu() != nil {
		return resp.GetResource().GetOu(), resp.GetRenderedMetadata(), nil
	}
	err := errors.Errorfc(codes.Internal, "%s is not a OUResource", resp.GetResource())
	log.InfraErr(err).Msgf("could not cast inventory resource")
	return nil, nil, err
}

func getOuFieldmask(body api.OU) (*fieldmaskpb.FieldMask, error) {
	fieldList := getProtoFieldListFromOpenapiValue(body, OpenAPIOuToProto)
	log.Debug().Msgf("Proto Valid Fields: %s", fieldList)
	return fieldmaskpb.New(&ouv1.OuResource{}, fieldList...)
}

func openapiToGrpcOU(body *api.OU) (*ouv1.OuResource, error) {
	metadata, metaErr := marshalMetadata(body.Metadata)
	if metaErr != nil {
		log.Debug().Msgf("marshal OU metadata error: %s", metaErr.Error())
	}

	ou := &ouv1.OuResource{
		Name:     body.Name,
		Metadata: metadata,
	}

	if !isUnset(body.ParentOu) {
		parentOU := *body.ParentOu
		ou.ParentOu = &ouv1.OuResource{
			ResourceId: parentOU,
		}
	}
	if body.OuKind != nil {
		ou.OuKind = *body.OuKind
	}

	err := validator.ValidateMessage(ou)
	if err != nil {
		log.InfraSec().InfraErr(err).Msg("could not validate inventory resource")
		return nil, errors.Wrap(err)
	}

	return ou, nil
}

func grpcToOpenAPIOU(
	ou *ouv1.OuResource,
	meta *inventory.GetResourceResponse_ResourceMetadata,
) *api.OU {
	var parentOUID *string

	metadata, metaErr := unmarshalMetadata(ou.GetMetadata())
	if metaErr != nil {
		log.Debug().Msgf("unmarshal OU metadata error: %s", metaErr.Error())
	}

	parentOU := ou.GetParentOu()
	if parentOU != nil {
		parentOUID = getPtr(parentOU.GetResourceId())
	}

	ouKind := ou.GetOuKind()
	ouID := ou.GetResourceId()
	obj := api.OU{
		OuID:       &ouID,
		Name:       ou.GetName(),
		OuKind:     &ouKind,
		ParentOu:   parentOUID,
		Metadata:   metadata,
		ResourceId: &ouID,
		Timestamps: GrpcToOpenAPITimestamps(ou),
	}

	if meta != nil {
		obj.InheritedMetadata = &api.Metadata{}
		obj.InheritedMetadata, metaErr = unmarshalMetadata(meta.GetLogiMetadata())
		if metaErr != nil {
			log.Debug().Msgf("unmarshal OU rendered logical metadata error: %s", metaErr.Error())
		}
	}
	return &obj
}
