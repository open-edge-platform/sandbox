// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package store

// endpoint.go - store information for Endpoints

import (
	"context"

	"golang.org/x/exp/slices"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	"github.com/open-edge-platform/infra-core/inventory/v2/internal/ent"
	endpoints "github.com/open-edge-platform/infra-core/inventory/v2/internal/ent/endpointresource"
	"github.com/open-edge-platform/infra-core/inventory/v2/internal/ent/hostresource"
	inv_v1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/inventory/v1"
	network_v1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/network/v1"
	cl "github.com/open-edge-platform/infra-core/inventory/v2/pkg/client"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/errors"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/util"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/util/collections"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/validator"
)

var endpointResourceCreationValidators = []resourceValidator[*network_v1.EndpointResource]{
	protoValidator[*network_v1.EndpointResource],
	doNotAcceptResourceID[*network_v1.EndpointResource],
}

func (is *InvStore) CreateEndpoint(ctx context.Context, in *network_v1.EndpointResource) (*inv_v1.Resource, error) {
	if err := validate(in, endpointResourceCreationValidators...); err != nil {
		return nil, err
	}

	res, err := ExecuteInTxAndReturnSingle[inv_v1.Resource](is)(ctx, endpointResourceCreator(in))
	if err != nil {
		return nil, err
	}

	zlog.Debug().Msgf("Endpoint Created: %s, %s", res.GetEndpoint().GetResourceId(), res)
	return res, nil
}

func (is *InvStore) GetEndpoint(ctx context.Context, id string) (*inv_v1.Resource, error) {
	res, err := ExecuteInRoTxAndReturnSingle[ent.EndpointResource](is)(
		ctx,
		func(ctx context.Context, tx *ent.Tx) (*ent.EndpointResource, error) {
			return getEndpointQuery(ctx, tx, id)
		})
	if err != nil {
		return nil, err
	}

	apiResource := entEndpointResourceToProtoEndpointResource(res)
	if err = validator.ValidateMessage(apiResource); err != nil {
		zlog.InfraSec().InfraErr(err).Msg("")
		return nil, errors.Wrap(err)
	}

	return &inv_v1.Resource{Resource: &inv_v1.Resource_Endpoint{Endpoint: apiResource}}, nil
}

func getEndpointQuery(ctx context.Context, tx *ent.Tx, resourceID string) (*ent.EndpointResource, error) {
	entity, err := tx.EndpointResource.Query().
		Where(endpoints.ResourceID(resourceID)).
		WithHost().
		Only(ctx)
	if err != nil {
		return nil, errors.Wrap(err)
	}
	return entity, nil
}

func (is *InvStore) UpdateEndpoint(
	ctx context.Context, id string, in *network_v1.EndpointResource, fm *fieldmaskpb.FieldMask,
) (*inv_v1.Resource, error) {
	zlog.Debug().Msgf("Update (%s): %v, fm: %v", id, in, fm)

	res, err := ExecuteInTxAndReturnSingle[inv_v1.Resource](is)(ctx,
		func(ctx context.Context, tx *ent.Tx) (*inv_v1.Resource, error) {
			endpointq, err := tx.EndpointResource.Query().
				Select(endpoints.FieldID).
				Where(endpoints.ResourceID(id)).
				Only(ctx)
			if err != nil {
				return nil, errors.Wrap(err)
			}

			updateBuilder := tx.EndpointResource.UpdateOneID(endpointq.ID)
			mut := updateBuilder.Mutation()

			// Look up the (new) referenced edges for this endpoint.
			mut.ResetHost()
			if in.GetHost() != nil && slices.Contains(fm.GetPaths(), "host") {
				host, queryErr := tx.HostResource.Query().
					Where(hostresource.ResourceID(in.GetHost().ResourceId)).
					Only(ctx)
				if queryErr != nil {
					return nil, errors.Wrap(err)
				}
				mut.SetHostID(host.ID)
			}

			err = buildEntMutate(in, mut, EmptyEnumStateMap, fm.GetPaths())
			if err != nil {
				return nil, err
			}

			// save UpdateOne
			_, err = updateBuilder.Save(ctx)
			if err != nil {
				return nil, errors.Wrap(err)
			}

			res, err := getEndpointQuery(ctx, tx, id)
			if err != nil {
				return nil, err
			}
			return util.WrapResource(entEndpointResourceToProtoEndpointResource(res))
		},
	)
	if err != nil {
		return nil, err
	}

	return res, err
}

