// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package store

import (
	"context"
	"strconv"
	"strings"

	"entgo.io/ent/dialect/sql"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	"github.com/open-edge-platform/infra-core/inventory/v2/internal/ent"
	"github.com/open-edge-platform/infra-core/inventory/v2/internal/ent/predicate"
	telemetryprofileres "github.com/open-edge-platform/infra-core/inventory/v2/internal/ent/telemetryprofile"
	inv_v1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/inventory/v1"
	telemetry_v1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/telemetry/v1"
	cl "github.com/open-edge-platform/infra-core/inventory/v2/pkg/client"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/errors"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/util"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/util/collections"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/validator"
)

// telemetryprofile.go  store information for TelemetryProfile objects

var telemetryProfileCreationValidators = []resourceValidator[*telemetry_v1.TelemetryProfile]{
	protoValidator[*telemetry_v1.TelemetryProfile],
	// Validator on proto resource do not need to enforce relations, those are enforced by the oneof of protobuf
	validateTelemetryProfileParameters,
	doNotAcceptResourceID[*telemetry_v1.TelemetryProfile],
}

func TelemetryProfileEnumStatusMap(fname string, eint int32) (ent.Value, error) {
	switch fname {
	case telemetryprofileres.FieldKind:
		return telemetryprofileres.Kind(telemetry_v1.TelemetryResourceKind_name[eint]), nil
	case telemetryprofileres.FieldLogLevel:
		return telemetryprofileres.LogLevel(telemetry_v1.SeverityLevel_name[eint]), nil
	default:
		zlog.InfraSec().InfraError("unknown Enum field %s", fname).Msg("")
		return nil, errors.Errorfc(codes.InvalidArgument, "unknown Enum field %s", fname)
	}
}

func (is *InvStore) CreateTelemetryProfile(
	ctx context.Context,
	in *telemetry_v1.TelemetryProfile,
) (*inv_v1.Resource, error) {
	if err := validate(in, telemetryProfileCreationValidators...); err != nil {
		return nil, err
	}

	res, err := ExecuteInTxAndReturnSingle[inv_v1.Resource](is)(ctx, telemetryProfileCreator(in))
	if err != nil {
		return nil, err
	}

	zlog.Debug().Msgf("Telemetry profile created: %s, %s", res.GetTelemetryProfile().GetResourceId(), res)
	return res, nil
}

func telemetryProfileCreator(in *telemetry_v1.TelemetryProfile) func(context.Context, *ent.Tx) (
	*inv_v1.Resource, error) {
	return func(ctx context.Context, tx *ent.Tx) (*inv_v1.Resource, error) {
		id := util.NewInvID(inv_v1.ResourceKind_RESOURCE_KIND_TELEMETRY_PROFILE)
		zlog.Debug().Msgf("CreateTelemetryProfile: %s", id)

		newEntity := tx.TelemetryProfile.Create()
		mut := newEntity.Mutation()

		if err := buildEntMutate(in, mut, TelemetryProfileEnumStatusMap, nil); err != nil {
			return nil, err
		}

		if err := setEdgeInstanceIDForMut(ctx, tx.Client(), mut, in.GetInstance()); err != nil {
			return nil, err
		}
		if err := setEdgeSiteIDForMut(ctx, tx.Client(), mut, in.GetSite()); err != nil {
			return nil, err
		}
		if err := setEdgeRegionIDForMut(ctx, tx.Client(), mut, in.GetRegion()); err != nil {
			return nil, err
		}
		if err := setEdgeTelemetryGroupIDForMut(ctx, tx.Client(), mut, in.GetGroup()); err != nil {
			return nil, err
		}

		if err := mut.SetField(telemetryprofileres.FieldResourceID, id); err != nil {
			return nil, errors.Wrap(err)
		}

		_, err := newEntity.Save(ctx)
		if err != nil {
			return nil, errors.Wrap(err)
		}

		res, err := getTelemetryProfileQuery(ctx, tx, id)
		if err != nil {
			return nil, err
		}
		if err := validateTelemetryProfileBeforeCommit(res); err != nil {
			return nil, err
		}
		return util.WrapResource(entTelemetryProfileToProtoTelemetryProfile(res))
	}
}

