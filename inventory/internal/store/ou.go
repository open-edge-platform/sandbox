// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package store

import (
	"context"

	"golang.org/x/exp/slices"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	"github.com/open-edge-platform/infra-core/inventory/v2/internal/ent"
	"github.com/open-edge-platform/infra-core/inventory/v2/internal/ent/ouresource"
	sites "github.com/open-edge-platform/infra-core/inventory/v2/internal/ent/siteresource"
	inv_v1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/inventory/v1"
	ou_v1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/ou/v1"
	cl "github.com/open-edge-platform/infra-core/inventory/v2/pkg/client"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/errors"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/util"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/util/collections"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/validator"
)

var ouResourceCreationValidators = []resourceValidator[*ou_v1.OuResource]{
	protoValidator[*ou_v1.OuResource],
	doNotAcceptResourceID[*ou_v1.OuResource],
}

func findAllConnectedChildOus(ctx context.Context, tx *ent.Tx, ou *ent.OuResource, seen map[int]struct{}) error {
	cs, err := ou.Edges.ChildrenOrErr()
	zlog.Debug().Msgf("id %v, err %v, cs %v, seen %v", ou.ID, err, cs, seen)
	if isEntLeaf(err) {
		return nil
	}
	if err != nil {
		return errors.Wrap(err)
	}
	for _, c := range cs {
		cq, qerr := tx.OuResource.Query().
			Where(ouresource.ResourceID(c.ResourceID)).
			WithParentOu().
			WithChildren().
			Only(ctx)
		if qerr != nil {
			return errors.Wrap(qerr)
		}
		qerr = findAllConnectedOus(ctx, tx, cq, seen)
		if qerr != nil {
			return qerr
		}
	}
	return nil
}

func findAllConnectedParentOus(ctx context.Context, tx *ent.Tx, ou *ent.OuResource, seen map[int]struct{}) error {
	p, err := ou.Edges.ParentOuOrErr()
	zlog.Debug().Msgf("id %v, err %v, p %v, seen %v", ou.ID, err, p, seen)
	if isEntRoot(err) {
		return nil
	}
	if err != nil {
		return errors.Wrap(err)
	}
	pq, err := tx.OuResource.Query().
		Where(ouresource.ResourceID(p.ResourceID)).
		WithParentOu().
		WithChildren().
		Only(ctx)
	if err != nil {
		return errors.Wrap(err)
	}

	// FIXME check not nil and wrap this
	return findAllConnectedOus(ctx, tx, pq, seen)
}

// findAllConnectedOus traverses the tree both upwards and downwards to find
// all ous connected to the given one and saves their IDs in the seen map.
func findAllConnectedOus(ctx context.Context, tx *ent.Tx, ou *ent.OuResource, seen map[int]struct{}) error {
	if _, ok := seen[ou.ID]; ok {
		return nil
	}
	seen[ou.ID] = struct{}{}

	// Recurse into children.
	if err := findAllConnectedChildOus(ctx, tx, ou, seen); err != nil {
		return err
	}
	// Recurse into parent.
	// FIXME check not nil and wrap this
	return findAllConnectedParentOus(ctx, tx, ou, seen)
}

// checkParentNestingDepth traverses the tree upwards of the given ID and
// ensures the maximum nesting limit is observed.
func checkParentNestingDepthOu(ctx context.Context, tx *ent.Tx, id, depth int) error {
	depth++
	if depth > util.MaxResourceNestingLevel {
		zlog.InfraSec().InfraError("id %v, depth %v, fail", id, depth)
		return errors.Errorfc(codes.InvalidArgument,
			"resource %v exceeds maximum resource nesting depth of %d",
			id, util.MaxResourceNestingLevel)
	}
	ou, err := tx.OuResource.Query().
		Where(ouresource.ID(id)).
		WithParentOu().
		Only(ctx)
	if err != nil {
		zlog.Debug().Msgf("id %v, err %v, depth %v", id, err, depth)
		return errors.Wrap(err)
	}
	p, err := ou.Edges.ParentOuOrErr()
	zlog.Debug().Msgf("id %v, err %v, depth %v, p %v", ou.ID, err, depth, p)
	if ent.IsNotFound(err) {
		return nil // We found a root.
	}

	// FIXME check not nil and wrap this
	return checkParentNestingDepthOu(ctx, tx, p.ID, depth)
}

