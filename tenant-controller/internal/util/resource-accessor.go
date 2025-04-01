// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package util

import (
	"fmt"

	"google.golang.org/protobuf/reflect/protoreflect"

	inv_v1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/inventory/v1"
)

// SetResourceValue - sets fieldValue for fieldName on given resource.
func SetResourceValue(resource *inv_v1.Resource, fieldName string, fieldValue any) error {
	if resource == nil {
		return fmt.Errorf("provided resource cannot be nil")
	}
	resourceReflect := resource.ProtoReflect()
	oneOfDescriptor := resourceReflect.Descriptor().Oneofs().ByName("resource")
	oneOfField := resourceReflect.WhichOneof(oneOfDescriptor)
	if oneOfField == nil {
		// shall never happen
		return fmt.Errorf(`given resource (%s) does not contain "resource" field`, resource)
	}

	oneOfValue := resourceReflect.Get(oneOfField)
	if !oneOfValue.IsValid() {
		return fmt.Errorf("given resource envelope (%s) is empty - it does not contain resource definition", resource)
	}
	innerMsg := oneOfValue.Message()

	field := innerMsg.Descriptor().Fields().ByName(protoreflect.Name(fieldName))
	if field == nil {
		return fmt.Errorf("given resource (%s) does not contain requested field: %s", resource, fieldName)
	}

	pv, err := toProtoType(fieldValue)
	if err != nil {
		return err
	}
	innerMsg.Set(field, pv)
	return nil
}

func toProtoType(a any) (protoreflect.Value, error) {
	switch v := a.(type) {
	case string:
		return protoreflect.ValueOfString(v), nil
	default:
		return protoreflect.Value{}, fmt.Errorf("unrecognized type: %v", a)
	}
}
