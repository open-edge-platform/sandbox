// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package policy

import (
	"context"
	"encoding/json"

	"github.com/open-policy-agent/opa/v1/rego"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"

	inv_v1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/inventory/v1"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/errors"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/logging"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/util"
)

var log = logging.GetLogger("InfraLogger")

const (
	clientKindKey               = "ClientKind"
	methodKey                   = "Method"
	desiredStateKey             = "DesiredState"
	currentStateKey             = "CurrentState"
	desiredStateFN              = "desired_state"
	currentStateFN              = "current_state"
	PolicyBundlePath            = "policyBundlePath"
	PolicyBundlePathDescription = "Path to policy bundle/files"
)

type Policy struct {
	query *rego.PreparedEvalQuery
}

func New(policyBundle string) (*Policy, error) {
	ctx := context.Background()

	query, err := rego.New(
		rego.Query("data.abac.abac"),
		rego.LoadBundle(policyBundle),
	).PrepareForEval(ctx)

	policy := Policy{query: &query}
	if err != nil {
		log.InfraSec().InfraErr(err).Msgf("can't load query")
		err = errors.Wrap(err)
	}
	return &policy, err
}

func setField(toVerify map[string]interface{}, resMessage proto.Message, fieldName, fieldKey string) error {
	fd := resMessage.ProtoReflect().Descriptor().Fields().ByName(protoreflect.Name(fieldName))
	// does not have the field we are looking for
	if fd == nil {
		return nil
	}
	val := resMessage.ProtoReflect().Get(fd)
	// they have but it is not an enum. Unsupported at the moment
	if fd.Kind() != protoreflect.EnumKind {
		log.InfraSec().InfraError("Unsupported field: %s, of kind: %s (%d), with value %s",
			fd.TextName(), fd.Kind(), fd.Kind(), val).Msg("")
		return errors.Errorfc(codes.InvalidArgument, "Unsupported filter field: %s, of kind: %s (%d), with value %s",
			fd.TextName(), fd.Kind(), fd.Kind(), val,
		)
	}
	value := int32(val.Enum())
	// UNSPECIFIED === unset
	if value != 0 {
		toVerify[fieldKey] = value
	}
	return nil
}

func buildInputMap(inMsg interface{}) (map[string]interface{}, error) {
	toVer := make(map[string]interface{})

	var err error
	var inMsgJSONbytes []byte
	var resMessage protoreflect.ProtoMessage

	reqMessage, resource, method, err := extractRequestDetails(inMsg)
	if err != nil {
		return nil, err
	}
	toVer[methodKey] = method

	// Create or Update
	if resource != nil {
		resMessage, err = util.UnwrapResource[proto.Message](resource)
		if err != nil {
			return nil, err
		}

		err = setField(toVer, resMessage, desiredStateFN, desiredStateKey)
		if err != nil {
			return nil, err
		}

		err = setField(toVer, resMessage, currentStateFN, currentStateKey)
		if err != nil {
			return nil, err
		}
	}

	inMsgJSONbytes, err = protojson.Marshal(reqMessage)
	if err != nil {
		log.InfraSec().InfraErr(err).Msg("error marshaling Proto to JSON bytes")
		return nil, errors.Errorfc(codes.InvalidArgument, "error marshaling Proto to JSON bytes")
	}

	err = json.Unmarshal(inMsgJSONbytes, &toVer)
	if err != nil {
		log.InfraSec().InfraErr(err).Msg("error while unmarshaling JSON bytes to JSON")
		return nil, errors.Errorfc(codes.InvalidArgument, "error while unmarshaling JSON bytes to JSON")
	}

	return toVer, nil
}

func extractRequestDetails(inMsg interface{}) (protoreflect.ProtoMessage, *inv_v1.Resource, string, error) {
	var resource *inv_v1.Resource
	var reqMessage protoreflect.ProtoMessage
	var method string

	switch message := inMsg.(type) {
	case *inv_v1.CreateResourceRequest:
		reqMessage = message
		createRes, ok := inMsg.(*inv_v1.CreateResourceRequest)
		if !ok {
			err := errors.Errorfc(codes.InvalidArgument, "failed to assert inMsg as CreateResourceRequest")
			log.InfraSec().InfraErr(err).Msg("")
			return nil, nil, "", err
		}
		resource = createRes.GetResource()
		method = "CREATE"
	case *inv_v1.UpdateResourceRequest:
		reqMessage = message
		updateRes, ok := inMsg.(*inv_v1.UpdateResourceRequest)
		if !ok {
			err := errors.Errorfc(codes.InvalidArgument, "failed to assert inMsg as UpdateResourceRequest")
			log.InfraSec().InfraErr(err).Msg("")
			return nil, nil, "", err
		}
		resource = updateRes.GetResource()
		method = "UPDATE"
	case *inv_v1.DeleteResourceRequest:
		reqMessage = message
		method = "DELETE"
	case *inv_v1.DeleteAllResourcesRequest:
		reqMessage = message
		method = "DELETE"
	default:
		errCase := errors.Errorfc(codes.InvalidArgument,
			"obtained unknown type of request to handle: %v", message)
		log.InfraSec().InfraErr(errCase).Msg("")
		return nil, nil, "", errCase
	}
	return reqMessage, resource, method, nil
}

func (p *Policy) Verify(clientKind string, inMsg interface{}) error {
	toVer, err := buildInputMap(inMsg)
	if err != nil {
		return err
	}
	toVer[clientKindKey] = clientKind

	// Depending on the previously prepared rego query and the input data let OPA decide on permission
	results, err := p.query.Eval(context.TODO(), rego.EvalInput(toVer))
	if err != nil {
		log.InfraSec().InfraErr(err).Msgf("eval failed")
		return errors.Errorfc(codes.PermissionDenied, "got %s for %s", err.Error(), toVer)
	}

	if !results.Allowed() {
		log.InfraSec().InfraError("API call blocked by OPA for client kind: %v", clientKind).Msg("")
		return errors.Errorfc(codes.PermissionDenied, "API call blocked by OPA for client kind: %v", clientKind)
	}

	log.InfraSec().Debug().Msg("API call is authorized for required changes")
	return nil
}
