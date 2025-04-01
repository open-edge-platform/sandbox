// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package invhandlers

import (
	"google.golang.org/grpc/codes"

	"github.com/open-edge-platform/infra-core/api/internal/types"
	"github.com/open-edge-platform/infra-core/api/internal/worker/clients"
	"github.com/open-edge-platform/infra-core/api/pkg/api/v0"
	inventory "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/inventory/v1"
	providerv1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/provider/v1"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/errors"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/validator"
)

// OpenAPIProviderToProto maps OpenAPI fields name to Proto fields name.
// The key is derived from the json property respectively of the
// structs Provider defined in edge-infrastructure-manager-openapi-types.gen.go.
var OpenAPIProviderToProto = map[string]string{
	"providerKind":   providerv1.ProviderResourceFieldProviderKind,
	"providerVendor": providerv1.ProviderResourceFieldProviderVendor,
	"name":           providerv1.ProviderResourceFieldName,
	"apiEndpoint":    providerv1.ProviderResourceFieldApiEndpoint,
	"apiCredentials": providerv1.ProviderResourceFieldApiCredentials,
	"config":         providerv1.ProviderResourceFieldConfig,
}

// OpenAPIProviderToProtoExcluded defines exclusion rules as there are some fields
// defined in the OpenAPI spec that are not currently mapped to the proto
// fields.
var OpenAPIProviderToProtoExcluded = map[string]struct{}{
	"providerID": {}, // providerID must not be set from the API
	"resourceId": {}, // resourceId must not be set from the API
	"timestamps": {}, // read-only field
}

func NewProvider(invClient *clients.InventoryClientHandler) InventoryResource {
	return &providerHandler{invClient: invClient}
}

type providerHandler struct {
	invClient *clients.InventoryClientHandler
}

func (h *providerHandler) Create(job *types.Job) (*types.Payload, error) {
	body, err := castProviderAPI(&job.Payload)
	if err != nil {
		return nil, err
	}

	provider, err := openapiProviderToGrpcProvider(body)
	if err != nil {
		return nil, err
	}

	req := &inventory.Resource{
		Resource: &inventory.Resource_Provider{
			Provider: provider,
		},
	}

	invResp, err := h.invClient.InvClient.Create(job.Context, req)
	if err != nil {
		return nil, err
	}

	createdProvider := invResp.GetProvider()
	obj := GrpcProviderToOpenAPIProvider(createdProvider)

	return &types.Payload{Data: obj}, err
}

func (h *providerHandler) Get(job *types.Job) (*types.Payload, error) {
	req, err := providerID(&job.Payload)
	if err != nil {
		return nil, err
	}

	invResp, err := h.invClient.InvClient.Get(job.Context, req)
	if err != nil {
		return nil, err
	}

	provider, err := castToProvider(invResp)
	if err != nil {
		return nil, err
	}

	obj := GrpcProviderToOpenAPIProvider(provider)

	return &types.Payload{Data: obj}, nil
}

func (h *providerHandler) Update(_ *types.Job) (*types.Payload, error) {
	// Unsupported, we should never reach this point
	err := errors.Errorfc(codes.Unimplemented, "you cannot update a provider, you can delete and create "+
		"a provider, if there are no dependants, instead")
	log.InfraSec().InfraErr(err).Msg("PATCH and PUT are unsupported operation for provider")
	return nil, err
}

func (h *providerHandler) Delete(job *types.Job) error {
	req, err := providerID(&job.Payload)
	if err != nil {
		return err
	}

	_, err = h.invClient.InvClient.Delete(job.Context, req)
	if err != nil {
		return err
	}

	return nil
}

func (h *providerHandler) List(job *types.Job) (*types.Payload, error) {
	filter, err := providerFilter(&job.Payload)
	if err != nil {
		return nil, err
	}

	resp, err := h.invClient.InvClient.List(job.Context, filter)
	if err != nil {
		return nil, err
	}

	providers := make([]api.Provider, 0, len(resp.GetResources()))

	for _, res := range resp.GetResources() {
		provider, err := castToProvider(res)
		if err != nil {
			return nil, err
		}
		obj := GrpcProviderToOpenAPIProvider(provider)
		providers = append(providers, *obj)
	}

	hasNext := resp.GetHasNext()
	totalElems := int(resp.GetTotalElements())
	providerList := api.ProviderList{
		Providers:     &providers,
		HasNext:       &hasNext,
		TotalElements: &totalElems,
	}

	payload := &types.Payload{Data: providerList}
	return payload, nil
}

func castProviderAPI(payload *types.Payload) (*api.Provider, error) {
	body, ok := payload.Data.(*api.Provider)
	if !ok {
		err := errors.Errorfc(codes.InvalidArgument,
			"body format is not Provider: %T",
			payload.Data,
		)
		log.InfraErr(err).Msgf("")
		return nil, err
	}
	return body, nil
}

