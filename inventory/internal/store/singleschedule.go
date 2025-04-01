// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package store

// singleschedule.go  store information for singleschedule objects

import (
	"context"
	"time"

	"golang.org/x/exp/slices"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	"github.com/open-edge-platform/infra-core/inventory/v2/internal/ent"
	ssr "github.com/open-edge-platform/infra-core/inventory/v2/internal/ent/singlescheduleresource"
	computev1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/compute/v1"
	inv_v1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/inventory/v1"
	locationv1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/location/v1"
	schedule_v1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/schedule/v1"
	cl "github.com/open-edge-platform/infra-core/inventory/v2/pkg/client"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/errors"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/util"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/util/collections"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/validator"
)

var singleScheduleCreationValidators = []resourceValidator[*schedule_v1.SingleScheduleResource]{
	protoValidator[*schedule_v1.SingleScheduleResource],
	validateSScheduledInput,
	doNotAcceptResourceID[*schedule_v1.SingleScheduleResource],
}

// enum status mapping.
func SingleScheduleEnumStatusMap(fname string, eint int32) (ent.Value, error) {
	switch fname {
	case ssr.FieldScheduleStatus:
		return ssr.ScheduleStatus(schedule_v1.ScheduleStatus_name[eint]), nil
	default:
		zlog.InfraSec().InfraError("unknown Enum field %s", fname).Msg("")
		return nil, errors.Errorfc(codes.InvalidArgument, "unknown Enum field %s", fname)
	}
}

func (is *InvStore) CreateSingleSchedule(ctx context.Context, in *schedule_v1.SingleScheduleResource) (*inv_v1.Resource, error) {
	if err := validate(in, singleScheduleCreationValidators...); err != nil {
		return nil, err
	}

	// disallow create of StartSeconds before current time
	now := uint64(time.Now().Unix()) //nolint:gosec // no overflow for a few billion years
	if in.StartSeconds <= now {
		zlog.InfraSec().InfraError("start %d cannot be earlier than current time %d", in.StartSeconds, now).Msg("")
		return nil, errors.Errorfc(codes.InvalidArgument, "Scheduled start time cannot be in the past")
	}

	res, err := ExecuteInTxAndReturnSingle[inv_v1.Resource](is)(ctx, singleScheduleCreator(in))
	if err != nil {
		return nil, err
	}

	zlog.Debug().Msgf("SingleSchedule Created: %s, %s", res.GetSingleschedule().GetResourceId(), res)
	return res, nil
}

func singleScheduleCreator(in *schedule_v1.SingleScheduleResource) func(context.Context, *ent.Tx) (
	*inv_v1.Resource, error) {
	return func(ctx context.Context, tx *ent.Tx) (*inv_v1.Resource, error) {
		id := util.NewInvID(inv_v1.ResourceKind_RESOURCE_KIND_SINGLESCHEDULE)
		zlog.Debug().Msgf("CreateSingleSchedule: %s", id)

		newEntity := tx.SingleScheduleResource.Create()
		mut := newEntity.Mutation()

		if err := buildEntMutate(in, mut, SingleScheduleEnumStatusMap, nil); err != nil {
			return nil, err
		}

		// Look up the optional site ID for this single schedule.
		if err := setEdgeSiteIDForMut(ctx, tx.Client(), mut, in.GetTargetSite()); err != nil {
			return nil, err
		}

		// Look up the optional host ID for this single schedule.
		if err := setEdgeHostIDForMut(ctx, tx.Client(), mut, in.GetTargetHost()); err != nil {
			return nil, err
		}

		// Look up the optional workload ID for this single schedule.
		if err := setEdgeWorkloadIDForMut(ctx, tx.Client(), mut, in.GetTargetWorkload()); err != nil {
			return nil, err
		}

		// Look up the optional region ID for this single schedule.
		if err := setEdgeRegionIDForMut(ctx, tx.Client(), mut, in.GetTargetRegion()); err != nil {
			return nil, err
		}

		// Set the resource_id field last.
		if err := mut.SetField(ssr.FieldResourceID, id); err != nil {
			return nil, errors.Wrap(err)
		}

		_, err := newEntity.Save(ctx)
		if err != nil {
			return nil, errors.Wrap(err)
		}

		res, err := getSingleScheduleQuery(ctx, tx, id)
		if err != nil {
			return nil, err
		}
		return util.WrapResource(entSingleScheduleResourceToProtoSingleScheduleResource(res))
	}
}

