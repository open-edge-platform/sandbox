// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package store

import (
	"time"

	"google.golang.org/grpc/codes"

	remoteaccessv1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/remoteaccess/v1"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/errors"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/util"
)

var remoteAccessResourceCreationValidators = []resourceValidator[*remoteaccessv1.RemoteAccessConfiguration]{
	protoValidator[*remoteaccessv1.RemoteAccessConfiguration],
	doNotAcceptResourceID[*remoteaccessv1.RemoteAccessConfiguration],
	expirationIsRequired,
	validateExpiration,
}

func validateCreationRequest(in *remoteaccessv1.RemoteAccessConfiguration) error {
	return validate(in, remoteAccessResourceCreationValidators...)
}

func expirationIsRequired(in *remoteaccessv1.RemoteAccessConfiguration) error {
	if in.ExpirationTimestamp == 0 {
		err := errors.Errorfc(codes.InvalidArgument, "expiration_timestamp cannot be 0")
		zlog.InfraSec().InfraErr(err).Msg("")
		return err
	}
	return nil
}

func validateExpiration(in *remoteaccessv1.RemoteAccessConfiguration) error {
	expirationTimestamp, err := util.Uint64ToInt64(in.ExpirationTimestamp)
	if err != nil {
		zlog.InfraSec().InfraErr(err).Msg("failed to parse expiration_timestamp")
		return err
	}
	expiration := time.Unix(expirationTimestamp, 0)
	start := time.Now()

	if expiration.Before(start.Add(inTimeOfRemoteAccess)) {
		err := errors.Errorfc(codes.InvalidArgument,
			"remote access cannot be granted for less than %s", inTimeOfRemoteAccess)
		zlog.InfraSec().InfraErr(err).Msg("")
		return err
	}

	if expiration.After(start.Add(maxTimeOfRemoteAccess)) {
		err := errors.Errorfc(codes.InvalidArgument,
			"remote access cannot be granted for more than %s", maxTimeOfRemoteAccess)
		zlog.InfraSec().InfraErr(err).Msg("")
		return err
	}
	return nil
}
