// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

//nolint:testpackage // testing internal functions
package auth

import (
	"context"
	"flag"
	"os"
	"testing"

	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/flags"
)

func TestMain(m *testing.M) {
	// Only needed to suppress the error
	flag.String(
		"policyBundle",
		"/rego/policy_bundle.tar.gz",
		"Path of policy rego file",
	)
	flag.Parse()

	run := m.Run() // run all tests
	os.Exit(run)
}

func Test_auth_init(t *testing.T) {
	type args struct {
		ctx             context.Context
		disableCredMgmt bool
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "Credentials Management enabled",
			args: args{
				ctx:             context.Background(),
				disableCredMgmt: false,
			},
			wantErr: true,
		},
		{
			name: "Credentials Management disabled",
			args: args{
				ctx:             context.Background(),
				disableCredMgmt: true,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flags.FlagDisableCredentialsManagement = flag.Bool(tt.name, tt.args.disableCredMgmt, "")
			if err := Init(); (err != nil) != tt.wantErr {
				t.Errorf("auth.Init() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
	// ensure the default value for the other tests
	flags.FlagDisableCredentialsManagement = flag.Bool("disable-credentials", false, "")
}
