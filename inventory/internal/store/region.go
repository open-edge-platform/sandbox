// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package store

// location.go - store information for Regions and Sites

import (
	"context"

	"golang.org/x/exp/slices"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	"github.com/open-edge-platform/infra-core/inventory/v2/internal/ent"
	regions "github.com/open-edge-platform/infra-core/inventory/v2/internal/ent/regionresource"
	sites "github.com/open-edge-platform/infra-core/inventory/v2/internal/ent/siteresource"
	inv_v1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/inventory/v1"
	location_v1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/location/v1"
	cl "github.com/open-edge-platform/infra-core/inventory/v2/pkg/client"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/errors"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/util"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/util/collections"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/validator"
)

var regionResourceCreationValidators = []resourceValidator[*location_v1.RegionResource]{
	protoValidator[*location_v1.RegionResource],
	doNotAcceptResourceID[*location_v1.RegionResource],
}

func findAllConnectedChildRegions(
	ctx context.Context, tx *ent.Tx, region *ent.RegionResource, seen map[int]struct{},
) error {
	cs, err := region.Edges.ChildrenOrErr()
	zlog.Debug().Msgf("id %v, err %v, cs %v, seen %v", region.ID, err, cs, seen)
	if ent.IsNotFound(err) || ent.IsNotLoaded(err) {
		return nil // We found a leaf.
	}
	if err != nil {
		return errors.Wrap(err)
	}
	for _, c := range cs {
		cq, qerr := tx.RegionResource.Query().
			Where(regions.ResourceID(c.ResourceID)).
			WithParentRegion().
			WithChildren().
			Only(ctx)
		if qerr != nil {
			return errors.Wrap(qerr)
		}
		qerr = findAllConnectedRegions(ctx, tx, cq, seen)
		if qerr != nil {
			return qerr
		}
	}
	return nil
}

func findAllConnectedParentRegions(
	ctx context.Context, tx *ent.Tx, region *ent.RegionResource, seen map[int]struct{},
) error {
	p, err := region.Edges.ParentRegionOrErr()
	zlog.Debug().Msgf("id %v, err %v, p %v, seen %v", region.ID, err, p, seen)
	if ent.IsNotFound(err) || ent.IsNotLoaded(err) {
		return nil // We found a root.
	}
	if err != nil {
		return errors.Wrap(err)
	}
	pq, err := tx.RegionResource.Query().
		Where(regions.ResourceID(p.ResourceID)).
		WithParentRegion().
		WithChildren().
		Only(ctx)
	if err != nil {
		return errors.Wrap(err)
	}
	err = findAllConnectedRegions(ctx, tx, pq, seen)
	if err != nil {
		return err
	}
	return err
}

// findAllConnectedRegions traverses the tree both upwards and downwards to find
// all regions connected to the given one and saves their IDs in the seen map.
func findAllConnectedRegions(
	ctx context.Context, tx *ent.Tx, region *ent.RegionResource, seen map[int]struct{},
) error {
	if _, ok := seen[region.ID]; ok {
		return nil
	}
	seen[region.ID] = struct{}{}

	// Recurse into children.
	if err := findAllConnectedChildRegions(ctx, tx, region, seen); err != nil {
		return err
	}
	// Recurse into parent.
	// FIXME check not nil and wrap this
	return findAllConnectedParentRegions(ctx, tx, region, seen)
}

// checkParentNestingDepth traverses the tree upwards of the given ID and
// ensures the maximum nesting limit is observed.
func checkParentNestingDepth(ctx context.Context, tx *ent.Tx, id, depth int) error {
	depth++
	if depth > util.MaxResourceNestingLevel {
		zlog.InfraSec().InfraError("id %v, depth %v, fail", id, depth)
		return errors.Errorfc(codes.InvalidArgument,
			"resource %v exceeds maximum resource nesting depth of %d",
			id, util.MaxResourceNestingLevel)
	}
	region, err := tx.RegionResource.Query().
		Where(regions.ID(id)).
		WithParentRegion().
		Only(ctx)
	if err != nil {
		zlog.Debug().Msgf("id %v, err %v, depth %v", id, err, depth)
		return errors.Wrap(err)
	}
	p, err := region.Edges.ParentRegionOrErr()
	zlog.Debug().Msgf("id %v, err %v, depth %v, p %v", region.ID, err, depth, p)
	if ent.IsNotFound(err) {
		return nil // We found a root.
	}

	// FIXME check not nil and wrap this
	return checkParentNestingDepth(ctx, tx, p.ID, depth)
}