func checkNestingLimitOu(ctx context.Context, tx *ent.Tx, id int) error {
	// Query the latest state of the resource.
	ou, err := tx.OuResource.Query().
		Where(ouresource.ID(id)).
		WithParentOu().
		WithChildren().
		Only(ctx)
	if err != nil {
		zlog.Debug().Msgf("id %v, err %v", id, err)
		return errors.Wrap(err)
	}

	// Build a set of all adjacent resources.
	cache := make(map[int]struct{})
	if err := findAllConnectedOus(ctx, tx, ou, cache); err != nil {
		return err
	}
	zlog.Debug().Msgf("cache: %v", cache)

	// Visit all resources and verify the depth check on them.
	for id := range cache {
		if err := checkParentNestingDepthOu(ctx, tx, id, 0); err != nil {
			return err
		}
	}

	return nil
}

func (is *InvStore) CreateOu(ctx context.Context, in *ou_v1.OuResource) (*inv_v1.Resource, error) {
	if err := validate(in, ouResourceCreationValidators...); err != nil {
		return nil, err
	}

	res, err := ExecuteInTxAndReturnSingle[inv_v1.Resource](is)(ctx, ouResourceCreator(in))
	if err != nil {
		return nil, err
	}

	zlog.Debug().Msgf("OU Created: %s, %s", res.GetOu().GetResourceId(), res)

	return res, nil
}

func ouResourceCreator(in *ou_v1.OuResource) func(context.Context, *ent.Tx) (
	*inv_v1.Resource, error) {
	return func(ctx context.Context, tx *ent.Tx) (*inv_v1.Resource, error) {
		id := util.NewInvID(inv_v1.ResourceKind_RESOURCE_KIND_OU)
		zlog.Debug().Msgf("CreateOu: %s", id)

		newEntity := tx.OuResource.Create()
		mut := newEntity.Mutation()

		if err := buildEntMutate(in, mut, EmptyEnumStateMap, nil); err != nil {
			return nil, err
		}

		// Look up the optional parent ou ID for this OU.
		if err := setParentOuForOuMut(ctx, mut, tx, in.GetParentOu()); err != nil {
			return nil, err
		}

		err := mut.SetField(ouresource.FieldResourceID, id)
		if err != nil {
			return nil, errors.Wrap(err)
		}

		resSave, err := newEntity.Save(ctx)
		if err != nil {
			return nil, errors.Wrap(err)
		}

		// Enforce the maximum nesting depth.
		err = checkNestingLimitOu(ctx, tx, resSave.ID)
		if err != nil {
			return nil, err
		}

		res, _, err := getOuQuery(ctx, tx, in.GetTenantId(), id, false)
		if err != nil {
			return nil, err
		}
		return util.WrapResource(entOuResourceToProtoOuResource(res))
	}
}

func (is *InvStore) GetOu(
	ctx context.Context, id, tenantID string,
) (*inv_v1.Resource, *inv_v1.GetResourceResponse_ResourceMetadata, error) {
	res, resMeta, err := ExecuteInRoTxAndReturnDouble[ent.OuResource, inv_v1.GetResourceResponse_ResourceMetadata](is)(
		ctx,
		func(ctx context.Context, tx *ent.Tx) (*ent.OuResource, *inv_v1.GetResourceResponse_ResourceMetadata, error) {
			return getOuQuery(ctx, tx, tenantID, id, true)
		})
	if err != nil {
		return nil, nil, err
	}

	apiResource := entOuResourceToProtoOuResource(res)
	if err = validator.ValidateMessage(apiResource); err != nil {
		zlog.InfraSec().InfraErr(err).Msg("")
		return nil, nil, errors.Wrap(err)
	}

	return &inv_v1.Resource{Resource: &inv_v1.Resource_Ou{Ou: apiResource}}, resMeta, nil
}

