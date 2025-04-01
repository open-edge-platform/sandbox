// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

// Package rbac implements utility functions for Role-Based Access Control
package rbac

import (
	"context"
	"strings"

	"github.com/grpc-ecosystem/go-grpc-middleware/util/metautils"
	"github.com/open-policy-agent/opa/v1/rego"
	"google.golang.org/grpc/codes"

	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/errors"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/logging"
)

const (
	EnableAuth            = "enableAuth"
	EnableAuthDescription = "Enable JWT token authentication of each API call"
	RbacRules             = "rbacRules"
	RbacRulesDescription  = "Path to the rego rules files which contains RBAC policies"
	GetKey                = "Get"
	ListKey               = "List"
	SummaryKey            = "Summary"
	SubscribeKey          = "Subscribe"
	FindKey               = "Find"
	PostKey               = "Post"
	PutKey                = "Put"
	PatchKey              = "Patch"
	DeleteKey             = "Delete"
	CreateKey             = "Create"
	RegisterKey           = "Register"
	UpdateKey             = "Update"
)

var (
	zlog  = logging.GetLogger("infra-rbac")
	woKey = "write-only"
	roKey = "read-only"
)

type Policy struct {
	queries map[string]*rego.PreparedEvalQuery
}

func New(ruleDir string) (*Policy, error) {
	ctx := context.Background()

	policies := Policy{
		queries: make(map[string]*rego.PreparedEvalQuery, 0),
	}

	woQuery, err := rego.New(
		rego.Query("data.authz.hasWriteAccess"),
		rego.Load([]string{ruleDir}, nil), // loads all files within directory
	).PrepareForEval(ctx)
	if err != nil {
		zlog.InfraSec().InfraErr(err).Msgf("can't load write-only query")
		return nil, errors.Wrap(err)
	}
	policies.queries[woKey] = &woQuery

	roQuery, err := rego.New(
		rego.Query("data.authz.hasReadAccess"),
		rego.Load([]string{ruleDir}, nil), // loads all files within directory
	).PrepareForEval(ctx)
	if err != nil {
		zlog.InfraSec().InfraErr(err).Msgf("can't load read-only query")
		return nil, errors.Wrap(err)
	}
	policies.queries[roKey] = &roQuery

	return &policies, nil
}

func (p *Policy) Verify(ctxClaims metautils.NiceMD, operation string) error {
	allowed := false

	switch strings.ToLower(operation) {
	case strings.ToLower(GetKey), strings.ToLower(ListKey), strings.ToLower(SummaryKey),
		strings.ToLower(SubscribeKey), strings.ToLower(FindKey):
		query, ok := p.queries[roKey]
		if !ok {
			zlog.InfraSec().InfraError("can't extract Read-Only query for %s", operation).Msg("")
			return errors.Errorfc(codes.PermissionDenied, "can't extract Read-Only query for %s", operation)
		}
		result, err := query.Eval(context.Background(), rego.EvalInput(ctxClaims))
		if err != nil {
			zlog.InfraSec().InfraError("got %s for %s", err.Error(), operation).Msg("")
			return errors.Errorfc(codes.PermissionDenied, "got %s for %s", err.Error(), operation)
		}
		if !result.Allowed() {
			zlog.InfraSec().InfraError("%s is blocked by OPA", operation).Msg("Authorization failed")
			return errors.Errorfc(codes.PermissionDenied, "%s is blocked by OPA", operation)
		}
		allowed = true
	case strings.ToLower(PostKey), strings.ToLower(PutKey), strings.ToLower(PatchKey), strings.ToLower(DeleteKey),
		strings.ToLower(CreateKey), strings.ToLower(RegisterKey), strings.ToLower(UpdateKey):
		query, ok := p.queries[woKey]
		if !ok {
			zlog.InfraSec().InfraError("can't extract Read-Write query for %s", operation).Msg("")
			return errors.Errorfc(codes.PermissionDenied, "can't extract Read-Write query for %s", operation)
		}
		result, err := query.Eval(context.Background(), rego.EvalInput(ctxClaims))
		if err != nil {
			zlog.InfraSec().InfraError("got %s for %s", err.Error(), operation).Msg("")
			return errors.Errorfc(codes.PermissionDenied, "got %s for %s", err.Error(), operation)
		}
		if !result.Allowed() {
			zlog.InfraSec().InfraError("%s is blocked by OPA", operation).Msg("")
			return errors.Errorfc(codes.PermissionDenied, "%s is blocked by OPA", operation)
		}
		allowed = true
	default:
		zlog.InfraSec().InfraError("authorization error - obtained unspecified operation: %s", operation).
			Msg("Authorization failed")
		return errors.Errorfc(codes.PermissionDenied, "authorization error - obtained unspecified operation: %s", operation)
	}

	if allowed {
		zlog.Debug().Msgf("Request %s is authorized", operation)
		return nil
	}

	// This is an internal error
	zlog.InfraSec().InfraError("something unexpected in the authorization process for %s", operation).Msg("Authorization failed")
	return errors.Errorf("something unexpected in the authorization process for %s", operation)
}
