// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package store

// site.go - store information for Sites

import (
	"context"

	"golang.org/x/exp/slices"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	"github.com/open-edge-platform/infra-core/inventory/v2/internal/ent"
	"github.com/open-edge-platform/infra-core/inventory/v2/internal/ent/hostresource"
	"github.com/open-edge-platform/infra-core/inventory/v2/internal/ent/ouresource"
	sites "github.com/open-edge-platform/infra-core/inventory/v2/internal/ent/siteresource"
	inv_v1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/inventory/v1"
	location_v1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/location/v1"
	cl "github.com/open-edge-platform/infra-core/inventory/v2/pkg/client"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/errors"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/util"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/util/collections"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/validator"
)

var siteResourceCreationValidators = []resourceValidator[*location_v1.SiteResource]{
	protoValidator[*location_v1.SiteResource],
	doNotAcceptResourceID[*location_v1.SiteResource],
}

func (is *InvStore) CreateSite(ctx context.Context, in *location_v1.SiteResource) (*inv_v1.Resource, error) {
	if err := validate(in, siteResourceCreationValidators...); err != nil {
		return nil, err
	}

	res, err := ExecuteInTxAndReturnSingle[inv_v1.Resource](is)(ctx, siteResourceCreator(in))
	if err != nil {
		return nil, err
	}

	zlog.Debug().Msgf("Site Created: %s, %s", res.GetSite().GetResourceId(), res)

	return res, nil
}

func siteResourceCreator(in *location_v1.SiteResource) func(context.Context, *ent.Tx) (
	*inv_v1.Resource, error) {
	return func(ctx context.Context, tx *ent.Tx) (*inv_v1.Resource, error) {
		id := util.NewInvID(inv_v1.ResourceKind_RESOURCE_KIND_SITE)
		zlog.Debug().Msgf("CreateSite: %s", id)

		newEntity := tx.SiteResource.Create()
		mut := newEntity.Mutation()

		if err := buildEntMutate(in, mut, EmptyEnumStateMap, nil); err != nil {
			return nil, err
		}

		// Look up the optional region ID for this site.
		if err := setEdgeRegionIDForMut(ctx, tx.Client(), mut, in.GetRegion()); err != nil {
			return nil, err
		}
		// Look up the optional ou ID for this site.
		if in.GetOu() != nil {
			ou, qerr := tx.OuResource.Query().
				Where(ouresource.ResourceID(in.GetOu().ResourceId)).
				Only(ctx)
			if qerr != nil {
				return nil, errors.Wrap(qerr)
			}
			mut.SetOuID(ou.ID)
		}
		// Look up the optional provider ID for this site.
		if err := setEdgeProviderIDForMut(ctx, tx.Client(), mut, in.GetProvider()); err != nil {
			return nil, err
		}

		// Set the resource_id field last.
		if err := mut.SetField(sites.FieldResourceID, id); err != nil {
			return nil, errors.Wrap(err)
		}

		_, err := newEntity.Save(ctx)
		if err != nil {
			return nil, errors.Wrap(err)
		}

		res, _, err := getSiteQuery(ctx, tx, in.GetTenantId(), id, false)
		if err != nil {
			return nil, err
		}
		return util.WrapResource(entSiteResourceToProtoSiteResource(res))
	}
}

func (is *InvStore) GetSite(
	ctx context.Context, id string, tenantID string,
) (*inv_v1.Resource, *inv_v1.GetResourceResponse_ResourceMetadata, error) {
	res, resMeta, err := ExecuteInRoTxAndReturnDouble[ent.SiteResource, inv_v1.GetResourceResponse_ResourceMetadata](is)(
		ctx,
		func(ctx context.Context, tx *ent.Tx) (*ent.SiteResource, *inv_v1.GetResourceResponse_ResourceMetadata, error) {
			return getSiteQuery(ctx, tx, tenantID, id, true)
		})
	if err != nil {
		return nil, nil, err
	}

	apiResource := entSiteResourceToProtoSiteResource(res)
	if err = validator.ValidateMessage(apiResource); err != nil {
		zlog.InfraSec().InfraErr(err).Msg("")
		return nil, nil, errors.Wrap(err)
	}

	return &inv_v1.Resource{Resource: &inv_v1.Resource_Site{Site: apiResource}}, resMeta, nil
}

