// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package store

// repeatedschedule.go  store information for repeatedschedule objects

import (
	"context"

	"golang.org/x/exp/slices"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	"github.com/open-edge-platform/infra-core/inventory/v2/internal/ent"
	rsr "github.com/open-edge-platform/infra-core/inventory/v2/internal/ent/repeatedscheduleresource"
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

var repeatedScheduleCreationValidators = []resourceValidator[*schedule_v1.RepeatedScheduleResource]{
	protoValidator[*schedule_v1.RepeatedScheduleResource],
	validateRScheduledInput,
	doNotAcceptResourceID[*schedule_v1.RepeatedScheduleResource],
}

// enum status mapping.
func RepeatedScheduleEnumStatusMap(fname string, eint int32) (ent.Value, error) {
	switch fname {
	case rsr.FieldScheduleStatus:
		return rsr.ScheduleStatus(schedule_v1.ScheduleStatus_name[eint]), nil
	default:
		zlog.InfraSec().InfraError("unknown Enum field %s", fname).Msg("")
		return nil, errors.Errorfc(codes.InvalidArgument, "unknown Enum field %s", fname)
	}
}

func (is *InvStore) CreateRepeatedSchedule(
	ctx context.Context,
	in *schedule_v1.RepeatedScheduleResource,
) (*inv_v1.Resource, error) {
	if err := validate(in, repeatedScheduleCreationValidators...); err != nil {
		return nil, err
	}

	res, err := ExecuteInTxAndReturnSingle[inv_v1.Resource](is)(ctx, repeatedScheduleCreator(in))
	if err != nil {
		return nil, err
	}

	zlog.Debug().Msgf("RepeatedSchedule Created: %s, %s", res.GetRepeatedschedule().GetResourceId(), res)

	return res, nil
}

func repeatedScheduleCreator(in *schedule_v1.RepeatedScheduleResource) func(context.Context, *ent.Tx) (
	*inv_v1.Resource, error) {
	return func(ctx context.Context, tx *ent.Tx) (*inv_v1.Resource, error) {
		id := util.NewInvID(inv_v1.ResourceKind_RESOURCE_KIND_REPEATEDSCHEDULE)
		zlog.Debug().Msgf("CreateRepeatedSchedule: %s", id)

		newEntity := tx.RepeatedScheduleResource.Create()
		mut := newEntity.Mutation()

		if err := buildEntMutate(in, mut, RepeatedScheduleEnumStatusMap, nil); err != nil {
			return nil, err
		}

		// Look up the optional site ID for this single schedule.
		if err := setEdgeRegionIDForMut(ctx, tx.Client(), mut, in.GetTargetRegion()); err != nil {
			return nil, err
		}

		if err := setEdgeSiteIDForMut(ctx, tx.Client(), mut, in.GetTargetSite()); err != nil {
			return nil, err
		}

		// Look up the optional site ID for this single schedule.
		if err := setEdgeHostIDForMut(ctx, tx.Client(), mut, in.GetTargetHost()); err != nil {
			return nil, err
		}

		// Look up the optional workload ID for this single schedule.
		if err := setEdgeWorkloadIDForMut(ctx, tx.Client(), mut, in.GetTargetWorkload()); err != nil {
			return nil, err
		}

		if err := mut.SetField(rsr.FieldResourceID, id); err != nil {
			return nil, errors.Wrap(err)
		}

		// save to persistence
		_, err := newEntity.Save(ctx)
		if err != nil {
			return nil, errors.Wrap(err)
		}

		res, err := getRepeatedScheduleQuery(ctx, tx, id)
		if err != nil {
			return nil, err
		}
		return util.WrapResource(entRepeatedScheduleResourceToProtoRepeatedScheduleResource(res))
	}
}

func (is *InvStore) GetRepeatedSchedule(ctx context.Context, id string) (*inv_v1.Resource, error) {
	res, err := ExecuteInRoTxAndReturnSingle[ent.RepeatedScheduleResource](is)(
		ctx,
		func(ctx context.Context, tx *ent.Tx) (*ent.RepeatedScheduleResource, error) {
			return getRepeatedScheduleQuery(ctx, tx, id)
		})
	if err != nil {
		return nil, err
	}

	apiResource := entRepeatedScheduleResourceToProtoRepeatedScheduleResource(res)
	if err = validator.ValidateMessage(apiResource); err != nil {
		zlog.InfraSec().InfraErr(err).Msg("")
		return nil, errors.Wrap(err)
	}

	return &inv_v1.Resource{Resource: &inv_v1.Resource_Repeatedschedule{Repeatedschedule: apiResource}}, nil
}

func getRepeatedScheduleQuery(ctx context.Context, tx *ent.Tx, resourceID string) (*ent.RepeatedScheduleResource, error) {
	entity, err := tx.RepeatedScheduleResource.Query().
		Where(rsr.ResourceID(resourceID)).
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

func (is *InvStore) UpdateRepeatedSchedule(
	ctx context.Context,
	id string,
	in *schedule_v1.RepeatedScheduleResource,
	fieldmask *fieldmaskpb.FieldMask,
) (*inv_v1.Resource, error) {
	if err := validateRScheduledInput(in); err != nil {
		return nil, err
	}
	zlog.Debug().Msgf("UpdateRepeatedSchedule (%s): %v, fm: %v", id, in, fieldmask)

	res, err := ExecuteInTxAndReturnSingle[inv_v1.Resource](is)(ctx,
		func(ctx context.Context, tx *ent.Tx) (*inv_v1.Resource, error) {
			entity, err := tx.RepeatedScheduleResource.Query().
				Select(rsr.FieldID).
				Where(rsr.ResourceID(id)).
				Only(ctx)
			if err != nil {
				return nil, errors.Wrap(err)
			}

			updateBuilder := tx.RepeatedScheduleResource.UpdateOneID(entity.ID)
			mut := updateBuilder.Mutation()

			// Look up the (new) referenced targets.
			err = is.setRelationsForRScheduleMutIfNeeded(ctx, mut, tx, in, fieldmask)
			if err != nil {
				return nil, err
			}

			err = buildEntMutate(in, mut, RepeatedScheduleEnumStatusMap, fieldmask.GetPaths())
			if err != nil {
				return nil, err
			}

			_, err = updateBuilder.Save(ctx)
			if err != nil {
				return nil, errors.Wrap(err)
			}

			res, err := getRepeatedScheduleQuery(ctx, tx, id)
			if err != nil {
				return nil, err
			}
			// Enforce the target presence (cannot be set both)
			if err := is.checkRScheduleTargets(res); err != nil {
				return nil, err
			}
			return util.WrapResource(entRepeatedScheduleResourceToProtoRepeatedScheduleResource(res))
		})
	if err != nil {
		return nil, err
	}

	return res, err
}

func (is *InvStore) DeleteRepeatedSchedule(ctx context.Context, id string) (*inv_v1.Resource, error) {
	// this is a "Hard Delete" as repeatedSchedule don't have state

	// FIXME - it should be impossible to delete a update that has other
	// dependent resources that are owned by the repeatedSchedule
	zlog.Debug().Msgf("DeleteRepeatedSchedule Hard Delete: %s", id)

	res, err := ExecuteInTxAndReturnSingle[inv_v1.Resource](is)(
		ctx,
		deleteRepeatedSchedule(id))

	return res, err
}

func deleteRepeatedSchedule(id string) func(ctx context.Context, tx *ent.Tx) (*inv_v1.Resource, error) {
	return func(ctx context.Context, tx *ent.Tx) (*inv_v1.Resource, error) {
		entity, err := tx.RepeatedScheduleResource.Query().
			Where(rsr.ResourceID(id)).
			Only(ctx)
		if err != nil {
			return nil, errors.Wrap(err)
		}

		err = tx.RepeatedScheduleResource.DeleteOneID(entity.ID).Exec(ctx)
		if err != nil {
			return nil, errors.Wrap(err)
		}

		return util.WrapResource(entRepeatedScheduleResourceToProtoRepeatedScheduleResource(entity))
	}
}

func (is *InvStore) DeleteRepeatedSchedules(
	ctx context.Context, tenantID string, _ bool,
) ([]*util.Tuple[DeletionKind, *inv_v1.Resource], error) {
	var deleted []*util.Tuple[DeletionKind, *inv_v1.Resource]
	txErr := ExecuteInTx(is)(
		ctx,
		func(ctx context.Context, tx *ent.Tx) error {
			all, err := tx.RepeatedScheduleResource.Query().Where(rsr.TenantID(tenantID)).All(ctx)
			if err != nil {
				return err
			}
			if _, err := tx.RepeatedScheduleResource.Delete().Where(rsr.TenantID(tenantID)).Exec(ctx); err != nil {
				return err
			}
			for _, element := range all {
				res, err := util.WrapResource(entRepeatedScheduleResourceToProtoRepeatedScheduleResource(element))
				if err != nil {
					return err
				}
				deleted = append(deleted, util.NewTuple(HARD, res))
			}
			return nil
		})
	return deleted, txErr
}

func filterRepeatedSchedules(ctx context.Context, client *ent.Client, filter *inv_v1.ResourceFilter) (
	[]*ent.RepeatedScheduleResource,
	int,
	error,
) {
	pred, err := getPredicate(inv_v1.ResourceKind_RESOURCE_KIND_REPEATEDSCHEDULE, filter.GetFilter())
	if err != nil {
		return nil, 0, err
	}

	orderOpts, err := GetOrderByOptions[rsr.OrderOption](filter.GetOrderBy(), rsr.ValidColumn)
	if err != nil {
		return nil, 0, err
	}

	offset, limit, err := getOffsetAndLimit(filter)
	if err != nil {
		return nil, 0, err
	}

	// perform query - And together all the predicates
	query := client.RepeatedScheduleResource.Query().
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

	repeatedScheduleList, err := query.All(ctx)
	if err != nil {
		return nil, 0, errors.Wrap(err)
	}

	// Count total number of item without applying pagination limits, order, or loading edges.
	total, err := client.RepeatedScheduleResource.Query().
		Where(pred).
		Count(ctx)
	if err != nil {
		return nil, 0, errors.Wrap(err)
	}

	return repeatedScheduleList, total, nil
}

func (is *InvStore) ListRepeatedSchedules(
	ctx context.Context, filter *inv_v1.ResourceFilter,
) ([]*inv_v1.GetResourceResponse, int, error) {
	resources, total, err := ExecuteInRoTxAndReturnDouble[[]*ent.RepeatedScheduleResource, int](is)(
		ctx,
		func(ctx context.Context, tx *ent.Tx) (*[]*ent.RepeatedScheduleResource, *int, error) {
			resources, total, err := filterRepeatedSchedules(ctx, tx.Client(), filter)
			if err != nil {
				return nil, nil, err
			}
			return &resources, &total, err
		},
	)
	if err != nil {
		return nil, 0, err
	}

	resps := collections.MapSlice[*ent.RepeatedScheduleResource, *inv_v1.GetResourceResponse](*resources,
		func(res *ent.RepeatedScheduleResource) *inv_v1.GetResourceResponse {
			return &inv_v1.GetResourceResponse{
				Resource: &inv_v1.Resource{
					Resource: &inv_v1.Resource_Repeatedschedule{
						Repeatedschedule: entRepeatedScheduleResourceToProtoRepeatedScheduleResource(res),
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

func (is *InvStore) FilterRepeatedSchedules(ctx context.Context, filter *inv_v1.ResourceFilter) (
	[]*cl.ResourceTenantIDCarrier, int, error,
) {
	resources, total, err := ExecuteInRoTxAndReturnDouble[[]*ent.RepeatedScheduleResource, int](is)(
		ctx, func(ctx context.Context, tx *ent.Tx) (*[]*ent.RepeatedScheduleResource, *int, error) {
			filtered, total, err := filterRepeatedSchedules(ctx, tx.Client(), filter)
			if err != nil {
				return nil, nil, err
			}
			return &filtered, &total, nil
		})
	if err != nil {
		return nil, 0, err
	}

	ids := collections.MapSlice[*ent.RepeatedScheduleResource, *cl.ResourceTenantIDCarrier](
		*resources, func(c *ent.RepeatedScheduleResource) *cl.ResourceTenantIDCarrier {
			return &cl.ResourceTenantIDCarrier{TenantId: c.TenantID, ResourceId: c.ResourceID}
		})

	return ids, *total, err
}

func (is *InvStore) setTargetHostForRScheduleMut(
	ctx context.Context, mut *ent.RepeatedScheduleResourceMutation, tx *ent.Tx, hosres *computev1.HostResource,
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

func (is *InvStore) setTargetSiteForRScheduleMut(
	ctx context.Context, mut *ent.RepeatedScheduleResourceMutation, tx *ent.Tx, sitres *locationv1.SiteResource,
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

func (is *InvStore) setTargetRegionForRScheduleMut(
	ctx context.Context, mut *ent.RepeatedScheduleResourceMutation, tx *ent.Tx, resource *locationv1.RegionResource,
) error {
	if resource != nil {
		id, qerr := getRegionIDFromResourceID(ctx, tx.Client(), resource)
		if qerr != nil {
			return qerr
		}
		mut.SetTargetRegionID(id)
	}
	return nil
}

func (is *InvStore) setTargetWorkloadForRScheduleMut(
	ctx context.Context, mut *ent.RepeatedScheduleResourceMutation, tx *ent.Tx, workloadres *computev1.WorkloadResource,
) error {
	if workloadres != nil {
		sitID, qerr := getWorkloadIDFromResourceID(ctx, tx.Client(), workloadres)
		if qerr != nil {
			return qerr
		}
		mut.SetTargetWorkloadID(sitID)
	}
	return nil
}

func (is *InvStore) setRelationsForRScheduleMutIfNeeded(
	ctx context.Context, mut *ent.RepeatedScheduleResourceMutation, tx *ent.Tx,
	in *schedule_v1.RepeatedScheduleResource, fieldmask *fieldmaskpb.FieldMask,
) error {
	mut.ResetTargetHost()
	if slices.Contains(fieldmask.GetPaths(), rsr.EdgeTargetHost) {
		if err := is.setTargetHostForRScheduleMut(ctx, mut, tx, in.GetTargetHost()); err != nil {
			return err
		}
	}
	mut.ResetTargetRegion()
	if slices.Contains(fieldmask.GetPaths(), rsr.EdgeTargetRegion) {
		if err := is.setTargetRegionForRScheduleMut(ctx, mut, tx, in.GetTargetRegion()); err != nil {
			return err
		}
	}
	mut.ResetTargetSite()
	if slices.Contains(fieldmask.GetPaths(), rsr.EdgeTargetSite) {
		if err := is.setTargetSiteForRScheduleMut(ctx, mut, tx, in.GetTargetSite()); err != nil {
			return err
		}
	}
	mut.ResetTargetWorkload()
	if slices.Contains(fieldmask.GetPaths(), rsr.EdgeTargetWorkload) {
		if err := is.setTargetWorkloadForRScheduleMut(ctx, mut, tx, in.GetTargetWorkload()); err != nil {
			return err
		}
	}
	return nil
}

// Verify that both target host and target site are not set.
// The provided Repeated Schedule must have eager loaded edges.
func (is *InvStore) checkRScheduleTargets(rsched *ent.RepeatedScheduleResource) error {
	setCount := 0
	if rsched.Edges.TargetHost != nil {
		setCount++
	}
	if rsched.Edges.TargetSite != nil {
		setCount++
	}
	if rsched.Edges.TargetWorkload != nil {
		setCount++
	}
	if rsched.Edges.TargetRegion != nil {
		setCount++
	}
	if setCount > 1 {
		zlog.InfraSec().InfraError("more than one target cannot be set at the same time").Msg("")
		return errors.Errorfc(codes.InvalidArgument,
			"repeated sched resource %v has more than one target set",
			rsched.ResourceID)
	}
	return nil
}

func validateCronFields(in *schedule_v1.RepeatedScheduleResource) error {
	if in.GetCronMinutes() == "" ||
		in.GetCronHours() == "" ||
		in.GetCronMonth() == "" ||
		in.GetCronDayWeek() == "" ||
		in.GetCronDayMonth() == "" {
		zlog.InfraSec().InfraError("cron fields must all be set").Msg("")
		return errors.Errorfc(codes.InvalidArgument, "cron fields must all be set")
	}
	return nil
}

func validateRScheduledInput(in *schedule_v1.RepeatedScheduleResource) error {
	// we don't verify target relations as it's guarded by protobuf's oneof
	return validateCronFields(in)
}
