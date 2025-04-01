// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package cert

import (
	"crypto/tls"
	"crypto/x509"
	"os"

	"google.golang.org/grpc/credentials"

	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/errors"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/logging"
)

var zlog = logging.GetLogger("InfraCert")

// HandleCertPaths creates credentials for gRPC dial options using paths of ca, key, cert files.
func HandleCertPaths(
	caPath string,
	keyPath string,
	certPath string,
	insecure bool,
) (credentials.TransportCredentials, error) {
	var cert tls.Certificate
	var err error

	cert, err = tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		return nil, errors.Wrap(err)
	}

	var clientCAs *x509.CertPool

	clientCAs, err = GetCertPool(caPath)
	if err != nil {
		return nil, err
	}

	tlsConfig := &tls.Config{
		Certificates:       []tls.Certificate{cert},
		ClientCAs:          clientCAs,
		InsecureSkipVerify: insecure,
	}

	return credentials.NewTLS(tlsConfig), nil
}

// GetCertPool loads the Certificate Authority from the given path.
func GetCertPool(caPath string) (*x509.CertPool, error) {
	certPool := x509.NewCertPool()
	ca, err := os.ReadFile(caPath)
	if err != nil {
		return nil, errors.Wrap(err)
	}
	if ok := certPool.AppendCertsFromPEM(ca); !ok {
		zlog.InfraSec().InfraError("failed to append CA certificate from %s", caPath).Msg("")
		return nil, errors.Errorf("failed to append CA certificate from %s", caPath)
	}
	return certPool, nil
}