func getOuQuery(ctx context.Context, tx *ent.Tx, tenantID, resourceID string, loadMetadata bool) (
	*ent.OuResource, *inv_v1.GetResourceResponse_ResourceMetadata, error,
) {
	entity, err := tx.OuResource.Query().
		Where(ouresource.ResourceID(resourceID)).
		WithParentOu().
		Only(ctx)
	if err != nil {
		return nil, nil, errors.Wrap(err)
	}

	if !loadMetadata {
		// Avoid loading inherited metadata
		return entity, nil, nil
	}

	// Build metadata hierarchy
	renderedMeta, err := getOusInheritedMeta(ctx, tx.Client(), []int{entity.ID}, tenantID)
	if err != nil {
		return nil, nil, errors.Wrap(err)
	}

	resMeta := BuildResourceMeta(map[string]string{}, renderedMeta[entity.ID])

	return entity, resMeta, nil
}

func (is *InvStore) UpdateOu(
	ctx context.Context, id string, in *ou_v1.OuResource, fieldmask *fieldmaskpb.FieldMask, tenantID string,
) (*inv_v1.Resource, error) {
	zlog.Debug().Msgf("UpdateOu (%s): %v, fm: %v", id, in, fieldmask)

	res, err := ExecuteInTxAndReturnSingle[inv_v1.Resource](is)(
		ctx,
		func(ctx context.Context, tx *ent.Tx) (*inv_v1.Resource, error) {
			entity, err := tx.OuResource.Query().
				Select(ouresource.FieldID).
				Where(ouresource.ResourceID(id)).
				Only(ctx)
			if err != nil {
				return nil, errors.Wrap(err)
			}

			updateBuilder := tx.OuResource.UpdateOneID(entity.ID)
			mut := updateBuilder.Mutation()

			// Look up the (new) referenced parent ou for this ou.
			err = setRelationsForOuMutIfNeeded(ctx, mut, tx, in, fieldmask)
			if err != nil {
				return nil, err
			}

			err = buildEntMutate(in, mut, EmptyEnumStateMap, fieldmask.GetPaths())
			if err != nil {
				return nil, err
			}

			_, err = updateBuilder.Save(ctx)
			if err != nil {
				return nil, errors.Wrap(err)
			}

			// Enforce the maximum nesting depth.
			err = checkNestingLimitOu(ctx, tx, entity.ID)
			if err != nil {
				return nil, err
			}

			res, _, err := getOuQuery(ctx, tx, tenantID, id, false)
			if err != nil {
				return nil, err
			}
			return util.WrapResource(entOuResourceToProtoOuResource(res))
		})
	if err != nil {
		return nil, err
	}

	return res, err
}

func (is *InvStore) DeleteOu(ctx context.Context, id string) (*inv_v1.Resource, error) {
	// this is a "Hard Delete" as Ous don't have state
	zlog.Debug().Msgf("DeleteOu Hard Delete: %s", id)

	res, err := ExecuteInTxAndReturnSingle[inv_v1.Resource](is)(
		ctx,
		deleteOu(id))

	return res, err
}