func endpointResourceCreator(in *network_v1.EndpointResource) func(context.Context, *ent.Tx) (
	*inv_v1.Resource, error) {
	return func(ctx context.Context, tx *ent.Tx) (*inv_v1.Resource, error) {
		id := util.NewInvID(inv_v1.ResourceKind_RESOURCE_KIND_ENDPOINT)
		zlog.Debug().Msgf("CreateEndpoint: %s", id)

		newEntity := tx.EndpointResource.Create()
		mut := newEntity.Mutation()

		if err := buildEntMutate(in, mut, EmptyEnumStateMap, nil); err != nil {
			return nil, err
		}

		// Look up the optional host ID for this endpoint.
		if in.GetHost() != nil {
			host, qerr := tx.HostResource.Query().
				Where(hostresource.ResourceID(in.GetHost().ResourceId)).
				Only(ctx)
			if qerr != nil {
				return nil, errors.Wrap(qerr)
			}
			mut.SetHostID(host.ID)
		}

		// Set the resource_id field last.
		if err := mut.SetField(endpoints.FieldResourceID, id); err != nil {
			return nil, errors.Wrap(err)
		}

		_, err := newEntity.Save(ctx)
		if err != nil {
			return nil, errors.Wrap(err)
		}

		res, err := getEndpointQuery(ctx, tx, id)
		if err != nil {
			return nil, err
		}
		return util.WrapResource(entEndpointResourceToProtoEndpointResource(res))
	}
}

func (is *InvStore) DeleteEndpoint(ctx context.Context, id string) (*inv_v1.Resource, error) {
	// this is a "Hard Delete" as Endpoints don't have state

	// FIXME - it should be impossible to delete a Endpoint that has other
	// dependent resources that are owned by the Endpoint
	zlog.Debug().Msgf("DeleteEndpoint Hard Delete: %s", id)

	res, err := ExecuteInTxAndReturnSingle[inv_v1.Resource](is)(ctx, deleteEndpoint(id))

	return res, err
}

type transactional func(ctx context.Context, tx *ent.Tx) (*inv_v1.Resource, error)

func deleteEndpoint(resourceID string) transactional {
	return func(ctx context.Context, tx *ent.Tx) (*inv_v1.Resource, error) {
		entity, err := tx.EndpointResource.Query().
			Where(endpoints.ResourceID(resourceID)).
			Only(ctx)
		if err != nil {
			return nil, errors.Wrap(err)
		}

		err = tx.EndpointResource.DeleteOneID(entity.ID).Exec(ctx)
		if err != nil {
			return nil, errors.Wrap(err)
		}
		return util.WrapResource(entEndpointResourceToProtoEndpointResource(entity))
	}
}

func (is *InvStore) ListEndpoints(ctx context.Context, filter *inv_v1.ResourceFilter) (
	[]*inv_v1.GetResourceResponse, int, error,
) {
	endpointResources, total, err := ExecuteInRoTxAndReturnDouble[[]*ent.EndpointResource, int](is)(
		ctx,
		func(ctx context.Context, tx *ent.Tx) (*[]*ent.EndpointResource, *int, error) {
			endpointResources, total, err := filterEndpoints(ctx, tx.Client(), filter)
			if err != nil {
				return nil, nil, err
			}
			return &endpointResources, &total, err
		},
	)
	if err != nil {
		return nil, 0, err
	}

	resps := collections.MapSlice[*ent.EndpointResource, *inv_v1.GetResourceResponse](*endpointResources,
		func(e *ent.EndpointResource) *inv_v1.GetResourceResponse {
			return &inv_v1.GetResourceResponse{
				Resource: &inv_v1.Resource{
					Resource: &inv_v1.Resource_Endpoint{
						Endpoint: entEndpointResourceToProtoEndpointResource(e),
					},
				},
			}
		})
	if err := collections.FirstError[*inv_v1.GetResourceResponse](resps, validateProto[*inv_v1.GetResourceResponse]); err != nil {
		zlog.InfraSec().InfraErr(err).Msg("")
		return nil, 0, errors.Wrap(err)
	}

	return resps, *total, nil
}

