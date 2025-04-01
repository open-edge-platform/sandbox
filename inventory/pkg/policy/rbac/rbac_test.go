// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package rbac_test

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/grpc-ecosystem/go-grpc-middleware/util/metautils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	grpc_status "google.golang.org/grpc/status"

	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/logging"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/policy/rbac"
)

var (
	regoPath = "/rego/authz.rego"
	log      = logging.GetLogger("rbac")
)

func TestMain(m *testing.M) {
	policyBundle := flag.String(
		"policyBundle",
		"/rego/policy_bundle.tar.gz",
		"Path of policy rego file",
	)
	log.Debug().Msgf("policyBundle specified for policy tests %s", *policyBundle)
	run := m.Run() // run all tests
	os.Exit(run)
}

func loadPolicyBundle(regoPath string) (*rbac.Policy, error) {
	pwd, err := os.Getwd()
	if err != nil {
		err = fmt.Errorf("could not get current directory pwd error %s", err.Error())
		return nil, err
	}
	regoPathFullPath := filepath.Join(pwd, "../../../", regoPath)
	pol, err := rbac.New(regoPathFullPath)
	if err != nil {
		return nil, err
	}

	return pol, nil
}

func TestAuthorizeOPA(t *testing.T) {
	p, err := loadPolicyBundle(regoPath)
	require.NoError(t, err)

	ctx := context.TODO()

	testCases := []struct {
		name       string
		md         metautils.NiceMD
		valid      bool
		expErrCode codes.Code
		methods    []string
	}{
		{
			name:       "Empty context 1",
			md:         metautils.ExtractIncoming(ctx),
			valid:      false,
			expErrCode: codes.PermissionDenied,
			methods:    []string{rbac.GetKey},
		},
		{
			name:       "Empty context 2",
			md:         metautils.ExtractIncoming(ctx),
			valid:      false,
			expErrCode: codes.PermissionDenied,
			methods:    []string{""},
		},
		{
			name: "Deprecated RO role NBI",
			md: func() metautils.NiceMD {
				roIMNiceMdNBI := metautils.ExtractIncoming(ctx)
				roIMNiceMdNBI.Add("realm_access/roles", "infra-manager-core-read-role")
				return roIMNiceMdNBI
			}(),
			valid:      false,
			expErrCode: codes.PermissionDenied,
		},
		{
			name: "RO role NBI",
			md: func() metautils.NiceMD {
				roIMNiceMdNBI := metautils.ExtractIncoming(ctx)
				roIMNiceMdNBI.Add("realm_access/roles", "im-r")
				return roIMNiceMdNBI
			}(),
			valid:      false,
			expErrCode: codes.PermissionDenied,
			methods:    []string{rbac.GetKey, rbac.ListKey, rbac.FindKey},
		},
		{
			name: "Deprecated RO role NBI fail write",
			md: func() metautils.NiceMD {
				roIMNiceMdNBI := metautils.ExtractIncoming(ctx)
				roIMNiceMdNBI.Add("realm_access/roles", "infra-manager-core-read-role")
				return roIMNiceMdNBI
			}(),
			valid:      false,
			expErrCode: codes.PermissionDenied,
			methods:    []string{rbac.PostKey, rbac.PatchKey, rbac.CreateKey, rbac.DeleteKey, rbac.PutKey},
		},
		{
			name: "RO role NBI fail write",
			md: func() metautils.NiceMD {
				roIMNiceMdNBI := metautils.ExtractIncoming(ctx)
				roIMNiceMdNBI.Add("realm_access/roles", "im-r")
				return roIMNiceMdNBI
			}(),
			valid:      false,
			expErrCode: codes.PermissionDenied,
			methods:    []string{rbac.PostKey, rbac.PatchKey, rbac.CreateKey, rbac.DeleteKey, rbac.PutKey},
		},
		{
			name: "Deprecated WO role NBI",
			md: func() metautils.NiceMD {
				roIMNiceMdNBI := metautils.ExtractIncoming(ctx)
				roIMNiceMdNBI.Add("realm_access/roles", "infra-manager-core-write-role")
				return roIMNiceMdNBI
			}(),
			valid:      false,
			expErrCode: codes.PermissionDenied,
		},
		{
			name: "WO role NBI",
			md: func() metautils.NiceMD {
				roIMNiceMdNBI := metautils.ExtractIncoming(ctx)
				roIMNiceMdNBI.Add("realm_access/roles", "im-rw")
				return roIMNiceMdNBI
			}(),
			valid:      false,
			expErrCode: codes.PermissionDenied,
			methods:    []string{rbac.PostKey, rbac.PatchKey, rbac.CreateKey, rbac.DeleteKey, rbac.PutKey},
		},
		{
			name: "Deprecated WO role NBI fail read",
			md: func() metautils.NiceMD {
				roIMNiceMdNBI := metautils.ExtractIncoming(ctx)
				roIMNiceMdNBI.Add("realm_access/roles", "infra-manager-core-write-role")
				return roIMNiceMdNBI
			}(),
			valid:      false,
			expErrCode: codes.PermissionDenied,
			methods:    []string{rbac.GetKey, rbac.ListKey, rbac.FindKey},
		},
		{
			name: "WO role NBI fail read",
			md: func() metautils.NiceMD {
				roIMNiceMdNBI := metautils.ExtractIncoming(ctx)
				roIMNiceMdNBI.Add("realm_access/roles", "im-rw")
				return roIMNiceMdNBI
			}(),
			valid:      false,
			expErrCode: codes.PermissionDenied,
			methods:    []string{rbac.GetKey, rbac.ListKey, rbac.FindKey},
		},
		{
			name: "Deprecated WO role NBI UUID",
			md: func() metautils.NiceMD {
				roIMNiceMdNBI := metautils.ExtractIncoming(ctx)
				roIMNiceMdNBI.Add("realm_access/roles", "00000000-0000-0000-0000-000000000000_infra-manager-core-write-role")
				return roIMNiceMdNBI
			}(),
			valid:      false,
			expErrCode: codes.PermissionDenied,
		},
		{
			name: "WO role NBI UUID ",
			md: func() metautils.NiceMD {
				roIMNiceMdNBI := metautils.ExtractIncoming(ctx)
				roIMNiceMdNBI.Add("realm_access/roles", "00000000-0000-0000-0000-000000000000_im-rw")
				return roIMNiceMdNBI
			}(),
			valid:   true,
			methods: []string{rbac.PostKey, rbac.PatchKey, rbac.CreateKey, rbac.DeleteKey, rbac.PutKey},
		},
		{
			name: "Deprecated WO role NBI UUID fail read",
			md: func() metautils.NiceMD {
				roIMNiceMdNBI := metautils.ExtractIncoming(ctx)
				roIMNiceMdNBI.Add("realm_access/roles", "00000000-0000-0000-0000-000000000000_infra-manager-core-write-role")
				return roIMNiceMdNBI
			}(),
			valid:      false,
			expErrCode: codes.PermissionDenied,
			methods:    []string{rbac.GetKey, rbac.ListKey, rbac.FindKey},
		},
		{
			name: "Deprecated WO role NBI wrong UUID",
			md: func() metautils.NiceMD {
				roIMNiceMdNBI := metautils.ExtractIncoming(ctx)
				roIMNiceMdNBI.Add("realm_access/roles", "00000000-0000-0000--000000000000_infra-manager-core-write-role")
				return roIMNiceMdNBI
			}(),
			valid:      false,
			expErrCode: codes.PermissionDenied,
			methods: []string{
				rbac.GetKey, rbac.ListKey, rbac.FindKey,
				rbac.PostKey, rbac.PatchKey, rbac.CreateKey, rbac.DeleteKey, rbac.PutKey,
			},
		},
		{
			name: "WO role NBI wrong UUID",
			md: func() metautils.NiceMD {
				roIMNiceMdNBI := metautils.ExtractIncoming(ctx)
				roIMNiceMdNBI.Add("realm_access/roles", "00000000-0000-0000--000000000000_im-rw")
				return roIMNiceMdNBI
			}(),
			valid:      false,
			expErrCode: codes.PermissionDenied,
			methods: []string{
				rbac.GetKey, rbac.ListKey, rbac.FindKey,
				rbac.PostKey, rbac.PatchKey, rbac.CreateKey, rbac.DeleteKey, rbac.PutKey,
			},
		},
		{
			name: "Deprecated RO role NBI UUID",
			md: func() metautils.NiceMD {
				roIMNiceMdNBI := metautils.ExtractIncoming(ctx)
				roIMNiceMdNBI.Add("realm_access/roles", "00000000-0000-0000-0000-000000000000_infra-manager-core-read-role")
				return roIMNiceMdNBI
			}(),
			valid:      false,
			expErrCode: codes.PermissionDenied,
		},
		{
			name: "RO role NBI UUID",
			md: func() metautils.NiceMD {
				roIMNiceMdNBI := metautils.ExtractIncoming(ctx)
				roIMNiceMdNBI.Add("realm_access/roles", "00000000-0000-0000-0000-000000000000_im-r")
				return roIMNiceMdNBI
			}(),
			valid:   true,
			methods: []string{rbac.GetKey, rbac.ListKey, rbac.FindKey},
		},
		{
			name: "Deprecated RO role NBI UUID fail write",
			md: func() metautils.NiceMD {
				roIMNiceMdNBI := metautils.ExtractIncoming(ctx)
				roIMNiceMdNBI.Add("realm_access/roles", "00000000-0000-0000-0000-000000000000_infra-manager-core-read-role")
				return roIMNiceMdNBI
			}(),
			valid:      false,
			expErrCode: codes.PermissionDenied,
			methods:    []string{rbac.PostKey, rbac.PatchKey, rbac.CreateKey, rbac.DeleteKey, rbac.PutKey},
		},
		{
			name: "RO role NBI UUID fail write",
			md: func() metautils.NiceMD {
				roIMNiceMdNBI := metautils.ExtractIncoming(ctx)
				roIMNiceMdNBI.Add("realm_access/roles", "00000000-0000-0000-0000-000000000000_im-r")
				return roIMNiceMdNBI
			}(),
			valid:      false,
			expErrCode: codes.PermissionDenied,
			methods:    []string{rbac.PostKey, rbac.PatchKey, rbac.CreateKey, rbac.DeleteKey, rbac.PutKey},
		},
		{
			name: "Deprecated RO role NBI wrong UUID",
			md: func() metautils.NiceMD {
				roIMNiceMdNBI := metautils.ExtractIncoming(ctx)
				roIMNiceMdNBI.Add("realm_access/roles", "00000000-0000-0000--000000000000_infra-manager-core-read-role")
				return roIMNiceMdNBI
			}(),
			valid:      false,
			expErrCode: codes.PermissionDenied,
			methods: []string{
				rbac.GetKey, rbac.ListKey, rbac.FindKey,
				rbac.PostKey, rbac.PatchKey, rbac.CreateKey, rbac.DeleteKey, rbac.PutKey,
			},
		},
		{
			name: "RO role NBI wrong UUID",
			md: func() metautils.NiceMD {
				roIMNiceMdNBI := metautils.ExtractIncoming(ctx)
				roIMNiceMdNBI.Add("realm_access/roles", "00000000-0000-0000--000000000000_im-r")
				return roIMNiceMdNBI
			}(),
			valid:      false,
			expErrCode: codes.PermissionDenied,
			methods: []string{
				rbac.GetKey, rbac.ListKey, rbac.FindKey,
				rbac.PostKey, rbac.PatchKey, rbac.CreateKey, rbac.DeleteKey, rbac.PutKey,
			},
		},
		{
			name: "Deprecated EN SBI role",
			md: func() metautils.NiceMD {
				roIMNiceMdNBI := metautils.ExtractIncoming(ctx)
				roIMNiceMdNBI.Add("realm_access/roles", "node-agent-readwrite-role")
				return roIMNiceMdNBI
			}(),
			valid:      false,
			expErrCode: codes.PermissionDenied,
		},
		{
			name: "EN SBI role",
			md: func() metautils.NiceMD {
				roIMNiceMdNBI := metautils.ExtractIncoming(ctx)
				roIMNiceMdNBI.Add("realm_access/roles", "en-agent-rw")
				return roIMNiceMdNBI
			}(),
			valid:      false,
			expErrCode: codes.PermissionDenied,
			methods: []string{
				rbac.GetKey, rbac.ListKey, rbac.FindKey,
				rbac.PostKey, rbac.PatchKey, rbac.CreateKey, rbac.DeleteKey, rbac.PutKey,
			},
		},
		{
			name: "Deprecated EN SBI role UUID",
			md: func() metautils.NiceMD {
				roIMNiceMdNBI := metautils.ExtractIncoming(ctx)
				roIMNiceMdNBI.Add("realm_access/roles", "00000000-0000-0000-0000-000000000000_node-agent-readwrite-role")
				return roIMNiceMdNBI
			}(),
			valid:      false,
			expErrCode: codes.PermissionDenied,
		},
		{
			name: "RW role NBI UUID read",
			md: func() metautils.NiceMD {
				roIMNiceMdNBI := metautils.ExtractIncoming(ctx)
				roIMNiceMdNBI.Add("realm_access/roles", "00000000-0000-0000-0000-000000000000_im-rw")
				return roIMNiceMdNBI
			}(),
			valid:   true,
			methods: []string{rbac.GetKey, rbac.ListKey, rbac.FindKey},
		},
		{
			name: "EN SBI role UUID",
			md: func() metautils.NiceMD {
				roIMNiceMdNBI := metautils.ExtractIncoming(ctx)
				roIMNiceMdNBI.Add("realm_access/roles", "00000000-0000-0000-0000-000000000000_en-agent-rw")
				return roIMNiceMdNBI
			}(),
			valid: true,
			methods: []string{
				rbac.GetKey, rbac.ListKey, rbac.FindKey,
				rbac.PostKey, rbac.PatchKey, rbac.CreateKey, rbac.DeleteKey, rbac.PutKey,
			},
		},
		{
			name: "Deprecated EN SBI role wrong UUID",
			md: func() metautils.NiceMD {
				roIMNiceMdNBI := metautils.ExtractIncoming(ctx)
				roIMNiceMdNBI.Add("realm_access/roles", "00000000-0000-0000--000000000000_node-agent-readwrite-role")
				return roIMNiceMdNBI
			}(),
			valid:      false,
			expErrCode: codes.PermissionDenied,
			methods: []string{
				rbac.GetKey, rbac.ListKey, rbac.FindKey,
				rbac.PostKey, rbac.PatchKey, rbac.CreateKey, rbac.DeleteKey, rbac.PutKey,
			},
		},
		{
			name: "EN SBI role wrong UUID",
			md: func() metautils.NiceMD {
				roIMNiceMdNBI := metautils.ExtractIncoming(ctx)
				roIMNiceMdNBI.Add("realm_access/roles", "00000000-0000-0000--000000000000_en-agent-rw")
				return roIMNiceMdNBI
			}(),
			valid:      false,
			expErrCode: codes.PermissionDenied,
			methods: []string{
				rbac.GetKey, rbac.ListKey, rbac.FindKey,
				rbac.PostKey, rbac.PatchKey, rbac.CreateKey, rbac.DeleteKey, rbac.PutKey,
			},
		},
		{
			name: "deprecated onboarding role",
			md: func() metautils.NiceMD {
				roIMNiceMdNBI := metautils.ExtractIncoming(ctx)
				roIMNiceMdNBI.Add("realm_access/roles", "00000000-0000-0000-0000-000000000000_edge-onboarding-role")
				return roIMNiceMdNBI
			}(),
			valid:      false,
			expErrCode: codes.PermissionDenied,
		},
		{
			name: "onboarding role",
			md: func() metautils.NiceMD {
				roIMNiceMdNBI := metautils.ExtractIncoming(ctx)
				roIMNiceMdNBI.Add("realm_access/roles", "00000000-0000-0000-0000-000000000000_en-ob")
				return roIMNiceMdNBI
			}(),
			valid: true,
			methods: []string{
				rbac.GetKey, rbac.ListKey, rbac.FindKey,
				rbac.PostKey, rbac.PatchKey, rbac.CreateKey, rbac.DeleteKey, rbac.PutKey,
			},
		},
		{
			name: "Deprecated onboarding role wrong UUID",
			md: func() metautils.NiceMD {
				roIMNiceMdNBI := metautils.ExtractIncoming(ctx)
				roIMNiceMdNBI.Add("realm_access/roles", "00000000-0000-0000--000000000000_edge-onboarding-role")
				return roIMNiceMdNBI
			}(),
			valid:      false,
			expErrCode: codes.PermissionDenied,
			methods: []string{
				rbac.GetKey, rbac.ListKey, rbac.FindKey,
				rbac.PostKey, rbac.PatchKey, rbac.CreateKey, rbac.DeleteKey, rbac.PutKey,
			},
		},
		{
			name: "onboarding role wrong UUID",
			md: func() metautils.NiceMD {
				roIMNiceMdNBI := metautils.ExtractIncoming(ctx)
				roIMNiceMdNBI.Add("realm_access/roles", "00000000-0000-0000--000000000000_en-ob")
				return roIMNiceMdNBI
			}(),
			valid:      false,
			expErrCode: codes.PermissionDenied,
			methods: []string{
				rbac.GetKey, rbac.ListKey, rbac.FindKey,
				rbac.PostKey, rbac.PatchKey, rbac.CreateKey, rbac.DeleteKey, rbac.PutKey,
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			for _, method := range tc.methods {
				err := p.Verify(tc.md, method)
				if tc.valid {
					require.NoErrorf(t, err, "method %s", method)
				} else {
					require.Errorf(t, err, "method %s", method)
					assert.IsTypef(t, err, grpc_status.Error(tc.expErrCode, tc.expErrCode.String()), "method %s", method)
				}
			}
		})
	}
}

func TestAuthorizeOPAError(t *testing.T) {
	_, err := loadPolicyBundle("")
	require.Error(t, err)
}