func (is *InvStore) GetTelemetryProfile(
	ctx context.Context, id string,
) (*inv_v1.Resource, error) {
	res, err := ExecuteInRoTxAndReturnSingle[ent.TelemetryProfile](is)(
		ctx,
		func(ctx context.Context, tx *ent.Tx) (*ent.TelemetryProfile, error) {
			return getTelemetryProfileQuery(ctx, tx, id)
		})
	if err != nil {
		return nil, err
	}

	apiResource := entTelemetryProfileToProtoTelemetryProfile(res)
	if err = validator.ValidateMessage(apiResource); err != nil {
		zlog.InfraSec().InfraErr(err).Msg("")
		return nil, errors.Wrap(err)
	}

	return &inv_v1.Resource{Resource: &inv_v1.Resource_TelemetryProfile{TelemetryProfile: apiResource}}, nil
}

func getTelemetryProfileQuery(ctx context.Context, tx *ent.Tx, resourceID string) (*ent.TelemetryProfile, error) {
	entity, err := tx.TelemetryProfile.Query().
		Where(telemetryprofileres.ResourceID(resourceID)).
		WithRegion().
		WithSite().
		WithInstance().
		WithGroup().
		Only(ctx)
	if err != nil {
		return nil, errors.Wrap(err)
	}
	return entity, nil
}

func (is *InvStore) UpdateTelemetryProfile(
	ctx context.Context, id string, in *telemetry_v1.TelemetryProfile, fieldmask *fieldmaskpb.FieldMask,
) (*inv_v1.Resource, error) {
	zlog.Debug().Msgf("UpdateTelemetryProfile (%s): %v, fm: %v", id, in, fieldmask)

	res, err := ExecuteInTxAndReturnSingle[inv_v1.Resource](is)(ctx,
		func(ctx context.Context, tx *ent.Tx) (*inv_v1.Resource, error) {
			entity, err := tx.TelemetryProfile.Query().
				Select(telemetryprofileres.FieldID).
				Where(telemetryprofileres.ResourceID(id)).
				Only(ctx)
			if err != nil {
				return nil, errors.Wrap(err)
			}

			updateBuilder := tx.TelemetryProfile.UpdateOneID(entity.ID)
			mut := updateBuilder.Mutation()

			err = setRelationsForTelemetryProfileMutIfNeeded(ctx, tx.Client(), mut, in)
			if err != nil {
				return nil, err
			}

			err = buildEntMutate(in, mut, TelemetryProfileEnumStatusMap, fieldmask.GetPaths())
			if err != nil {
				return nil, err
			}

			_, err = updateBuilder.Save(ctx)
			if err != nil {
				return nil, errors.Wrap(err)
			}

			res, err := getTelemetryProfileQuery(ctx, tx, id)
			if err != nil {
				return nil, err
			}

			if err := validateTelemetryProfileBeforeCommit(res); err != nil {
				return nil, err
			}
			return util.WrapResource(entTelemetryProfileToProtoTelemetryProfile(res))
		})
	if err != nil {
		return nil, err
	}

	return res, err
}