func getSiteQuery(ctx context.Context, tx *ent.Tx, tenantID, resourceID string, loadMetadata bool) (
	*ent.SiteResource, *inv_v1.GetResourceResponse_ResourceMetadata, error,
) {
	entity, err := tx.SiteResource.Query().
		Where(sites.ResourceID(resourceID)).
		WithRegion().
		WithProvider().
		WithOu().
		Only(ctx)
	if err != nil {
		return nil, nil, errors.Wrap(err)
	}

	if !loadMetadata {
		// Skip loading inherited metadata
		return entity, nil, nil
	}

	// Build metadata hierarchy
	var phyMeta, logiMeta map[int]map[string]string
	phyMeta, logiMeta, err = getSitesInheritedMeta(ctx, tx.Client(), []int{entity.ID}, tenantID)
	if err != nil {
		return nil, nil, err
	}

	resMeta := BuildResourceMeta(phyMeta[entity.ID], logiMeta[entity.ID])

	return entity, resMeta, nil
}

func (is *InvStore) UpdateSite(
	ctx context.Context, id string, in *location_v1.SiteResource, fieldmask *fieldmaskpb.FieldMask, tenantID string,
) (*inv_v1.Resource, error) {
	zlog.Debug().Msgf("UpdateSite (%s): %v, fm: %v", id, in, fieldmask)

	res, err := ExecuteInTxAndReturnSingle[inv_v1.Resource](is)(ctx,
		func(ctx context.Context, tx *ent.Tx) (*inv_v1.Resource, error) {
			entity, err := tx.SiteResource.Query().
				Select(sites.FieldID).
				Where(sites.ResourceID(id)).
				Only(ctx)
			if err != nil {
				return nil, errors.Wrap(err)
			}

			updateBuilder := tx.SiteResource.UpdateOneID(entity.ID)
			mut := updateBuilder.Mutation()

			// Look up the (new) referenced edges for this site.
			err = setRelationsForSiteMutIfNeeded(ctx, tx.Client(), mut, in, fieldmask)
			if err != nil {
				return nil, err
			}

			err = buildEntMutate(in, mut, EmptyEnumStateMap, fieldmask.GetPaths())
			if err != nil {
				return nil, err
			}

			// save UpdateOne
			_, err = updateBuilder.Save(ctx)
			if err != nil {
				return nil, errors.Wrap(err)
			}

			res, _, err := getSiteQuery(ctx, tx, tenantID, id, false)
			if err != nil {
				return nil, err
			}
			return util.WrapResource(entSiteResourceToProtoSiteResource(res))
		},
	)
	if err != nil {
		return nil, err
	}

	return res, err
}

func (is *InvStore) DeleteSite(ctx context.Context, id string) (*inv_v1.Resource, error) {
	// this is a "Hard Delete" as Sites don't have state
	zlog.Debug().Msgf("DeleteSite Hard Delete: %s", id)

	res, err := ExecuteInTxAndReturnSingle[inv_v1.Resource](is)(ctx, deleteSite(id))
	return res, err
}

