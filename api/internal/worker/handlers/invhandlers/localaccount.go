// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package invhandlers

import (
	"google.golang.org/grpc/codes"

	"github.com/open-edge-platform/infra-core/api/internal/types"
	"github.com/open-edge-platform/infra-core/api/internal/worker/clients"
	"github.com/open-edge-platform/infra-core/api/pkg/api/v0"
	inventory "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/inventory/v1"
	localaccountv1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/localaccount/v1"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/errors"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/validator"
)

// OpenAPILocalAccountToProto maps OpenAPI fields name to Proto fields name.
// The key is derived from the json property respectively of the
// structs LocalAccount defined in edge-infrastructure-manager-openapi-types.gen.go.
var OpenAPILocalAccountToProto = map[string]string{
	"username": localaccountv1.LocalAccountResourceFieldUsername,
	"sshKey":   localaccountv1.LocalAccountResourceFieldSshKey,
}

// OpenAPILocalAccountToProtoExcluded defines exclusion rules as there are some fields
// defined in the OpenAPI spec that are not currently mapped to the proto
// fields.
var OpenAPILocalAccountToProtoExcluded = map[string]struct{}{
	"localAccountID": {}, // localAccountID must not be set from the API
	"resourceId":     {}, // resourceId must not be set from the API
	"timestamps":     {}, // read-only field
}

func NewLocalAccountHandler(invClient *clients.InventoryClientHandler) InventoryResource {
	return &localAccountHandler{invClient: invClient}
}

type localAccountHandler struct {
	invClient *clients.InventoryClientHandler
}

func (h *localAccountHandler) Create(job *types.Job) (*types.Payload, error) {
	body, err := castLocalAccountAPI(&job.Payload)
	if err != nil {
		return nil, err
	}

	localaccount, err := openapiLocalAccountToGrpcLocalAccount(body)
	if err != nil {
		return nil, err
	}

	req := &inventory.Resource{
		Resource: &inventory.Resource_LocalAccount{
			LocalAccount: localaccount,
		},
	}

	invResp, err := h.invClient.InvClient.Create(job.Context, req)
	if err != nil {
		return nil, err
	}

	createdLocalAccount := invResp.GetLocalAccount()
	obj := GrpcLocalAccountToOpenAPIcreatedLocalAccount(createdLocalAccount)

	return &types.Payload{Data: obj}, err
}

func (h *localAccountHandler) Get(job *types.Job) (*types.Payload, error) {
	req, err := localAccountID(&job.Payload)
	if err != nil {
		return nil, err
	}

	invResp, err := h.invClient.InvClient.Get(job.Context, req)
	if err != nil {
		return nil, err
	}

	localAccount, err := castToLocalAccount(invResp)
	if err != nil {
		return nil, err
	}

	obj := GrpcLocalAccountToOpenAPIcreatedLocalAccount(localAccount)

	return &types.Payload{Data: obj}, nil
}

func (h *localAccountHandler) Delete(job *types.Job) error {
	req, err := localAccountID(&job.Payload)
	if err != nil {
		return err
	}
	_, err = h.invClient.InvClient.Delete(job.Context, req)
	if err != nil {
		return err
	}

	return nil
}

func (h *localAccountHandler) List(job *types.Job) (*types.Payload, error) {
	filter, err := localAccountFilter(&job.Payload)
	if err != nil {
		return nil, err
	}

	resp, err := h.invClient.InvClient.List(job.Context, filter)
	if err != nil {
		return nil, err
	}

	localAccounts := make([]api.LocalAccount, 0, len(resp.GetResources()))

	for _, res := range resp.GetResources() {
		localAccount, err := castToLocalAccount(res)
		if err != nil {
			return nil, err
		}
		obj := GrpcLocalAccountToOpenAPIcreatedLocalAccount(localAccount)
		localAccounts = append(localAccounts, *obj)
	}

	hasNext := resp.GetHasNext()
	totalElems := int(resp.GetTotalElements())
	localAccountList := api.LocalAccountList{
		LocalAccounts: &localAccounts,
		HasNext:       &hasNext,
		TotalElements: &totalElems,
	}

	payload := &types.Payload{Data: localAccountList}
	return payload, nil
}