func checkNestingLimit(ctx context.Context, tx *ent.Tx, id int) error {
	// Query the latest state of the resource.
	region, err := tx.RegionResource.Query().
		Where(regions.ID(id)).
		WithParentRegion().
		WithChildren().
		Only(ctx)
	if err != nil {
		zlog.Debug().Msgf("id %v, err %v", id, err)
		return errors.Wrap(err)
	}

	// Build a set of all adjacent resources.
	cache := make(map[int]struct{})
	if err := findAllConnectedRegions(ctx, tx, region, cache); err != nil {
		return err
	}
	zlog.Debug().Msgf("cache: %v", cache)

	// Visit all resources and verify the depth check on them.
	for id := range cache {
		if err := checkParentNestingDepth(ctx, tx, id, 0); err != nil {
			return err
		}
	}

	return nil
}

func (is *InvStore) CreateRegion(ctx context.Context, in *location_v1.RegionResource) (*inv_v1.Resource, error) {
	if err := validate(in, regionResourceCreationValidators...); err != nil {
		return nil, err
	}

	res, err := ExecuteInTxAndReturnSingle[inv_v1.Resource](is)(ctx, regionResourceCreator(in))
	if err != nil {
		return nil, err
	}

	zlog.Debug().Msgf("Region Created: %s, %s", res.GetRegion().GetResourceId(), res)
	return res, nil
}

func regionResourceCreator(in *location_v1.RegionResource) func(context.Context, *ent.Tx) (
	*inv_v1.Resource, error) {
	return func(ctx context.Context, tx *ent.Tx) (*inv_v1.Resource, error) {
		id := util.NewInvID(inv_v1.ResourceKind_RESOURCE_KIND_REGION)
		zlog.Debug().Msgf("CreateRegion: %s", id)

		newEntity := tx.RegionResource.Create()
		mut := newEntity.Mutation()

		if err := buildEntMutate(in, mut, EmptyEnumStateMap, nil); err != nil {
			return nil, err
		}

		// Look up the optional parent region ID for this region.
		if err := setParentRegionForRegionMut(ctx, tx.Client(), mut, in.GetParentRegion()); err != nil {
			return nil, err
		}

		// Set the resource_id field last.
		err := mut.SetField(regions.FieldResourceID, id)
		if err != nil {
			return nil, errors.Wrap(err)
		}

		_, err = newEntity.Save(ctx)
		if err != nil {
			return nil, errors.Wrap(err)
		}

		res, _, err := getRegionQuery(ctx, tx, in.GetTenantId(), id, false)
		if err != nil {
			return nil, err
		}

		err = checkNestingLimit(ctx, tx, res.ID)
		if err != nil {
			return nil, errors.Wrap(err)
		}

		return util.WrapResource(entRegionResourceToProtoRegionResource(res))
	}
}

func (is *InvStore) GetRegion(
	ctx context.Context, id, tenantID string,
) (*inv_v1.Resource, *inv_v1.GetResourceResponse_ResourceMetadata, error) {
	res, resMeta, err := ExecuteInRoTxAndReturnDouble[ent.RegionResource, inv_v1.GetResourceResponse_ResourceMetadata](is)(
		ctx,
		func(ctx context.Context, tx *ent.Tx) (*ent.RegionResource, *inv_v1.GetResourceResponse_ResourceMetadata, error) {
			return getRegionQuery(ctx, tx, tenantID, id, true)
		})
	if err != nil {
		return nil, nil, err
	}

	apiResource := entRegionResourceToProtoRegionResource(res)
	if err = validator.ValidateMessage(apiResource); err != nil {
		zlog.InfraSec().InfraErr(err).Msg("")
		return nil, nil, errors.Wrap(err)
	}

	return &inv_v1.Resource{Resource: &inv_v1.Resource_Region{Region: apiResource}}, resMeta, nil
}

func getRegionQuery(ctx context.Context, tx *ent.Tx, tenantID, resourceID string, loadMetadata bool) (
	*ent.RegionResource, *inv_v1.GetResourceResponse_ResourceMetadata, error,
) {
	entity, err := tx.RegionResource.Query().
		Where(regions.ResourceID(resourceID)).
		WithParentRegion().
		Only(ctx)
	if err != nil {
		return nil, nil, errors.Wrap(err)
	}

	if !loadMetadata {
		// Avoid loading inherited metadata
		return entity, nil, nil
	}

	// Build metadata hierarchy
	renderedMeta, err := getRegionsInheritedMeta(ctx, tx.Client(), []int{entity.ID}, tenantID)
	if err != nil {
		return nil, nil, errors.Wrap(err)
	}

	resMeta := BuildResourceMeta(renderedMeta[entity.ID], map[string]string{})

	return entity, resMeta, nil
}