func setRelationsForTelemetryProfileMutIfNeeded(
	ctx context.Context, client *ent.Client, mut *ent.TelemetryProfileMutation, in *telemetry_v1.TelemetryProfile,
) error {
	// Given that Instance, Site and Region are mutually exclusive relations.
	// Setting one of them means that we need to clear the others.
	// This is not managed by the generic buildEntMutate, that only clears edges that are set to nil
	// but part of the fieldmask.
	if in.GetInstance() != nil {
		mut.ResetInstance()
		mut.ClearSite()
		mut.ClearRegion()
		if err := setEdgeInstanceIDForMut(ctx, client, mut, in.GetInstance()); err != nil {
			return err
		}
	}
	if in.GetSite() != nil {
		mut.ResetSite()
		mut.ClearInstance()
		mut.ClearRegion()
		if err := setEdgeSiteIDForMut(ctx, client, mut, in.GetSite()); err != nil {
			return err
		}
	}
	if in.GetRegion() != nil {
		mut.ResetRegion()
		mut.ClearInstance()
		mut.ClearSite()
		if err := setEdgeRegionIDForMut(ctx, client, mut, in.GetRegion()); err != nil {
			return err
		}
	}
	if in.GetGroup() != nil {
		mut.ResetGroup()
		if err := setEdgeTelemetryGroupIDForMut(ctx, client, mut, in.GetGroup()); err != nil {
			return err
		}
	}

	return nil
}

func (is *InvStore) DeleteTelemetryProfile(ctx context.Context, id string) (*inv_v1.Resource, error) {
	// this is a "Hard Delete" as telemetry profile don't have state to reconcile
	zlog.Debug().Msgf("DeleteTelemetryProfile Hard Delete: %s", id)

	res, err := ExecuteInTxAndReturnSingle[inv_v1.Resource](is)(ctx, deleteTelemetryProfile(id))

	return res, err
}

func deleteTelemetryProfile(id string) func(ctx context.Context, tx *ent.Tx) (*inv_v1.Resource, error) {
	return func(ctx context.Context, tx *ent.Tx) (*inv_v1.Resource, error) {
		res, err := tx.TelemetryProfile.Query().
			Where(telemetryprofileres.ResourceID(id)).
			Only(ctx)
		if err != nil {
			return nil, errors.Wrap(err)
		}

		err = tx.TelemetryProfile.DeleteOneID(res.ID).Exec(ctx)
		if err != nil {
			return nil, errors.Wrap(err)
		}

		return util.WrapResource(entTelemetryProfileToProtoTelemetryProfile(res))
	}
}

func (is *InvStore) DeleteTelemetryProfiles(
	ctx context.Context, tenantID string, _ bool,
) ([]*util.Tuple[DeletionKind, *inv_v1.Resource], error) {
	var deleted []*util.Tuple[DeletionKind, *inv_v1.Resource]
	txErr := ExecuteInTx(is)(
		ctx,
		func(ctx context.Context, tx *ent.Tx) error {
			collection, err := tx.TelemetryProfile.Query().Where(telemetryprofileres.TenantID(tenantID)).All(ctx)
			if err != nil {
				return err
			}
			if _, err := tx.TelemetryProfile.Delete().Where(telemetryprofileres.TenantID(tenantID)).Exec(ctx); err != nil {
				return err
			}
			for _, element := range collection {
				res, err := util.WrapResource(entTelemetryProfileToProtoTelemetryProfile(element))
				if err != nil {
					return err
				}
				deleted = append(deleted, util.NewTuple(HARD, res))
			}
			return nil
		})
	return deleted, txErr
}

func filterTelemetryProfile(ctx context.Context, client *ent.Client, filter *inv_v1.ResourceFilter) (
	[]*ent.TelemetryProfile,
	int,
	error,
) {
	pred, err := getPredicate(inv_v1.ResourceKind_RESOURCE_KIND_TELEMETRY_PROFILE, filter.GetFilter())
	if err != nil {
		return nil, 0, err
	}

	offset, limit, err := getOffsetAndLimit(filter)
	if err != nil {
		return nil, 0, err
	}

	return filterTelemetryProfileByPredicates(ctx, client, pred, filter.GetOrderBy(), offset, limit)
}