func (h *localAccountHandler) Update(_ *types.Job) (*types.Payload, error) {
	// Unsupported, we should never reach this point
	err := errors.Errorfc(codes.Unimplemented, "you cannot update a localaccount, you can delete and create "+
		"a localaccount, if there are no dependants, instead")
	log.InfraSec().InfraErr(err).Msg("PATCH and PUT are unsupported operation for localaccount")
	return nil, err
}

func castLocalAccountAPI(payload *types.Payload) (*api.LocalAccount, error) {
	body, ok := payload.Data.(*api.LocalAccount)
	if !ok {
		err := errors.Errorfc(codes.InvalidArgument,
			"body format is not LocalAccount: %T",
			payload.Data,
		)
		log.InfraErr(err).Msgf("")
		return nil, err
	}
	return body, nil
}

func openapiLocalAccountToGrpcLocalAccount(body *api.LocalAccount) (*localaccountv1.LocalAccountResource, error) {
	localaccount := &localaccountv1.LocalAccountResource{}

	if body.Username != "" {
		localaccount.Username = body.Username
	}

	if body.SshKey != "" {
		localaccount.SshKey = body.SshKey
	}

	err := validator.ValidateMessage(localaccount)
	if err != nil {
		log.InfraSec().InfraErr(err).Msg("could not validate inventory resource")
		return nil, errors.Wrap(err)
	}
	return localaccount, nil
}

func GrpcLocalAccountToOpenAPIcreatedLocalAccount(
	localAccount *localaccountv1.LocalAccountResource,
) *api.LocalAccount {
	resID := localAccount.GetResourceId()
	Username := localAccount.GetUsername()
	Sshkey := localAccount.GetSshKey()

	obj := api.LocalAccount{
		Username:       Username,
		SshKey:         Sshkey,
		ResourceId:     &resID,
		LocalAccountID: &resID,
		Timestamps:     GrpcToOpenAPITimestamps(localAccount),
	}
	return &obj
}

func localAccountID(payload *types.Payload) (string, error) {
	params, ok := payload.Params.(LocalAccountURLParams)
	if !ok {
		err := errors.Errorfc(codes.InvalidArgument, "LocalAccountURLParams incorrectly formatted: %T",
			payload.Data)
		log.InfraErr(err).Msg("could not parse job payload params")
		return "", err
	}
	return params.LocalAccountID, nil
}

func castToLocalAccount(resp *inventory.GetResourceResponse) (
	*localaccountv1.LocalAccountResource, error,
) {
	if resp.GetResource().GetLocalAccount() != nil {
		return resp.GetResource().GetLocalAccount(), nil
	}
	err := errors.Errorfc(codes.Internal, "%s is not a LocalAccountResource", resp.GetResource())
	log.InfraErr(err).Msgf("could not cast inventory resource")

	return nil, err
}

func localAccountFilter(payload *types.Payload) (*inventory.ResourceFilter, error) {
	req := &inventory.ResourceFilter{
		Resource: &inventory.Resource{
			Resource: &inventory.Resource_LocalAccount{
				LocalAccount: &localaccountv1.LocalAccountResource{},
			},
		},
	}
	if payload.Data != nil {
		query, ok := payload.Data.(api.GetLocalAccountsParams)
		if !ok {
			err := errors.Errorfc(codes.InvalidArgument,
				"GetLocalaccountParams incorrectly formatted: %T",
				payload.Data,
			)
			log.InfraErr(err).Msg("list operation")
			return nil, err
		}
		err := castLocalAccountQueryList(&query, req)
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

func castLocalAccountQueryList(
	query *api.GetLocalAccountsParams,
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
