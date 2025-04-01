// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package server_test

import (
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/open-edge-platform/infra-core/inventory/v2/internal/server"
	inv_testing "github.com/open-edge-platform/infra-core/inventory/v2/pkg/testing"
)

var certPath string

func TestMain(m *testing.M) {
	// Currently unused
	flag.String(
		"policyBundle",
		"/rego/policy_bundle.tar.gz",
		"Path of policy rego file",
	)
	flag.Parse()
	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	projectRoot := filepath.Dir(filepath.Dir(wd))

	policyPath := projectRoot + "/out"
	certPath = projectRoot + "/cert/certificates"
	migrationsDir := projectRoot + "/out"

	inv_testing.StartTestingEnvironment(policyPath, certPath, migrationsDir)
	run := m.Run() // run all tests
	inv_testing.StopTestingEnvironment()

	os.Exit(run)
}

func Test_getServerOpts(t *testing.T) {
	tests := []struct {
		name    string
		args    server.Options
		wantErr bool
	}{
		{
			name: "Valid server options",
			args: server.Options{
				InsecureGrpc: true,
			},
			wantErr: false,
		},
		{
			name: "Valid secure server options",
			args: server.Options{
				InsecureGrpc: false,
				CaCertPath:   certPath + "/ca-cert.pem",
				TLSCertPath:  certPath + "/server-cert.pem",
				TLSKeyPath:   certPath + "/server-key.pem",
			},
			wantErr: false,
		},
		{
			name: "Valid metrics options",
			args: server.Options{
				EnableMetrics: false,
				CaCertPath:    certPath + "/ca-cert.pem",
				TLSCertPath:   certPath + "/server-cert.pem",
				TLSKeyPath:    certPath + "/server-key.pem",
			},
			wantErr: false,
		},
		{
			name: "Valid server options with auth and tracing",

			args: server.Options{
				EnableTracing:  true,
				EnableAuth:     true,
				InsecureGrpc:   true,
				EnableAuditing: false,
			},
			wantErr: false,
		},
		{
			name: "Invalid server options - no cert paths",
			args: server.Options{
				InsecureGrpc: false,
				CaCertPath:   "",
				TLSCertPath:  "",
				TLSKeyPath:   "",
			},
			wantErr: true,
		},
		{
			name: "Invalid secure server options",
			args: server.Options{
				InsecureGrpc: false,
				CaCertPath:   certPath + "/ca-cert.pem",
				TLSCertPath:  certPath + "/server-cert.pem",
				TLSKeyPath:   certPath + "/not-existent.pem",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := server.GetServerOpts(tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetServerOpts() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}