func deleteOu(id string) func(ctx context.Context, tx *ent.Tx) (*inv_v1.Resource, error) {
	return func(ctx context.Context, tx *ent.Tx) (*inv_v1.Resource, error) {
		entity, err := tx.OuResource.Query().
			Where(ouresource.ResourceID(id)).
			WithChildren().
			Only(ctx)
		if err != nil {
			return nil, errors.Wrap(err)
		}

		if len(entity.Edges.Children) != 0 {
			// OU has children
			zlog.InfraSec().InfraError("the ou has relations with ou and cannot be deleted").Msg("")
			return nil, errors.Errorfc(codes.FailedPrecondition,
				"the ou has relations with ou and cannot be deleted")
		}

		// FIXME: this could be solved with a back-reference to sites in the schema. Not done due to protobuf circular dep.
		// Query any child site
		_, err = tx.SiteResource.Query().
			Where(sites.HasOuWith(ouresource.ResourceID(id))).
			First(ctx)
		if err != nil && !ent.IsNotFound(err) {
			// Unexpected error when querying sites, rollback
			return nil, errors.Wrap(err)
		}
		if err == nil {
			// OU has a child site
			zlog.InfraSec().InfraError("the ou has relations with site and cannot be deleted").Msg("")
			return nil, errors.Errorfc(codes.FailedPrecondition,
				"the ou has relations with site and cannot be deleted")
		}

		err = tx.OuResource.DeleteOneID(entity.ID).Exec(ctx)
		if err != nil {
			return nil, errors.Wrap(err)
		}

		return util.WrapResource(entOuResourceToProtoOuResource(entity))
	}
}

func (is *InvStore) DeleteOus(
	ctx context.Context, tenantID string, _ bool,
) ([]*util.Tuple[DeletionKind, *inv_v1.Resource], error) {
	var deleted []*util.Tuple[DeletionKind, *inv_v1.Resource]
	txErr := ExecuteInTx(is)(
		ctx,
		func(ctx context.Context, tx *ent.Tx) error {
			collection, err := tx.OuResource.Query().Where(ouresource.TenantID(tenantID)).All(ctx)
			if err != nil {
				return err
			}
			if _, err := tx.OuResource.Delete().Where(ouresource.TenantID(tenantID)).Exec(ctx); err != nil {
				return err
			}
			for _, element := range collection {
				res, err := util.WrapResource(entOuResourceToProtoOuResource(element))
				if err != nil {
					return err
				}
				deleted = append(deleted, util.NewTuple(HARD, res))
			}
			return nil
		})
	return deleted, txErr
}

func filterOus(ctx context.Context, client *ent.Client, filter *inv_v1.ResourceFilter, metadata bool) (
	[]ouWithInheritedMeta, int, error,
) {
	pred, err := getPredicate(inv_v1.ResourceKind_RESOURCE_KIND_OU, filter.GetFilter())
	if err != nil {
		return nil, 0, err
	}

	orderOpts, err := GetOrderByOptions[ouresource.OrderOption](filter.GetOrderBy(), ouresource.ValidColumn)
	if err != nil {
		return nil, 0, err
	}

	offset, limit, err := getOffsetAndLimit(filter)
	if err != nil {
		return nil, 0, err
	}

	// perform query - And together all the predicates
	query := client.OuResource.Query().
		WithParentOu().
		Where(pred).
		Order(orderOpts...).
		Offset(offset)

	// Limits number of query results if existent
	if limit != 0 {
		query = query.Limit(limit)
	}

	ousList, err := query.All(ctx)
	if err != nil {
		return nil, 0, errors.Wrap(err)
	}

	// Count total number of item without applying pagination limits, order, or loading edges.
	total, err := client.OuResource.Query().
		Where(pred).
		Count(ctx)
	if err != nil {
		return nil, 0, errors.Wrap(err)
	}

	// Gather the rendered metadata
	var logiMeta map[int]map[string]string
	if metadata {
		ouIDs := collections.MapSlice[*ent.OuResource, int](ousList, func(o *ent.OuResource) int { return o.ID })
		logiMeta, err = getOusInheritedMeta(ctx, client, ouIDs)
		if err != nil {
			return nil, 0, err
		}
	}
	ouWithMetaList := collections.MapSlice[*ent.OuResource, ouWithInheritedMeta](
		ousList,
		func(o *ent.OuResource) ouWithInheritedMeta {
			return ouWithInheritedMeta{
				resource: o,
				meta: inheritedMeta{
					logical: logiMeta[o.ID],
				},
			}
		})
	return ouWithMetaList, total, nil
}