func (is *InvStore) GetSingleSchedule(ctx context.Context, id string) (*inv_v1.Resource, error) {
	res, err := ExecuteInRoTxAndReturnSingle[ent.SingleScheduleResource](is)(
		ctx,
		func(ctx context.Context, tx *ent.Tx) (*ent.SingleScheduleResource, error) {
			return getSingleScheduleQuery(ctx, tx, id)
		})
	if err != nil {
		return nil, err
	}

	apiResource := entSingleScheduleResourceToProtoSingleScheduleResource(res)
	if err = validator.ValidateMessage(apiResource); err != nil {
		zlog.InfraSec().InfraErr(err).Msg("")
		return nil, errors.Wrap(err)
	}

	return &inv_v1.Resource{Resource: &inv_v1.Resource_Singleschedule{Singleschedule: apiResource}}, nil
}

func getSingleScheduleQuery(ctx context.Context, tx *ent.Tx, resourceID string) (*ent.SingleScheduleResource, error) {
	entity, err := tx.SingleScheduleResource.Query().
		Where(ssr.ResourceID(resourceID)).
		WithTargetHost().
		WithTargetSite().
		WithTargetRegion().
		WithTargetWorkload().
		Only(ctx)
	if err != nil {
		return nil, errors.Wrap(err)
	}
	return entity, nil
}

//nolint:cyclop // high cyclomatic complexity (11) due to validity checks and transaction
func (is *InvStore) UpdateSingleSchedule(
	ctx context.Context,
	id string,
	in *schedule_v1.SingleScheduleResource,
	fieldmask *fieldmaskpb.FieldMask,
) (*inv_v1.Resource, error) {
	if err := validateSScheduledInput(in); err != nil {
		return nil, err
	}

	// disallow update of StartSeconds before current time, if StartSeconds is set using fieldmask
	now := uint64(time.Now().Unix()) //nolint:gosec // no overflow for a few billion years
	if slices.Contains(fieldmask.GetPaths(), ssr.FieldStartSeconds) && in.StartSeconds <= now {
		zlog.InfraSec().InfraError("start %d cannot be earlier than current time %d", in.StartSeconds, now).Msg("")
		return nil, errors.Errorfc(codes.InvalidArgument, "Scheduled start time cannot be in the past")
	}

	zlog.Debug().Msgf("UpdateSingleSchedule (%s): %v, fm: %v", id, in, fieldmask)

	res, err := ExecuteInTxAndReturnSingle[inv_v1.Resource](is)(ctx,
		func(ctx context.Context, tx *ent.Tx) (*inv_v1.Resource, error) {
			entity, err := tx.SingleScheduleResource.Query().
				Select(ssr.FieldID).
				Where(ssr.ResourceID(id)).
				Only(ctx)
			if err != nil {
				return nil, errors.Wrap(err)
			}

			updateBuilder := tx.SingleScheduleResource.UpdateOneID(entity.ID)
			mut := updateBuilder.Mutation()

			// Look up the (new) referenced targets.
			err = is.setRelationsForSScheduleMutIfNeeded(ctx, mut, tx, in, fieldmask)
			if err != nil {
				return nil, err
			}

			err = buildEntMutate(in, mut, SingleScheduleEnumStatusMap, fieldmask.GetPaths())
			if err != nil {
				return nil, err
			}

			_, err = updateBuilder.Save(ctx)
			if err != nil {
				return nil, errors.Wrap(err)
			}

			res, err := getSingleScheduleQuery(ctx, tx, id)
			if err != nil {
				return nil, err
			}
			// Enforce the target presence (cannot be set both)
			if err := is.checkSScheduleUpdate(res); err != nil {
				return nil, err
			}
			return util.WrapResource(entSingleScheduleResourceToProtoSingleScheduleResource(res))
		})
	if err != nil {
		return nil, err
	}

	return res, err
}

func (is *InvStore) DeleteSingleSchedule(ctx context.Context, id string) (*inv_v1.Resource, error) {
	// this is a "Hard Delete" as singleSchedule don't have state to reconcile
	zlog.Debug().Msgf("DeleteSingleSchedule Hard Delete: %s", id)

	res, err := ExecuteInTxAndReturnSingle[inv_v1.Resource](is)(ctx, deleteSingleSchedule(id))
	return res, err
}

func deleteSingleSchedule(id string) func(ctx context.Context, tx *ent.Tx) (*inv_v1.Resource, error) {
	return func(ctx context.Context, tx *ent.Tx) (*inv_v1.Resource, error) {
		entity, err := tx.SingleScheduleResource.Query().
			Where(ssr.ResourceID(id)).
			Only(ctx)
		if err != nil {
			return nil, errors.Wrap(err)
		}

		err = tx.SingleScheduleResource.DeleteOneID(entity.ID).Exec(ctx)
		if err != nil {
			return nil, errors.Wrap(err)
		}

		return util.WrapResource(entSingleScheduleResourceToProtoSingleScheduleResource(entity))
	}
}