func deleteSite(id string) func(ctx context.Context, tx *ent.Tx) (*inv_v1.Resource, error) {
	return func(ctx context.Context, tx *ent.Tx) (*inv_v1.Resource, error) {
		entity, err := tx.SiteResource.Query().
			Where(sites.ResourceID(id)).
			Only(ctx)
		if err != nil {
			return nil, errors.Wrap(err)
		}

		// FIXME: this could be solved with a back-reference to hosts in the schema. Not done due to protobuf circular dep.
		// Query any child host
		_, err = tx.HostResource.Query().
			Where(hostresource.HasSiteWith(sites.ResourceID(id))).
			First(ctx)
		if err != nil && !ent.IsNotFound(err) {
			// Unexpected error when querying hosts, rollback
			return nil, errors.Wrap(err)
		}
		if err == nil {
			// Site has a child site
			zlog.InfraSec().InfraError("the site has relations with host and cannot be deleted").Msg("")
			return nil, errors.Errorfc(codes.FailedPrecondition,
				"the site has relations with host and cannot be deleted")
		}

		if err := tx.SiteResource.DeleteOneID(entity.ID).Exec(ctx); err != nil {
			return nil, errors.Wrap(err)
		}

		return util.WrapResource(entSiteResourceToProtoSiteResource(entity))
	}
}

func (is *InvStore) DeleteSites(
	ctx context.Context, tenantID string, _ bool,
) ([]*util.Tuple[DeletionKind, *inv_v1.Resource], error) {
	var deleted []*util.Tuple[DeletionKind, *inv_v1.Resource]
	txErr := ExecuteInTx(is)(
		ctx,
		func(ctx context.Context, tx *ent.Tx) error {
			collection, err := tx.SiteResource.Query().Where(sites.TenantID(tenantID)).All(ctx)
			if err != nil {
				return err
			}
			if _, err := tx.SiteResource.Delete().Where(sites.TenantID(tenantID)).Exec(ctx); err != nil {
				return err
			}
			for _, element := range collection {
				res, err := util.WrapResource(entSiteResourceToProtoSiteResource(element))
				if err != nil {
					return err
				}
				deleted = append(deleted, util.NewTuple(HARD, res))
			}
			return nil
		})
	return deleted, txErr
}

func filterSites(ctx context.Context, client *ent.Client, filter *inv_v1.ResourceFilter, metadata bool) (
	[]siteWithInheritedMeta, int, error,
) {
	pred, err := getPredicate(inv_v1.ResourceKind_RESOURCE_KIND_SITE, filter.GetFilter())
	if err != nil {
		return nil, 0, err
	}

	orderOpts, err := GetOrderByOptions[sites.OrderOption](filter.GetOrderBy(), sites.ValidColumn)
	if err != nil {
		return nil, 0, err
	}

	offset, limit, err := getOffsetAndLimit(filter)
	if err != nil {
		return nil, 0, err
	}

	// perform query - And together all the predicates
	query := client.SiteResource.Query().
		WithRegion().
		WithOu().
		WithProvider().
		Where(pred).
		Order(orderOpts...).
		Offset(offset)

	// Limits number of query results if existent
	if limit != 0 {
		query = query.Limit(limit)
	}

	sitesList, err := query.All(ctx)
	if err != nil {
		return nil, 0, errors.Wrap(err)
	}

	// Count total number of item without applying pagination limits, order, or loading edges.
	total, err := client.SiteResource.Query().
		Where(pred).
		Count(ctx)
	if err != nil {
		return nil, 0, errors.Wrap(err)
	}

	var logiMeta, phyMeta map[int]map[string]string
	if metadata {
		// Gather the rendered metadata
		siteIDs := collections.MapSlice[*ent.SiteResource, int](sitesList, func(s *ent.SiteResource) int { return s.ID })
		phyMeta, logiMeta, err = getSitesInheritedMeta(ctx, client, siteIDs)
		if err != nil {
			return nil, 0, err
		}
	}
	siteWithMetaList := collections.MapSlice[*ent.SiteResource, siteWithInheritedMeta](
		sitesList,
		func(s *ent.SiteResource) siteWithInheritedMeta {
			return siteWithInheritedMeta{
				resource: s,
				meta: inheritedMeta{
					physical: phyMeta[s.ID],
					logical:  logiMeta[s.ID],
				},
			}
		})
	return siteWithMetaList, total, nil
}