func (is *InvStore) ListOus(ctx context.Context, filter *inv_v1.ResourceFilter) (
	[]*inv_v1.GetResourceResponse, int, error,
) {
	resources, total, err := ExecuteInRoTxAndReturnDouble[[]ouWithInheritedMeta, int](is)(
		ctx,
		func(ctx context.Context, tx *ent.Tx) (*[]ouWithInheritedMeta, *int, error) {
			resources, total, err := filterOus(ctx, tx.Client(), filter, true)
			if err != nil {
				return nil, nil, err
			}
			return &resources, &total, err
		},
	)
	if err != nil {
		return nil, 0, err
	}

	resps := collections.MapSlice[ouWithInheritedMeta, *inv_v1.GetResourceResponse](*resources,
		func(res ouWithInheritedMeta) *inv_v1.GetResourceResponse {
			return &inv_v1.GetResourceResponse{
				Resource: &inv_v1.Resource{
					Resource: &inv_v1.Resource_Ou{
						Ou: entOuResourceToProtoOuResource(res.resource),
					},
				},
				RenderedMetadata: BuildResourceMeta(map[string]string{}, res.meta.logical),
			}
		})
	if err := collections.FirstError[*inv_v1.GetResourceResponse](resps, validateProto[*inv_v1.GetResourceResponse]); err != nil {
		zlog.InfraSec().InfraErr(err).Msg("")
		return nil, 0, errors.Wrap(err)
	}

	return resps, *total, nil
}

func (is *InvStore) FilterOus(ctx context.Context, filter *inv_v1.ResourceFilter) ([]*cl.ResourceTenantIDCarrier, int, error) {
	resources, total, err := ExecuteInRoTxAndReturnDouble[[]ouWithInheritedMeta, int](is)(
		ctx, func(ctx context.Context, tx *ent.Tx) (*[]ouWithInheritedMeta, *int, error) {
			filteredOus, total, err := filterOus(ctx, tx.Client(), filter, false)
			if err != nil {
				return nil, nil, err
			}
			return &filteredOus, &total, nil
		})
	if err != nil {
		return nil, 0, err
	}
	ids := collections.MapSlice[ouWithInheritedMeta, *cl.ResourceTenantIDCarrier](
		*resources, func(c ouWithInheritedMeta) *cl.ResourceTenantIDCarrier {
			return &cl.ResourceTenantIDCarrier{TenantId: c.resource.TenantID, ResourceId: c.resource.ResourceID}
		})

	return ids, *total, err
}

func getOuIDFromResourceID(ctx context.Context, tx *ent.Tx, ouRes *ou_v1.OuResource) (int, error) {
	ou, qerr := tx.OuResource.Query().
		Where(ouresource.ResourceID(ouRes.ResourceId)).
		Only(ctx)
	if qerr != nil {
		return 0, errors.Wrap(qerr)
	}
	return ou.ID, nil
}

func setParentOuForOuMut(
	ctx context.Context, mut *ent.OuResourceMutation, tx *ent.Tx, oures *ou_v1.OuResource,
) error {
	if oures != nil {
		ouID, qerr := getOuIDFromResourceID(ctx, tx, oures)
		if qerr != nil {
			return qerr
		}
		mut.SetParentOuID(ouID)
	}
	return nil
}

func setRelationsForOuMutIfNeeded(
	ctx context.Context, mut *ent.OuResourceMutation, tx *ent.Tx, in *ou_v1.OuResource, fieldmask *fieldmaskpb.FieldMask,
) error {
	mut.ResetParentOu()
	if slices.Contains(fieldmask.GetPaths(), ouresource.EdgeParentOu) {
		if err := setParentOuForOuMut(ctx, mut, tx, in.GetParentOu()); err != nil {
			return err
		}
	}
	return nil
}
