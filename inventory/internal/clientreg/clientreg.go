// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package clientreg

import (
	"context"
	"sync"

	uuid "github.com/google/uuid"
	"golang.org/x/exp/slices"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"

	inv_v1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/inventory/v1"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/errors"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/logging"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/tracing"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/util"
)

var zlog = logging.GetLogger("InfraInvClientReg")

type ClientInfo struct {
	Name          string
	Version       string
	ClientKind    inv_v1.ClientKind
	ResourceKinds []inv_v1.ResourceKind
	Stream        inv_v1.InventoryService_SubscribeEventsServer
}

type ClientReg struct {
	regClients    sync.Map
	enableTracing bool
}

func NewClientReg(enableTracing bool) *ClientReg {
	cr := &ClientReg{
		enableTracing: enableTracing,
	}
	return cr
}

func (cr *ClientReg) RegisterClient(clientInfo ClientInfo) (string, error) {
	if clientInfo.Name == "" {
		zlog.InfraSec().InfraError("client name empty").Msg("")
		return "", errors.Errorfc(codes.InvalidArgument, "client name empty")
	}
	if clientInfo.ClientKind == inv_v1.ClientKind_CLIENT_KIND_UNSPECIFIED {
		zlog.InfraSec().InfraError("unspecified client kind").Msg("")
		return "", errors.Errorfc(codes.InvalidArgument, "unspecified client kind")
	}
	slices.Sort(clientInfo.ResourceKinds)
	if len(slices.Compact(clientInfo.ResourceKinds)) != len(clientInfo.ResourceKinds) {
		zlog.InfraSec().InfraError("resource kind subscriptions contain duplicates: %v", clientInfo.ResourceKinds).Msg("")
		return "", errors.Errorfc(codes.InvalidArgument,
			"resource kind subscriptions contain duplicates: %v", clientInfo.ResourceKinds)
	}
	if clientInfo.Stream == nil {
		zlog.InfraSec().InfraError("gRPC stream missing").Msg("")
		return "", errors.Errorfc(codes.Internal, "gRPC stream missing")
	}

	// generate UUID
	clientUUID := uuid.New().String()

	zlog.InfraSec().Info().Msgf("RegisterClient %s", clientUUID)
	cr.regClients.Store(clientUUID, clientInfo)

	return clientUUID, nil
}

func (cr *ClientReg) ClientRegistrationMap() *sync.Map {
	return &cr.regClients
}

func (cr *ClientReg) UpdateClient(clientUUID string, kinds []inv_v1.ResourceKind) error {
	v, ok := cr.regClients.Load(clientUUID)
	if !ok {
		return errors.Errorfc(codes.NotFound, "client with UUID %v not found", clientUUID)
	}
	clientInfo, ok := v.(ClientInfo)
	if !ok {
		return errors.Errorfc(codes.Internal, "client with UUID %v has corrupt info", clientUUID)
	}
	slices.Sort(kinds)
	if len(slices.Compact(kinds)) != len(kinds) {
		zlog.InfraSec().InfraError("resource kind subscriptions contain duplicates: %v", kinds).Msg("")
		return errors.Errorfc(codes.InvalidArgument,
			"resource kind subscriptions contain duplicates: %v", kinds)
	}
	clientInfo.ResourceKinds = kinds
	zlog.InfraSec().Info().Msgf("UpdateClient %s with kinds %v", clientUUID, kinds)
	cr.regClients.Store(clientUUID, clientInfo)

	return nil
}

func (cr *ClientReg) ExitClient(clientUUID string) {
	zlog.InfraSec().Info().Msgf("ExitClient: %s", clientUUID)
	cr.regClients.Delete(clientUUID)
}

func (cr *ClientReg) StreamNotify(
	ctx context.Context,
	eventKind inv_v1.SubscribeEventsResponse_EventKind,
	resource *inv_v1.Resource,
	sourceUUID string,
) {
	resID, err := util.GetResourceIDFromResource(resource)
	if err != nil {
		return
	}
	kind := util.GetResourceKindFromResource(resource)
	cr.notifyClients(ctx, eventKind, resource, sourceUUID, resID, kind)
}

func (cr *ClientReg) notifyClients(
	ctx context.Context,
	eventKind inv_v1.SubscribeEventsResponse_EventKind,
	resource *inv_v1.Resource,
	sourceUUID, resID string,
	kind inv_v1.ResourceKind,
) {
	// Iterate over the registered clients.
	cr.regClients.Range(func(uuid, info any) bool {
		// Don't notify the source of the notification.
		if uuid == sourceUUID {
			return true // Skip, but continue.
		}

		clientInfo, ok := info.(ClientInfo)
		if !ok || clientInfo.Stream == nil {
			return true // Skip, but continue.
		}

		for _, subscribedKind := range clientInfo.ResourceKinds {
			if kind != subscribedKind {
				continue
			}

			zlog := zlog.TraceCtx(ctx)
			zlog.Debug().Msgf("Found Client to Stream to: %s", info)
			stream := clientInfo.Stream
			if cr.enableTracing {
				ctx = tracing.StartTraceFromRemote(ctx, "infra-inventory", "notify")
				md, ok := metadata.FromIncomingContext(ctx)
				if ok {
					zlog.Debug().Msgf("ctx incoming metadata %v", md)
					err := stream.SetHeader(md)
					if err != nil {
						zlog.InfraErr(err).Msgf("header not set in stream")
					}
				}
				tracing.StopTrace(ctx)
			}

			subresp := inv_v1.SubscribeEventsResponse{
				ResourceId: resID,
				Resource:   resource,
				EventKind:  eventKind,
			}
			if err := stream.Send(&subresp); err != nil {
				zlog.Warn().Msgf("Problem streaming to: %s", info)
			}
		}
		return true
	})
}