func (is *InvStore) DeleteSingleSchedules(
	ctx context.Context, tenantID string, _ bool,
) ([]*util.Tuple[DeletionKind, *inv_v1.Resource], error) {
	var deleted []*util.Tuple[DeletionKind, *inv_v1.Resource]
	txErr := ExecuteInTx(is)(
		ctx,
		func(ctx context.Context, tx *ent.Tx) error {
			all, err := tx.SingleScheduleResource.Query().Where(ssr.TenantID(tenantID)).All(ctx)
			if err != nil {
				return err
			}
			if _, err := tx.SingleScheduleResource.Delete().Where(ssr.TenantID(tenantID)).Exec(ctx); err != nil {
				return err
			}
			for _, element := range all {
				res, err := util.WrapResource(entSingleScheduleResourceToProtoSingleScheduleResource(element))
				if err != nil {
					return err
				}
				deleted = append(deleted, util.NewTuple(HARD, res))
			}
			return nil
		})
	return deleted, txErr
}

func filterSingleSchedule(ctx context.Context, client *ent.Client, filter *inv_v1.ResourceFilter) (
	[]*ent.SingleScheduleResource,
	int,
	error,
) {
	pred, err := getPredicate(inv_v1.ResourceKind_RESOURCE_KIND_SINGLESCHEDULE, filter.GetFilter())
	if err != nil {
		return nil, 0, err
	}

	orderOpts, err := GetOrderByOptions[ssr.OrderOption](filter.GetOrderBy(), ssr.ValidColumn)
	if err != nil {
		return nil, 0, err
	}

	offset, limit, err := getOffsetAndLimit(filter)
	if err != nil {
		return nil, 0, err
	}

	// perform query - And together all the predicates
	query := client.SingleScheduleResource.Query().
		WithTargetSite().
		WithTargetRegion().
		WithTargetHost().
		WithTargetWorkload().
		Where(pred).
		Order(orderOpts...).
		Offset(offset)

	// Limits number of query results if existent
	if limit != 0 {
		query = query.Limit(limit)
	}

	singleScheduleList, err := query.All(ctx)
	if err != nil {
		return nil, 0, errors.Wrap(err)
	}

	// Count total number of item without applying pagination limits, order, or loading edges.
	total, err := client.SingleScheduleResource.Query().
		Where(pred).
		Count(ctx)
	if err != nil {
		return nil, 0, errors.Wrap(err)
	}

	return singleScheduleList, total, nil
}

