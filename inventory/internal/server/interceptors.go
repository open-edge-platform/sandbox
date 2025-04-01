// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"context"

	"golang.org/x/exp/slices"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	inv_v1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/inventory/v1"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/errors"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/tenant"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/util"
)

var (
	errMissingTenantID = errors.Errorfc(codes.InvalidArgument, "received request doesn't specify tenant ID")

	errTenantIDMismatch = errors.Errorfc(
		codes.InvalidArgument,
		"tenant ID specified in resource creation request is different that tenant ID defined in requested resource")

	errTenantUpdateNotAllowed = errors.Errorfc(codes.InvalidArgument, "tenant update is not allowed")

	errTenantIDAssertionFailed = errors.Errorfc(codes.InvalidArgument, "type assertion failed for tenantIDCarrier")
)

type tenantIDCarrier interface {
	GetTenantId() string
}

// TenantContextExtractingInterceptor - intercept tenantID provided by incoming gRPC request,
// and puts it into context send to underlying handlers.
func TenantContextExtractingInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		switch v := req.(type) {
		case *inv_v1.CreateResourceRequest:
			return handleCreateResourceRequest(ctx, v, handler)
		case *inv_v1.UpdateResourceRequest:
			return handleUpdateResourceRequest(ctx, v, handler)
		case tenantIDCarrier:
			return handleTenantIdentifierCarryingRequests(ctx, v, handler)
		default:
			return handler(ctx, req)
		}
	}
}

func handleTenantIdentifierCarryingRequests(ctx context.Context, req tenantIDCarrier, handler grpc.UnaryHandler) (any, error) {
	if req.GetTenantId() == "" {
		return nil, errMissingTenantID
	}
	return handler(tenant.AddTenantIDToContext(ctx, req.GetTenantId()), req)
}

func handleCreateResourceRequest(ctx context.Context, req *inv_v1.CreateResourceRequest, handler grpc.UnaryHandler) (any, error) {
	if req.GetTenantId() == "" {
		return nil, errMissingTenantID
	}

	resource, err := util.UnwrapResource[proto.Message](req.GetResource())
	if err != nil {
		return nil, err
	}

	tenantID, ok := resource.(tenantIDCarrier)
	if !ok {
		return nil, errTenantIDAssertionFailed
	}
	if req.GetTenantId() != tenantID.GetTenantId() {
		return nil, errTenantIDMismatch
	}

	return handler(tenant.AddTenantIDToContext(ctx, req.GetTenantId()), req)
}

func handleUpdateResourceRequest(ctx context.Context, req *inv_v1.UpdateResourceRequest, handler grpc.UnaryHandler) (any, error) {
	if req.GetTenantId() == "" {
		return nil, errMissingTenantID
	}

	resource, err := util.UnwrapResource[proto.Message](req.GetResource())
	if err != nil {
		return nil, err
	}

	tenantID, ok := resource.(tenantIDCarrier)
	if !ok {
		return nil, errTenantIDAssertionFailed
	}
	if isTenantIDUpdateRequested(tenantID, req.GetFieldMask()) {
		return nil, errTenantUpdateNotAllowed
	}

	return handler(tenant.AddTenantIDToContext(ctx, req.GetTenantId()), req)
}

func isTenantIDUpdateRequested(carrier tenantIDCarrier, fm *fieldmaskpb.FieldMask) bool {
	return carrier.GetTenantId() != "" && slices.Contains(fm.GetPaths(), "tenant_id")
}