func providerFilter(payload *types.Payload) (*inventory.ResourceFilter, error) {
	req := &inventory.ResourceFilter{
		Resource: &inventory.Resource{Resource: &inventory.Resource_Provider{Provider: &providerv1.ProviderResource{}}},
	}
	if payload.Data != nil {
		query, ok := payload.Data.(api.GetProvidersParams)
		if !ok {
			err := errors.Errorfc(codes.InvalidArgument,
				"GetProvidersParams incorrectly formatted: %T",
				payload.Data,
			)
			log.InfraErr(err).Msg("list operation")
			return nil, err
		}
		err := castProviderQueryList(&query, req)
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

func providerID(payload *types.Payload) (string, error) {
	params, ok := payload.Params.(ProviderURLParams)
	if !ok {
		err := errors.Errorfc(codes.InvalidArgument, "ProviderURLParams incorrectly formatted: %T",
			payload.Data)
		log.InfraErr(err).Msg("could not parse job payload params")
		return "", err
	}
	return params.ProviderID, nil
}

func castProviderQueryList(
	query *api.GetProvidersParams,
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
	if query.OrderBy != nil {
		req.OrderBy = *query.OrderBy
	}
	if query.Filter != nil {
		req.Filter = *query.Filter
	}
	return nil
}

// helpers method to convert between API formats.
func castToProvider(resp *inventory.GetResourceResponse) (
	*providerv1.ProviderResource, error,
) {
	if resp.GetResource().GetProvider() != nil {
		return resp.GetResource().GetProvider(), nil
	}
	err := errors.Errorfc(codes.Internal, "%s is not a ProviderResource", resp.GetResource())
	log.InfraErr(err).Msgf("could not cast inventory resource")
	return nil, err
}

func GrpcProviderKindToOpenAPIProviderKind(pk providerv1.ProviderKind) api.ProviderKind {
	if pk == providerv1.ProviderKind_PROVIDER_KIND_BAREMETAL {
		return api.PROVIDERKINDBAREMETAL
	}
	return api.PROVIDERKINDUNSPECIFIED
}

func openAPIProviderKindTogrpcProviderKind(pk api.ProviderKind) providerv1.ProviderKind {
	if pk == api.PROVIDERKINDBAREMETAL {
		return providerv1.ProviderKind_PROVIDER_KIND_BAREMETAL
	}
	return providerv1.ProviderKind_PROVIDER_KIND_UNSPECIFIED
}

func GrpcProviderVendorToOpenAPIProviderVendor(pv providerv1.ProviderVendor) api.ProviderVendor {
	switch pv {
	case providerv1.ProviderVendor_PROVIDER_VENDOR_LENOVO_LXCA:
		return api.PROVIDERVENDORLENOVOLXCA
	case providerv1.ProviderVendor_PROVIDER_VENDOR_LENOVO_LOCA:
		return api.PROVIDERVENDORLENOVOLOCA
	default:
		return api.PROVIDERVENDORUNSPECIFIED
	}
}

func openAPIProviderVendorTogrpcProviderVendor(pv *api.ProviderVendor) providerv1.ProviderVendor {
	if pv != nil {
		switch *pv {
		case api.PROVIDERVENDORLENOVOLXCA:
			return providerv1.ProviderVendor_PROVIDER_VENDOR_LENOVO_LXCA
		case api.PROVIDERVENDORLENOVOLOCA:
			return providerv1.ProviderVendor_PROVIDER_VENDOR_LENOVO_LOCA
		default:
			return providerv1.ProviderVendor_PROVIDER_VENDOR_UNSPECIFIED
		}
	}
	return providerv1.ProviderVendor_PROVIDER_VENDOR_UNSPECIFIED
}

func openapiProviderToGrpcProvider(body *api.Provider) (*providerv1.ProviderResource, error) {
	provider := &providerv1.ProviderResource{}

	provider.ProviderKind = openAPIProviderKindTogrpcProviderKind(body.ProviderKind)
	provider.ProviderVendor = openAPIProviderVendorTogrpcProviderVendor(body.ProviderVendor)
	provider.Name = body.Name
	provider.ApiEndpoint = body.ApiEndpoint
	if body.ApiCredentials != nil {
		provider.ApiCredentials = append(provider.ApiCredentials, *body.ApiCredentials...)
	}
	if body.Config != nil {
		provider.Config = *body.Config
	}

	err := validator.ValidateMessage(provider)
	if err != nil {
		log.InfraSec().InfraErr(err).Msg("could not validate inventory resource")
		return nil, errors.Wrap(err)
	}

	return provider, nil
}

func GrpcProviderToOpenAPIProvider(provider *providerv1.ProviderResource) *api.Provider {
	resID := provider.GetResourceId()
	providerKind := GrpcProviderKindToOpenAPIProviderKind(provider.GetProviderKind())
	providerVendor := GrpcProviderVendorToOpenAPIProviderVendor(provider.GetProviderVendor())
	resName := provider.GetName()
	apiEndpoint := provider.GetApiEndpoint()
	apiCredentials := provider.GetApiCredentials()
	config := provider.GetConfig()

	obj := api.Provider{
		ApiEndpoint:    apiEndpoint,
		ApiCredentials: &apiCredentials,
		Name:           resName,
		ProviderID:     &resID,
		ProviderKind:   providerKind,
		ProviderVendor: &providerVendor,
		Config:         &config,
		ResourceId:     &resID,
		Timestamps:     GrpcToOpenAPITimestamps(provider),
	}

	return &obj
}