func (is *InvStore) ListSingleSchedules(
	ctx context.Context, filter *inv_v1.ResourceFilter,
) ([]*inv_v1.GetResourceResponse, int, error) {
	resources, total, err := ExecuteInRoTxAndReturnDouble[[]*ent.SingleScheduleResource, int](is)(
		ctx,
		func(ctx context.Context, tx *ent.Tx) (*[]*ent.SingleScheduleResource, *int, error) {
			resources, total, err := filterSingleSchedule(ctx, tx.Client(), filter)
			if err != nil {
				return nil, nil, err
			}
			return &resources, &total, err
		},
	)
	if err != nil {
		return nil, 0, err
	}

	resps := collections.MapSlice[*ent.SingleScheduleResource, *inv_v1.GetResourceResponse](*resources,
		func(res *ent.SingleScheduleResource) *inv_v1.GetResourceResponse {
			return &inv_v1.GetResourceResponse{
				Resource: &inv_v1.Resource{
					Resource: &inv_v1.Resource_Singleschedule{
						Singleschedule: entSingleScheduleResourceToProtoSingleScheduleResource(res),
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

func (is *InvStore) FilterSingleSchedules(ctx context.Context, filter *inv_v1.ResourceFilter) (
	[]*cl.ResourceTenantIDCarrier, int, error,
) {
	resources, total, err := ExecuteInRoTxAndReturnDouble[[]*ent.SingleScheduleResource, int](is)(
		ctx, func(ctx context.Context, tx *ent.Tx) (*[]*ent.SingleScheduleResource, *int, error) {
			filtered, total, err := filterSingleSchedule(ctx, tx.Client(), filter)
			if err != nil {
				return nil, nil, err
			}
			return &filtered, &total, nil
		})
	if err != nil {
		return nil, 0, err
	}

	ids := collections.MapSlice[*ent.SingleScheduleResource, *cl.ResourceTenantIDCarrier](
		*resources, func(c *ent.SingleScheduleResource) *cl.ResourceTenantIDCarrier {
			return &cl.ResourceTenantIDCarrier{TenantId: c.TenantID, ResourceId: c.ResourceID}
		})

	return ids, *total, err
}

func (is *InvStore) setTargetHostForSScheduleMut(
	ctx context.Context, mut *ent.SingleScheduleResourceMutation, tx *ent.Tx, hosres *computev1.HostResource,
) error {
	if hosres != nil {
		hosID, qerr := getHostIDFromResourceID(ctx, tx.Client(), hosres)
		if qerr != nil {
			return qerr
		}
		mut.SetTargetHostID(hosID)
	}
	return nil
}

func (is *InvStore) setTargetSiteForSScheduleMut(
	ctx context.Context, mut *ent.SingleScheduleResourceMutation, tx *ent.Tx, sitres *locationv1.SiteResource,
) error {
	if sitres != nil {
		sitID, qerr := getSiteIDFromResourceID(ctx, tx.Client(), sitres)
		if qerr != nil {
			return qerr
		}
		mut.SetTargetSiteID(sitID)
	}
	return nil
}

func (is *InvStore) setTargetRegionForSScheduleMut(
	ctx context.Context, mut *ent.SingleScheduleResourceMutation, tx *ent.Tx, reg *locationv1.RegionResource,
) error {
	if reg != nil {
		id, qerr := getRegionIDFromResourceID(ctx, tx.Client(), reg)
		if qerr != nil {
			return qerr
		}
		mut.SetTargetRegionID(id)
	}
	return nil
}

func (is *InvStore) setTargetWorkloadForSScheduleMut(
	ctx context.Context, mut *ent.SingleScheduleResourceMutation, tx *ent.Tx, workloadres *computev1.WorkloadResource,
) error {
	if workloadres != nil {
		workID, qerr := getWorkloadIDFromResourceID(ctx, tx.Client(), workloadres)
		if qerr != nil {
			return qerr
		}
		mut.SetTargetWorkloadID(workID)
	}
	return nil
}

func (is *InvStore) setRelationsForSScheduleMutIfNeeded(
	ctx context.Context,
	mut *ent.SingleScheduleResourceMutation,
	tx *ent.Tx,
	in *schedule_v1.SingleScheduleResource,
	fieldmask *fieldmaskpb.FieldMask,
) error {
	mut.ResetTargetHost()
	if slices.Contains(fieldmask.GetPaths(), ssr.EdgeTargetHost) {
		if err := is.setTargetHostForSScheduleMut(ctx, mut, tx, in.GetTargetHost()); err != nil {
			return err
		}
	}
	mut.ResetTargetRegion()
	if slices.Contains(fieldmask.GetPaths(), ssr.EdgeTargetRegion) {
		if err := is.setTargetRegionForSScheduleMut(ctx, mut, tx, in.GetTargetRegion()); err != nil {
			return err
		}
	}
	mut.ResetTargetSite()
	if slices.Contains(fieldmask.GetPaths(), ssr.EdgeTargetSite) {
		if err := is.setTargetSiteForSScheduleMut(ctx, mut, tx, in.GetTargetSite()); err != nil {
			return err
		}
	}
	mut.ResetTargetWorkload()
	if slices.Contains(fieldmask.GetPaths(), ssr.EdgeTargetWorkload) {
		if err := is.setTargetWorkloadForSScheduleMut(ctx, mut, tx, in.GetTargetWorkload()); err != nil {
			return err
		}
	}
	return nil
}

// Verify that both target host and target site are not set.
// Checks on start and end as well.
// The given Single Schedule must have eager loaded edges.
func (is *InvStore) checkSScheduleUpdate(ssched *ent.SingleScheduleResource) error {
	setCount := 0
	if ssched.Edges.TargetHost != nil {
		setCount++
	}
	if ssched.Edges.TargetSite != nil {
		setCount++
	}
	if ssched.Edges.TargetWorkload != nil {
		setCount++
	}
	if ssched.Edges.TargetRegion != nil {
		setCount++
	}
	if setCount > 1 {
		zlog.InfraSec().InfraError("more than one target cannot be set at the same time").Msg("")
		return errors.Errorfc(codes.InvalidArgument,
			"single sched resource %v has more than one target",
			ssched.ResourceID)
	}

	if ssched.EndSeconds != 0 && ssched.EndSeconds <= ssched.StartSeconds {
		zlog.InfraSec().InfraError("end cannot be <= than start seconds").Msg("")
		return errors.Errorfc(codes.InvalidArgument, "end cannot be <= than start seconds")
	}

	return nil
}

func validateSScheduledInput(in *schedule_v1.SingleScheduleResource) error {
	if in.EndSeconds != 0 && in.EndSeconds <= in.StartSeconds {
		zlog.InfraSec().InfraError("end cannot be <= than start seconds").Msg("")
		return errors.Errorfc(codes.InvalidArgument, "end cannot be <= than start seconds")
	}
	return nil
}
