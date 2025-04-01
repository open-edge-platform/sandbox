// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package store

import (
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/proto"

	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/errors"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/validator"
)

type resourceValidator[T invResource] func(in T) error

type resourceIDGetter interface {
	GetResourceId() string
}

type invResource interface {
	proto.Message
	resourceIDGetter
}

func validate[T invResource](in T, validators ...resourceValidator[T]) error {
	for _, validate := range validators {
		if err := validate(in); err != nil {
			return err
		}
	}
	return nil
}

func validateProto[T proto.Message](in T) error {
	return protoValidator(in)
}

func protoValidator[T proto.Message](in T) error {
	if err := validator.ValidateMessage(in); err != nil {
		zlog.InfraSec().InfraErr(err).Msg("")
		return err
	}
	return nil
}

func doNotAcceptResourceID[T invResource](in T) error {
	if in.GetResourceId() != "" {
		zlog.InfraSec().InfraError("resource ID can't be set in create requests").Msg("")
		return errors.Errorfc(codes.InvalidArgument, "resource ID can't be set in create requests")
	}
	return nil
}
