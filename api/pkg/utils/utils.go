// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"context"
	"fmt"
	"math"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"

	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/errors"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/logging"
)

var log = logging.GetLogger("utils")

const authKey = "authorization"

// AppendJWTtoContext assumes that the context already has a JWT. All it does is it converts it properly
// to be compliant with Authenticator in gRPC interceptor which expects it to be a part of the metadata key (created by
// metadata package) inside the context.
func AppendJWTtoContext(ctx context.Context) (context.Context, error) {
	md, _ := metadata.FromIncomingContext(ctx)
	copiedCtx := metadata.NewOutgoingContext(ctx, md)
	jwtIntfc := ctx.Value(authKey)
	jwt, ok := jwtIntfc.(string)
	if ok && jwt != "" {
		return metadata.AppendToOutgoingContext(copiedCtx, authKey, jwt), nil
	}

	// temporarily returning context obtained at the beginning of the function - this is to handle the case,
	// when the authentication is not enabled in the API component
	// err := fmt.Errorf("JWT does not exist in the provided context %v: %v", jwtIntfc, ctx)
	// return nil, errors.Errorfc(codes.Internal, err.Error())
	// TODO when return error, encapsulate it into: err = errors.Errorfc(codes.Internal, err.Error())
	err := fmt.Errorf("can't append JWT to the context")
	log.InfraSec().Debug().Err(err).Msgf("%v due to allowing unauthenticated request from pre-defined client "+
		"(JWT is not present in the message context)", err)
	return ctx, nil
}

func SafeUintToInt(u uint) (int, error) {
	if u > math.MaxInt {
		return 0, errors.Errorfc(codes.InvalidArgument, "uint value exceeds int range")
	}
	return int(u), nil
}

func SafeUint64ToInt(u uint64) (int, error) {
	if u > math.MaxInt64 {
		return 0, errors.Errorfc(codes.InvalidArgument, "uint64 value exceeds int range")
	}
	return int(u), nil
}

func SafeIntToUint64(i int) (uint64, error) {
	if i < 0 {
		return 0, errors.Errorfc(codes.InvalidArgument, "int value is negative and cannot be converted to uint64")
	}
	return uint64(i), nil
}

func SafeIntToUint32(i int) (uint32, error) {
	if i < 0 {
		return 0, errors.Errorfc(codes.InvalidArgument, "int value is negative and cannot be converted to uint32")
	}
	if i > math.MaxUint32 {
		return 0, errors.Errorfc(codes.InvalidArgument, "int value exceeds uint32 range")
	}
	return uint32(i), nil
}