func filterTelemetryProfileByPredicates(
	ctx context.Context,
	client *ent.Client,
	pred predicate.TelemetryProfile,
	orderBy string,
	offset, limit int,
) ([]*ent.TelemetryProfile, int, error) {
	orderOpts, err := GetOrderByOptions[telemetryprofileres.OrderOption](orderBy, telemetryprofileres.ValidColumn)
	if err != nil {
		return nil, 0, err
	}

	// perform query - And together all the predicates
	query := client.TelemetryProfile.Query().
		WithInstance().
		WithSite().
		WithRegion().
		WithGroup().
		Where(pred).
		Order(orderOpts...).
		Offset(offset)

	// Limits number of query results if existent
	if limit != 0 {
		query = query.Limit(limit)
	}

	telProfileList, err := query.All(ctx)
	if err != nil {
		return nil, 0, errors.Wrap(err)
	}

	// Count total number of item without applying pagination limits, order, or loading edges.
	total, err := client.TelemetryProfile.Query().
		Where(pred).
		Count(ctx)
	if err != nil {
		return nil, 0, errors.Wrap(err)
	}

	return telProfileList, total, nil
}

func (is *InvStore) ListTelemetryProfile(
	ctx context.Context, filter *inv_v1.ResourceFilter,
) ([]*inv_v1.GetResourceResponse, int, error) {
	resources, total, err := ExecuteInRoTxAndReturnDouble[[]*ent.TelemetryProfile, int](is)(
		ctx,
		func(ctx context.Context, tx *ent.Tx) (*[]*ent.TelemetryProfile, *int, error) {
			resources, total, err := filterTelemetryProfile(ctx, tx.Client(), filter)
			if err != nil {
				return nil, nil, err
			}
			return &resources, &total, err
		},
	)
	if err != nil {
		return nil, 0, err
	}

	resps := collections.MapSlice[*ent.TelemetryProfile, *inv_v1.GetResourceResponse](*resources,
		func(res *ent.TelemetryProfile) *inv_v1.GetResourceResponse {
			return &inv_v1.GetResourceResponse{
				Resource: &inv_v1.Resource{
					Resource: &inv_v1.Resource_TelemetryProfile{
						TelemetryProfile: entTelemetryProfileToProtoTelemetryProfile(res),
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

func (is *InvStore) FilterTelemetryProfile(ctx context.Context, filter *inv_v1.ResourceFilter) (
	[]*cl.ResourceTenantIDCarrier, int, error,
) {
	resources, total, err := ExecuteInRoTxAndReturnDouble[[]*ent.TelemetryProfile, int](is)(
		ctx, func(ctx context.Context, tx *ent.Tx) (*[]*ent.TelemetryProfile, *int, error) {
			filtered, total, err := filterTelemetryProfile(ctx, tx.Client(), filter)
			if err != nil {
				return nil, nil, err
			}
			return &filtered, &total, nil
		})
	if err != nil {
		return nil, 0, err
	}

	ids := collections.MapSlice[*ent.TelemetryProfile, *cl.ResourceTenantIDCarrier](
		*resources, func(c *ent.TelemetryProfile) *cl.ResourceTenantIDCarrier {
			return &cl.ResourceTenantIDCarrier{TenantId: c.TenantID, ResourceId: c.ResourceID}
		})

	return ids, *total, err
}

func validateTelemetryProfileParameters(in *telemetry_v1.TelemetryProfile) error {
	switch in.GetKind() {
	case telemetry_v1.TelemetryResourceKind_TELEMETRY_RESOURCE_KIND_METRICS:
		if in.MetricsInterval == 0 {
			err := errors.Errorfc(codes.InvalidArgument,
				"Metrics interval must be set for TelemetryProfile of kind METRICS")
			zlog.InfraSec().InfraErr(err).Msg("")
			return err
		}
	case telemetry_v1.TelemetryResourceKind_TELEMETRY_RESOURCE_KIND_LOGS:
		if in.LogLevel == telemetry_v1.SeverityLevel_SEVERITY_LEVEL_UNSPECIFIED {
			err := errors.Errorfc(codes.InvalidArgument,
				"Log level must be set for TelemetryProfile of kind LOGS")
			zlog.InfraSec().InfraErr(err).Msg("")
			return err
		}
	case telemetry_v1.TelemetryResourceKind_TELEMETRY_RESOURCE_KIND_UNSPECIFIED:
		err := errors.Errorfc(codes.InvalidArgument,
			"Unsupported telemetry resource kind %v", in.Kind)
		zlog.InfraSec().InfraErr(err).Msg("")
		return err
	}
	return nil
}

func validateTelemetryProfileGroup(in *ent.TelemetryProfile) error {
	if in.Edges.Group == nil {
		err := errors.Errorfc(codes.InvalidArgument,
			"TelemetryGroupResource must be set for TelemetryProfile")
		zlog.InfraSec().InfraErr(err).Msg("")
		return err
	}

	if in.Edges.Group.ResourceID == "" {
		err := errors.Errorfc(codes.InvalidArgument,
			"TelemetryGroupResource ID must be set for TelemetryProfile")
		zlog.InfraSec().InfraErr(err).Msg("")
		return err
	}

	telemetryGroup := in.Edges.Group

	if in.Kind != telemetryprofileres.Kind(telemetryGroup.Kind) {
		err := errors.Errorfc(codes.InvalidArgument,
			"TelemetryProfile and TelemetryGroupResource should have the same kind, "+
				"got %v for TelemetryProfile and %v for TelemetryGroupResource",
			in.Kind,
			telemetryprofileres.Kind(telemetryGroup.Kind))
		zlog.InfraSec().InfraErr(err).Msg("")
		return err
	}

	return nil
}

// validateTelemetryProfileBeforeCommit checks if the updated TelemetryProfile is correct.
// In particular, it validates possible kind mismatch between TelemetryProfile and TelemetryGroup.
// The given telemetryProfile must have the Group eager loaded.
func validateTelemetryProfileBeforeCommit(telemetryProfile *ent.TelemetryProfile) error {
	if err := validateTelemetryProfileGroup(telemetryProfile); err != nil {
		return err
	}
	return validateTelemetryProfileParameters(entTelemetryProfileToProtoTelemetryProfile(telemetryProfile))
}

func (is *InvStore) ListInheritedTelemetryProfile(
	ctx context.Context, in *inv_v1.ListInheritedTelemetryProfilesRequest,
) ([]*telemetry_v1.TelemetryProfile, int, error) {
	resources, total, err := ExecuteInRoTxAndReturnDouble[[]*ent.TelemetryProfile, int](is)(
		ctx,
		func(ctx context.Context, tx *ent.Tx) (*[]*ent.TelemetryProfile, *int, error) {
			zeroInt := 0

			telemetryProfilesIDs, err := getInheritedTelemetry(ctx, tx.Client(), in.GetInheritBy(), in.GetTenantId())
			if err != nil {
				return nil, &zeroInt, err
			}

			// Shortcut to avoid wrong query below
			if len(telemetryProfilesIDs) == 0 {
				zlog.Debug().Msgf("no inherited telemetry profiles: request=%v", in)
				profilesToBeReturned := make([]*ent.TelemetryProfile, 0)
				return &profilesToBeReturned, &zeroInt, nil
			}

			filter := in.GetFilter()
			offset, limit, err := getOffsetAndLimit(filter)
			if err != nil {
				return nil, &zeroInt, err
			}

			// Preds will filter on both the IDs for the telemetry profiles we are interested into and the filter provided
			preds := []predicate.TelemetryProfile{
				telemetryprofileres.IDIn(telemetryProfilesIDs...),
			}
			pred, err := getPredicate(inv_v1.ResourceKind_RESOURCE_KIND_TELEMETRY_PROFILE, filter.GetFilter())
			if err != nil {
				return nil, &zeroInt, err
			}
			preds = append(preds, pred)

			entTelemetryProfiles, total, err := filterTelemetryProfileByPredicates(
				ctx, tx.Client(), telemetryprofileres.And(preds...), filter.GetOrderBy(), offset, limit)
			if err != nil {
				return nil, &zeroInt, err
			}

			return &entTelemetryProfiles, &total, nil
		},
	)
	if err != nil {
		return nil, 0, err
	}

	resps := collections.MapSlice[*ent.TelemetryProfile, *telemetry_v1.TelemetryProfile](*resources,
		entTelemetryProfileToProtoTelemetryProfile)

	return resps, *total, nil
}

// inheritBy should be already validated.
func getInheritedTelemetry(
	ctx context.Context,
	client *ent.Client,
	inheritBy *inv_v1.ListInheritedTelemetryProfilesRequest_InheritBy,
	tenantID string,
) ([]int, error) {
	// This query (CommonTableExpression) does a depth-first traversal of the Region hierarchy, while gathering also
	// linked Telemetry Profiles to the traversing region. The resulting table contains the enumeration of all possible
	// node to root paths in the tree, and the row will contain as column the ID and resource ID where the path starts
	// and the array of IDs of all the telemetry profiles gathered along the path.
	// TODO (ITEP-779): check for more optimized query
	regionHierarchyQuery := `
	WITH RECURSIVE region_hierarchy AS (
    -- start from root regions
    SELECT r.ID as curr_id, r.resource_id AS curr_res_id, r.region_resource_parent_region AS parent_id,
		(SELECT ARRAY_AGG(tp.ID)
         FROM telemetry_profiles AS tp 
         WHERE tp.telemetry_profile_region=r.ID AND tp.tenant_id=$2
        ) AS tps
    FROM region_resources AS r
    WHERE r.region_resource_parent_region IS NULL AND r.tenant_id=$2
    UNION ALL
    SELECT r.ID as curr_id, r.resource_id AS curr_res_id, r.region_resource_parent_region AS parent_id,
   	ARRAY_CAT(
		(SELECT ARRAY_AGG(tp.ID) 
         FROM telemetry_profiles AS tp 
 	     WHERE tp.telemetry_profile_region=r.ID AND tp.tenant_id=$2), rh.tps
        ) AS tps
    FROM region_resources AS r
    JOIN region_hierarchy AS rh ON rh.curr_id=r.region_resource_parent_region 
	WHERE r.tenant_id=$2
	)`
	// Here we render the result by a Site ID. We join the result from the region hierarchy with the parent region for the
	// given site, and prepend to the Telemetry Profile IDs the Telemetry Profiles IDs of the searched Site.
	// The result will be an array of inherited Telemetry Profiles IDs.
	bySiteIDQuery := regionHierarchyQuery + `
	SELECT ARRAY_CAT(
		(SELECT ARRAY_AGG(tp.ID) 
		 FROM telemetry_profiles AS tp 
		 WHERE tp.telemetry_profile_site = site.ID AND tp.tenant_id=$2), rh.tps
	     ) AS tps
	FROM site_resources AS site 
	LEFT JOIN region_hierarchy AS rh ON site.site_resource_region=rh.curr_id
	WHERE site.resource_id=$1 AND site.tenant_id=$2;`
	// Here filter only by the searched region exploiting the result from the region hierarchy
	byRegionIDQuery := regionHierarchyQuery + `
	SELECT tps
	FROM region_hierarchy AS rh
	WHERE rh.curr_res_id=$1;`
	// Here we append to the result coming from the region hierarchy, the telemetry profiles coming from the given instance,
	// and its respective Site (retrieved traversing the Host->Site relation). We render the result by Instance ID.
	// We join the Site associated to the instance with the region hierarchy. The Site is retrieved going via the
	// Instance->Host->Site relation. In the result we prepend the telemetry profile IDs associated to the Instance and Site.
	// The result will be an array of inherited Telemetry Profile IDs.
	byInstanceIDQuery := regionHierarchyQuery + `
	SELECT 
		ARRAY_CAT(
			ARRAY_CAT(
				(SELECT ARRAY_AGG(tp.ID) 
				 FROM telemetry_profiles AS tp 
				 WHERE tp.telemetry_profile_instance = inst.ID AND tp.tenant_id=$2), 
				(SELECT ARRAY_AGG(tp.ID) 
				 FROM telemetry_profiles AS tp 
				 WHERE tp.telemetry_profile_site = site.ID AND tp.tenant_id=$2)
			),
			rh.tps) AS tps
	FROM instance_resources AS inst
	LEFT JOIN host_resources AS host ON inst.ID=host.instance_resource_host AND host.tenant_id=$2 
	LEFT JOIN site_resources AS site ON host.host_resource_site=site.ID AND site.tenant_id=$2 
	LEFT JOIN region_resources AS region ON site.site_resource_region=region.ID AND region.tenant_id=$2
	LEFT JOIN region_hierarchy AS rh ON rh.curr_id=region.ID
	WHERE inst.resource_id=$1 AND inst.tenant_id=$2;`

	var query string
	// 	WHERE inst.resource_id=$1 AND inst.tenant_id=$2 AND host.tenant_id=$2 AND site.tenant_id=$2 AND region.tenant_id=$2;`
	var resourceID string
	// validate rules ensure that one of these is already set
	switch inheritBy.GetId().(type) {
	case *inv_v1.ListInheritedTelemetryProfilesRequest_InheritBy_InstanceId:
		query = byInstanceIDQuery
		resourceID = inheritBy.GetInstanceId()
	case *inv_v1.ListInheritedTelemetryProfilesRequest_InheritBy_SiteId:
		query = bySiteIDQuery
		resourceID = inheritBy.GetSiteId()
	case *inv_v1.ListInheritedTelemetryProfilesRequest_InheritBy_RegionId:
		query = byRegionIDQuery
		resourceID = inheritBy.GetRegionId()
	}
	return executeQuery(ctx, client, query, resourceID, tenantID)
}

func executeQuery(ctx context.Context, client *ent.Client, query string, resourceID, tenantID interface{}) (
	[]int, error,
) {
	rows, err := client.QueryContext(ctx, query, resourceID, tenantID)
	if err != nil {
		zlog.InfraSec().Err(err).Msg("error inheriting telemetry profiles")
		return nil, errors.Errorfc(codes.Internal, "error inheriting telemetry profiles")
	}
	var telemetryProfileIDs []int
	moreLines := false
	for rows.Next() {
		if moreLines {
			// We expect a single line in the result!
			if err = rows.Close(); err != nil {
				zlog.InfraSec().Err(err).Msgf("error while closing the result of a query")
			}
			zlog.InfraSec().Error().Msgf("error parsing results while inheriting telemetry profiles, more lines than expected")
			return nil, errors.Errorfc(codes.Internal,
				"error parsing results while inheriting telemetry profiles, more lines than expected")
		}
		var dbProfileIDs sql.NullString
		if err = rows.Scan(&dbProfileIDs); err != nil {
			zlog.InfraSec().Err(err).Msgf("error parsing results while inheriting telemetry profiles")
			return nil, errors.Errorfc(codes.Internal, "error parsing results while inheriting telemetry profiles")
		}
		telemetryProfileIDs, err = getIntSliceFromSQLVectorString(dbProfileIDs.String)
		if err != nil {
			return nil, err
		}
		moreLines = true
	}
	return telemetryProfileIDs, err
}

// Convert a string containing a SQL vector of integers "{X, Y, Z}" to a slice of integer.
func getIntSliceFromSQLVectorString(vectorInteger string) ([]int, error) {
	err := error(nil)
	strVectorInt := strings.TrimPrefix(vectorInteger, "{")
	strVectorInt = strings.TrimSuffix(strVectorInt, "}")
	sliceStrIntegers := strings.FieldsFunc(strVectorInt, func(r rune) bool {
		return r == ','
	})
	valueIntegers := make([]int, len(sliceStrIntegers))
	for i, pID := range sliceStrIntegers {
		var pIDInt int
		if pIDInt, err = strconv.Atoi(pID); err != nil {
			zlog.InfraSec().Err(err).Msgf("error while vector of integers from SQL output")
			return nil, errors.Errorfc(codes.Internal, "error while vector of integers from SQL output")
		}
		valueIntegers[i] = pIDInt
	}
	return valueIntegers, nil
}