func (is *InvStore) UpdateRegion(
	ctx context.Context, id string, in *location_v1.RegionResource, fm *fieldmaskpb.FieldMask, tenantID string,
) (*inv_v1.Resource, error) {
	zlog.Debug().Msgf("Update (%s): %v, fm: %v", id, in, fm)

	res, err := ExecuteInTxAndReturnSingle[inv_v1.Resource](is)(ctx,
		func(ctx context.Context, tx *ent.Tx) (*inv_v1.Resource, error) {
			regq, err := tx.RegionResource.Query().
				Select(regions.FieldID).
				Where(regions.ResourceID(id)).
				Only(ctx)
			if err != nil {
				return nil, errors.Wrap(err)
			}

			updateBuilder := tx.RegionResource.UpdateOneID(regq.ID)
			mut := updateBuilder.Mutation()

			// Look up the (new) referenced parent region for this region.
			err = setRelationsForRegionMutIfNeeded(ctx, mut, tx, in, fm)
			if err != nil {
				return nil, err
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

			// Enforce the maximum nesting depth.
			err = checkNestingLimit(ctx, tx, regq.ID)
			if err != nil {
				return nil, err
			}

			// Get updated resource including eager loaded edges
			res, _, err := getRegionQuery(ctx, tx, tenantID, id, false)
			if err != nil {
				return nil, err
			}

			return util.WrapResource(entRegionResourceToProtoRegionResource(res))
		},
	)
	if err != nil {
		return nil, err
	}

	return res, err
}

func (is *InvStore) DeleteRegion(ctx context.Context, id string) (*inv_v1.Resource, error) {
	// this is a "Hard Delete" as Regions don't have state

	zlog.Debug().Msgf("DeleteRegion Hard Delete: %s", id)

	res, err := ExecuteInTxAndReturnSingle[inv_v1.Resource](is)(ctx, deleteRegion(id))

	return res, err
}

func deleteRegion(id string) func(ctx context.Context, tx *ent.Tx) (*inv_v1.Resource, error) {
	return func(ctx context.Context, tx *ent.Tx) (*inv_v1.Resource, error) {
		entity, err := tx.RegionResource.Query().
			Where(regions.ResourceID(id)).
			WithChildren().
			Only(ctx)
		if err != nil {
			return nil, errors.Wrap(err)
		}

		if len(entity.Edges.Children) != 0 {
			// Region has children
			zlog.InfraSec().InfraError("the region has relations with region and cannot be deleted").Msg("")
			return nil, errors.Errorfc(codes.FailedPrecondition,
				"the region has relations with region and cannot be deleted")
		}

		// FIXME: this could be solved with a back-reference to sites in the schema. Not done due to protobuf circular dep.
		// Query any child site
		_, err = tx.SiteResource.Query().
			Where(sites.HasRegionWith(regions.ResourceID(id))).
			First(ctx)
		if err != nil && !ent.IsNotFound(err) {
			// Unexpected error when querying sites, rollback
			return nil, errors.Wrap(err)
		}
		if err == nil {
			// Region has a child site
			zlog.InfraSec().InfraError("the region has relations with site and cannot be deleted").Msg("")
			return nil, errors.Errorfc(codes.FailedPrecondition,
				"the region has relations with site and cannot be deleted")
		}

		err = tx.RegionResource.DeleteOneID(entity.ID).Exec(ctx)
		if err != nil {
			return nil, errors.Wrap(err)
		}
		return util.WrapResource(entRegionResourceToProtoRegionResource(entity))
	}
}

func (is *InvStore) DeleteRegions(
	ctx context.Context, tenantID string, _ bool,
) ([]*util.Tuple[DeletionKind, *inv_v1.Resource], error) {
	var deleted []*util.Tuple[DeletionKind, *inv_v1.Resource]
	txErr := ExecuteInTx(is)(
		ctx,
		func(ctx context.Context, tx *ent.Tx) error {
			collection, err := tx.RegionResource.Query().
				Where(regions.TenantID(tenantID), regions.Not(regions.HasChildren())).All(ctx)
			if err != nil {
				return err
			}
			if _, err := tx.RegionResource.Delete().Where(regions.TenantID(tenantID)).Exec(ctx); err != nil {
				return err
			}
			for _, element := range collection {
				res, err := util.WrapResource(entRegionResourceToProtoRegionResource(element))
				if err != nil {
					return err
				}
				deleted = append(deleted, util.NewTuple(HARD, res))
			}
			return nil
		})
	return deleted, txErr
}

func filterRegion(ctx context.Context, client *ent.Client, filter *inv_v1.ResourceFilter, metadata bool) (
	[]regionWithInheritedMeta, int, error,
) {
	pred, err := getPredicate(inv_v1.ResourceKind_RESOURCE_KIND_REGION, filter.GetFilter())
	if err != nil {
		return nil, 0, err
	}

	orderOpts, err := GetOrderByOptions[regions.OrderOption](filter.GetOrderBy(), regions.ValidColumn)
	if err != nil {
		return nil, 0, err
	}

	offset, limit, err := getOffsetAndLimit(filter)
	if err != nil {
		return nil, 0, err
	}

	// perform query - And together all the predicates
	query := client.RegionResource.Query().
		WithParentRegion().
		Where(pred).
		Order(orderOpts...).
		Offset(offset)

	// Limits number of query results if existent
	if limit != 0 {
		query = query.Limit(limit)
	}

	regionsList, err := query.All(ctx)
	if err != nil {
		return nil, 0, errors.Wrap(err)
	}

	// Count total number of item without applying pagination limits, order, or loading edges.
	total, err := client.RegionResource.Query().
		Where(pred).
		Count(ctx)
	if err != nil {
		return nil, 0, errors.Wrap(err)
	}

	var phyMeta map[int]map[string]string
	if metadata {
		regionIDs := collections.MapSlice[*ent.RegionResource, int](regionsList, func(r *ent.RegionResource) int { return r.ID })
		phyMeta, err = getRegionsInheritedMeta(ctx, client, regionIDs)
		if err != nil {
			return nil, 0, err
		}
	}
	regionWithMetaList := collections.MapSlice[*ent.RegionResource, regionWithInheritedMeta](
		regionsList,
		func(r *ent.RegionResource) regionWithInheritedMeta {
			return regionWithInheritedMeta{
				resource: r,
				meta: inheritedMeta{
					physical: phyMeta[r.ID],
				},
			}
		})

	return regionWithMetaList, total, nil
}

func (is *InvStore) ListRegions(ctx context.Context, filter *inv_v1.ResourceFilter) (
	[]*inv_v1.GetResourceResponse, int, error,
) {
	resources, total, err := ExecuteInRoTxAndReturnDouble[[]regionWithInheritedMeta, int](is)(
		ctx,
		func(ctx context.Context, tx *ent.Tx) (*[]regionWithInheritedMeta, *int, error) {
			resources, total, err := filterRegion(ctx, tx.Client(), filter, true)
			if err != nil {
				return nil, nil, err
			}
			return &resources, &total, err
		},
	)
	if err != nil {
		return nil, 0, err
	}

	resps := collections.MapSlice[regionWithInheritedMeta, *inv_v1.GetResourceResponse](*resources,
		func(res regionWithInheritedMeta) *inv_v1.GetResourceResponse {
			return &inv_v1.GetResourceResponse{
				Resource: &inv_v1.Resource{
					Resource: &inv_v1.Resource_Region{
						Region: entRegionResourceToProtoRegionResource(res.resource),
					},
				},
				RenderedMetadata: BuildResourceMeta(res.meta.physical, map[string]string{}),
			}
		})
	if err := collections.FirstError[*inv_v1.GetResourceResponse](resps, validateProto[*inv_v1.GetResourceResponse]); err != nil {
		zlog.InfraSec().InfraErr(err).Msg("")
		return nil, 0, errors.Wrap(err)
	}

	return resps, *total, nil
}

func (is *InvStore) FilterRegions(ctx context.Context, filter *inv_v1.ResourceFilter) (
	[]*cl.ResourceTenantIDCarrier, int, error,
) {
	resources, total, err := ExecuteInRoTxAndReturnDouble[[]regionWithInheritedMeta, int](is)(
		ctx, func(ctx context.Context, tx *ent.Tx) (*[]regionWithInheritedMeta, *int, error) {
			filteredRegions, total, err := filterRegion(ctx, tx.Client(), filter, false)
			if err != nil {
				return nil, nil, err
			}
			return &filteredRegions, &total, nil
		})
	if err != nil {
		return nil, 0, err
	}

	ids := collections.MapSlice[regionWithInheritedMeta, *cl.ResourceTenantIDCarrier](
		*resources, func(c regionWithInheritedMeta) *cl.ResourceTenantIDCarrier {
			return &cl.ResourceTenantIDCarrier{TenantId: c.resource.TenantID, ResourceId: c.resource.ResourceID}
		})

	return ids, *total, err
}

func setRelationsForRegionMutIfNeeded(
	ctx context.Context,
	mut *ent.RegionResourceMutation,
	tx *ent.Tx,
	in *location_v1.RegionResource,
	fieldmask *fieldmaskpb.FieldMask,
) error {
	mut.ResetParentRegion()
	if slices.Contains(fieldmask.GetPaths(), regions.EdgeParentRegion) {
		if err := setParentRegionForRegionMut(ctx, tx.Client(), mut, in.GetParentRegion()); err != nil {
			return err
		}
	}
	return nil
}

func getRegionIDFromResourceID(ctx context.Context, client *ent.Client, regionRes *location_v1.RegionResource) (int, error) {
	region, qerr := client.RegionResource.Query().
		Where(regions.ResourceID(regionRes.ResourceId)).
		Only(ctx)
	if qerr != nil {
		return 0, errors.Wrap(qerr)
	}
	return region.ID, nil
}