func (is *InvStore) ListSites(ctx context.Context, filter *inv_v1.ResourceFilter) (
	[]*inv_v1.GetResourceResponse, int, error,
) {
	resources, total, err := ExecuteInRoTxAndReturnDouble[[]siteWithInheritedMeta, int](is)(
		ctx,
		func(ctx context.Context, tx *ent.Tx) (*[]siteWithInheritedMeta, *int, error) {
			resources, total, err := filterSites(ctx, tx.Client(), filter, true)
			if err != nil {
				return nil, nil, err
			}
			return &resources, &total, err
		},
	)
	if err != nil {
		return nil, 0, err
	}

	resps := collections.MapSlice[siteWithInheritedMeta, *inv_v1.GetResourceResponse](*resources,
		func(res siteWithInheritedMeta) *inv_v1.GetResourceResponse {
			return &inv_v1.GetResourceResponse{
				Resource: &inv_v1.Resource{
					Resource: &inv_v1.Resource_Site{
						Site: entSiteResourceToProtoSiteResource(res.resource),
					},
				},
				RenderedMetadata: BuildResourceMeta(res.meta.physical, res.meta.logical),
			}
		})
	if err := collections.FirstError[*inv_v1.GetResourceResponse](resps, validateProto[*inv_v1.GetResourceResponse]); err != nil {
		zlog.InfraSec().InfraErr(err).Msg("")
		return nil, 0, errors.Wrap(err)
	}

	return resps, *total, nil
}

func (is *InvStore) FilterSites(ctx context.Context, filter *inv_v1.ResourceFilter) ([]*cl.ResourceTenantIDCarrier, int, error) {
	resources, total, err := ExecuteInRoTxAndReturnDouble[[]siteWithInheritedMeta, int](is)(
		ctx, func(ctx context.Context, tx *ent.Tx) (*[]siteWithInheritedMeta, *int, error) {
			filteredSites, total, err := filterSites(ctx, tx.Client(), filter, false)
			if err != nil {
				return nil, nil, err
			}
			return &filteredSites, &total, nil
		})
	if err != nil {
		return nil, 0, err
	}

	ids := collections.MapSlice[siteWithInheritedMeta, *cl.ResourceTenantIDCarrier](
		*resources, func(c siteWithInheritedMeta) *cl.ResourceTenantIDCarrier {
			return &cl.ResourceTenantIDCarrier{TenantId: c.resource.TenantID, ResourceId: c.resource.ResourceID}
		})

	return ids, *total, err
}

func getSiteIDFromResourceID(ctx context.Context, client *ent.Client, siteRes *location_v1.SiteResource) (int, error) {
	site, qerr := client.SiteResource.Query().
		Where(sites.ResourceID(siteRes.ResourceId)).
		Only(ctx)
	if qerr != nil {
		return 0, errors.Wrap(qerr)
	}
	return site.ID, nil
}

func setRelationsForSiteMutIfNeeded(
	ctx context.Context,
	client *ent.Client,
	mut *ent.SiteResourceMutation,
	in *location_v1.SiteResource,
	fieldmask *fieldmaskpb.FieldMask,
) error {
	mut.ResetRegion()
	if slices.Contains(fieldmask.GetPaths(), sites.EdgeRegion) {
		if err := setEdgeRegionIDForMut(ctx, client, mut, in.GetRegion()); err != nil {
			return err
		}
	}
	mut.ResetOu()
	if in.GetOu() != nil && slices.Contains(fieldmask.GetPaths(), sites.EdgeOu) {
		ou, queryErr := client.OuResource.Query().
			Where(ouresource.ResourceID(in.GetOu().ResourceId)).
			Only(ctx)
		if queryErr != nil {
			return errors.Wrap(queryErr)
		}
		mut.SetOuID(ou.ID)
	}
	mut.ResetProvider()
	if slices.Contains(fieldmask.GetPaths(), sites.EdgeProvider) {
		if err := setEdgeProviderIDForMut(ctx, client, mut, in.GetProvider()); err != nil {
			return err
		}
	}
	return nil
}