func filterEndpoints(ctx context.Context, client *ent.Client, filter *inv_v1.ResourceFilter) (
	[]*ent.EndpointResource, int, error,
) {
	pred, err := getPredicate(inv_v1.ResourceKind_RESOURCE_KIND_ENDPOINT, filter.GetFilter())
	if err != nil {
		return nil, 0, err
	}

	orderOpts, err := GetOrderByOptions[endpoints.OrderOption](filter.GetOrderBy(), endpoints.ValidColumn)
	if err != nil {
		return nil, 0, err
	}

	offset, limit, err := getOffsetAndLimit(filter)
	if err != nil {
		return nil, 0, err
	}

	// perform query - And together all the predicates
	query := client.EndpointResource.Query().
		WithHost().
		Where(pred).
		Order(orderOpts...).
		Offset(offset)

	// Limits number of query results if existent
	if limit != 0 {
		query = query.Limit(limit)
	}

	endpointList, err := query.All(ctx)
	if err != nil {
		return nil, 0, errors.Wrap(err)
	}

	// Count total number of item without applying pagination limits, order, or loading edges.
	total, err := client.EndpointResource.Query().
		Where(pred).
		Count(ctx)
	if err != nil {
		return nil, 0, errors.Wrap(err)
	}

	return endpointList, total, nil
}

func (is *InvStore) FilterEndpoints(ctx context.Context, filter *inv_v1.ResourceFilter) (
	[]*cl.ResourceTenantIDCarrier, int, error,
) {
	endpointResources, total, err := ExecuteInRoTxAndReturnDouble[[]*ent.EndpointResource, int](is)(
		ctx, func(ctx context.Context, tx *ent.Tx) (*[]*ent.EndpointResource, *int, error) {
			racs, total, err := filterEndpoints(ctx, tx.Client(), filter)
			if err != nil {
				return nil, nil, err
			}
			return &racs, &total, nil
		})
	if err != nil {
		return nil, 0, err
	}

	ids := collections.MapSlice[*ent.EndpointResource, *cl.ResourceTenantIDCarrier](
		*endpointResources, func(c *ent.EndpointResource) *cl.ResourceTenantIDCarrier {
			return &cl.ResourceTenantIDCarrier{TenantId: c.TenantID, ResourceId: c.ResourceID}
		})

	return ids, *total, err
}

func (is *InvStore) DeleteEndpoints(
	ctx context.Context, tenantID string, _ bool,
) ([]*util.Tuple[DeletionKind, *inv_v1.Resource], error) {
	var deleted []*util.Tuple[DeletionKind, *inv_v1.Resource]
	txErr := ExecuteInTx(is)(
		ctx,
		func(ctx context.Context, tx *ent.Tx) error {
			all, err := tx.EndpointResource.Query().Where(endpoints.TenantID(tenantID)).All(ctx)
			if err != nil {
				return err
			}

			if _, err := tx.EndpointResource.Delete().Where(endpoints.TenantID(tenantID)).Exec(ctx); err != nil {
				return err
			}
			for _, element := range all {
				res, err := util.WrapResource(entEndpointResourceToProtoEndpointResource(element))
				if err != nil {
					return err
				}
				deleted = append(deleted, util.NewTuple(HARD, res))
			}
			return nil
		})
	return deleted, txErr
}
