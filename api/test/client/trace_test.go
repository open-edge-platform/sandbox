// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"context"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

	"github.com/open-edge-platform/infra-core/api/pkg/api/v0"
	"github.com/open-edge-platform/infra-core/api/test/utils"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/tracing"
)

const (
	traceURL = "127.0.0.1:4318"
)

var traceAttribs = map[string]string{
	"test.name": "trace",
}

func setTransport(c *api.Client) error {
	c.Client = &http.Client{
		Transport: otelhttp.NewTransport(http.DefaultTransport),
	}
	return nil
}

func TestTraceWithRegion(t *testing.T) {
	if _, present := os.LookupEnv("RUN_TRACE_TEST"); !present {
		log.Info().Msg("Skipping TRACE test!")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	cleanup, _ := tracing.NewTraceExporterHTTP(traceURL, "infra-trace", traceAttribs)
	if cleanup != nil {
		defer func() {
			err := cleanup(context.Background())
			if err != nil {
				log.Err(err).Msg("error cleanup")
			}
		}()
		log.Info().Msg("Tracing enabled")
	} else {
		log.Info().Msg("Tracing disabled")
	}

	ctx = tracing.StartTrace(ctx, "infra-trace", "trace-test")
	// Always stop a trace that was started
	defer tracing.StopTrace(ctx)

	log := log.TraceCtx(ctx)

	log.Info().Msgf("Begin trace region tests")

	apiClient, err := api.NewClientWithResponses(*apiUrl, setTransport)
	require.NoError(t, err)

	r1 := CreateRegion(t, ctx, apiClient, utils.Region1Request)

	r2 := CreateRegion(t, ctx, apiClient, utils.Region2Request)

	utils.Site1Request.RegionId = r1.JSON201.RegionID
	s1 := CreateSite(t, ctx, apiClient, utils.Site1Request)

	utils.Site2Request.RegionId = r2.JSON201.RegionID
	s2 := CreateSite(t, ctx, apiClient, utils.Site2Request)

	regionsGet, err := apiClient.GetRegionsWithResponse(
		ctx,
		&api.GetRegionsParams{},
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, regionsGet.StatusCode())

	sites1, err := apiClient.GetRegionsRegionIDWithResponse(
		ctx,
		*r1.JSON201.RegionID,
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, sites1.StatusCode())

	sites2, err := apiClient.GetRegionsRegionIDWithResponse(
		ctx,
		*r2.JSON201.RegionID,
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, sites2.StatusCode())

	s1res, err := apiClient.GetSitesSiteIDWithResponse(
		ctx,
		*s1.JSON201.SiteID,
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, s1res.StatusCode())

	s2res, err := apiClient.GetSitesSiteIDWithResponse(
		ctx,
		*s2.JSON201.SiteID,
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, s2res.StatusCode())

	log.Info().Msgf("End trace region tests")

	// Give the exporter enough time to send the trace to the server
	time.Sleep(10 * time.Second)
}

func TestTraceWithHost(t *testing.T) {
	if _, present := os.LookupEnv("RUN_TRACE_TEST"); !present {
		log.Info().Msg("Skipping TRACE test!")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	cleanup, _ := tracing.NewTraceExporterHTTP(traceURL, "infra-trace", traceAttribs)
	if cleanup != nil {
		defer func() {
			err := cleanup(context.Background())
			if err != nil {
				log.Err(err).Msg("error cleanup")
			}
		}()
		log.Info().Msg("Tracing enabled")
	} else {
		log.Info().Msg("Tracing disabled")
	}

	ctx = tracing.StartTrace(ctx, "infra-trace", "trace-test")
	// Always stop a trace that was started
	defer tracing.StopTrace(ctx)

	log := log.TraceCtx(ctx)

	log.Info().Msgf("Begin trace host tests")

	apiClient, err := api.NewClientWithResponses(*apiUrl, setTransport)
	require.NoError(t, err)

	resList, err := apiClient.GetComputeHostsWithResponse(
		ctx,
		&api.GetComputeHostsParams{},
		AddJWTtoTheHeader,
		AddProjectIDtoTheHeader,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resList.StatusCode())

	log.Info().Msgf("End trace host tests")

	// Give the exporter enough time to send the trace to the server
	time.Sleep(10 * time.Second)
}
