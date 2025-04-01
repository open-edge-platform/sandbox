// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package store_test

import (
	"flag"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/goccy/go-json"
	"github.com/mennanov/fmutils"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/exp/maps"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	"github.com/open-edge-platform/infra-core/inventory/v2/internal/store"
	inv_v1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/inventory/v1"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/client"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/errors"
	inv_testing "github.com/open-edge-platform/infra-core/inventory/v2/pkg/testing"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/util"
)

const (
	metaR1 = `[{"key":"key1-test","value":"region_key1_lvl1-test"},{"key":"key2-test","value":"region_key2_lvl1-test"},{"key":"key3-test","value":"region_key3_lvl1-test"}]`
	metaR2 = `[{"key":"key1-test","value":"region_key1_lvl2-test"},{"key":"key2-test","value":"region_key2_lvl2-test"},{"key":"key4-test","value":"region_key4_lvl2-test"}]`
	metaR3 = `[{"key":"key1-test","value":"region_key1_lvl3-test"},{"key":"key5-test","value":"region_key5_lvl3-test"}]`
	metaR5 = `[{"key":"key1-test","value":"region_key1_lvl4-test"}]`

	metaO1 = `[{"key":"key1-test","value":"ou_key1_lvl1-test"},{"key":"key2-test","value":"ou_key2_lvl1-test"},{"key":"key3-test","value":"ou_key3_lvl1-test"}]`
	metaO2 = `[{"key":"key1-test","value":"ou_key1_lvl2-test"},{"key":"key2-test","value":"ou_key2_lvl2-test"},{"key":"key4-test","value":"ou_key4_lvl2-test"}]`
	metaO3 = `[{"key":"key1-test","value":"ou_key1_lvl3-test"},{"key":"key5-test","value":"ou_key5_lvl3-test"}]`
	metaO5 = `[{"key":"key1-test","value":"ou_key1_lvl4-test"}]`

	metaHost1 = `[{"key":"key1-test","value":"host_key1-test"},{"key":"key2-test","value":"host_key2-test"},{"key":"key3-test","value":"host_key3-test"},{"key":"key4-test","value":"host_key4-test"}]`
	metaHost2 = `[{"key":"key1-test","value":"host_key1-test"},{"key":"key2-test","value":"host_key2-test"},{"key":"key3-test","value":"host_key3_mod-test"}]`

	metaDuplicatedKeys = `[{"key":"key1-test","value":"host_key1-test"},{"key":"key1-test","value":"host_key2-test"},{"key":"key3-test","value":"host_key3_mod-test"}]`

	tenantIDZero = "00000000-0000-0000-0000-000000000000"
	tenantIDOne  = "11111111-1111-1111-1111-111111111111"
)

var (
	expPhyMeta1  = `[{"key":"key1-test","value":"region_key1_lvl3-test"},{"key":"key2-test","value":"region_key2_lvl2-test"},{"key":"key3-test","value":"region_key3_lvl1-test"},{"key":"key4-test","value":"region_key4_lvl2-test"},{"key":"key5-test","value":"region_key5_lvl3-test"}]`
	expPhyMeta2  = `[{"key":"key2-test","value":"region_key2_lvl2-test"},{"key":"key3-test","value":"region_key3_lvl1-test"},{"key":"key4-test","value":"region_key4_lvl2-test"},{"key":"key5-test","value":"region_key5_lvl3-test"}]`
	expPhyMeta3  = `[{"key":"key1-test","value":"ou_key1_lvl4-test"},{"key":"key2-test","value":"region_key2_lvl2-test"},{"key":"key3-test","value":"region_key3_lvl1-test"},{"key":"key4-test","value":"region_key4_lvl2-test"},{"key":"key5-test","value":"region_key5_lvl3-test"}]`
	expLogiMeta1 = `[{"key":"key1-test","value":"ou_key1_lvl3-test"},{"key":"key2-test","value":"ou_key2_lvl2-test"},{"key":"key3-test","value":"ou_key3_lvl1-test"},{"key":"key4-test","value":"ou_key4_lvl2-test"},{"key":"key5-test","value":"ou_key5_lvl3-test"}]`
	expLogiMeta2 = `[{"key":"key2-test","value":"ou_key2_lvl2-test"},{"key":"key3-test","value":"ou_key3_lvl1-test"},{"key":"key4-test","value":"ou_key4_lvl2-test"},{"key":"key5-test","value":"ou_key5_lvl3-test"}]`
	emptyString  = ""
	metaO6       = `[{"key":"key10-test","value":"ou_key10_lvl4-test"}]`
)

func TestMain(m *testing.M) {
	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	// Currently unused
	flag.String(
		"policyBundle",
		wd+"/../../out/policy_bundle.tar.gz",
		"Path of policy rego file",
	)
	flag.Parse()
	projectRoot := filepath.Dir(filepath.Dir(wd))

	policyPath := projectRoot + "/out"
	certPath := projectRoot + "/cert/certificates"
	migrationsDir := projectRoot + "/out"

	inv_testing.StartTestingEnvironment(policyPath, certPath, migrationsDir)
	run := m.Run() // run all tests
	inv_testing.StopTestingEnvironment()

	os.Exit(run)
}

// General unit testing helper functions.

// CompareMetadata: compares the given metadata strings to verify if they are equal.
// The metadata are expected to be in the form [{"key":"KEY1","value":"VALUE1"},{"key":"KEY2","value":"VALUE2"}].
func CompareMetadata(t *testing.T, actual, expected string) bool {
	t.Helper()

	if actual == expected {
		return true
	}
	var actualM []store.Metadata
	var expectedM []store.Metadata
	err := json.Unmarshal([]byte(actual), &actualM)
	if err != nil {
		t.Errorf("Failed to unmarshal actual Metadata failed %s", err)
		return false
	}
	log.Info().Msgf("ActualM %v", actualM)
	err = json.Unmarshal([]byte(expected), &expectedM)
	if err != nil {
		t.Errorf("Failed to unmarshal actual Metadata failed %s", err)
		return false
	}
	log.Info().Msgf("ExpectedM %v", expectedM)
	actualMetaMap, err := store.MetadataToMetaMap(actualM)
	if err != nil {
		t.Errorf("Failed to convert actual to Meta Map %s", err)
		return false
	}
	expectedMetaMap, err := store.MetadataToMetaMap(expectedM)
	if err != nil {
		t.Errorf("Failed to convert expected to Meta Map %s", err)
		return false
	}
	log.Info().Msgf("Actual %v", actualMetaMap)
	log.Info().Msgf("Expected: %v", expectedMetaMap)
	return maps.Equal[map[string]string, map[string]string](actualMetaMap, expectedMetaMap)
}

// asserStrongRelationError: asserts that the given error is due to failing removing entity with strong relation.
func assertStrongRelationError(t *testing.T, err error, expString string) {
	t.Helper()

	require.Error(t, err)
	s, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.FailedPrecondition, s.Code())
	assert.Contains(t, errors.ErrorToStringWithDetails(err), expString)
}

// Verifies that the expected event was received as the last event in the provided event channel.
func assertReceiveEvent(
	t *testing.T,
	eventChan chan *client.WatchEvents,
	expectedEventKind inv_v1.SubscribeEventsResponse_EventKind,
	expectedResKind inv_v1.ResourceKind,
	expectedResID string,
) {
	t.Helper()
	select {
	case ev, ok := <-eventChan:
		require.True(t, ok, "No events received")
		kind, err := util.GetResourceKindFromResourceID(ev.Event.ResourceId)
		require.NoError(t, err, "resource manager did receive a strange event", ev.Event.ResourceId)
		assert.Equal(t, expectedEventKind, ev.Event.EventKind, "Wrong event kind")
		assert.Equal(t, expectedResKind, kind, "Wrong resource kind")
		assert.Equal(t, expectedResID, ev.Event.ResourceId, "Wrong resource ID")
		assert.Equal(t, util.GetResourceKindFromResource(ev.Event.Resource), kind, "Resource kinds not equal")
		id, err := util.GetResourceIDFromResource(ev.Event.Resource)
		require.NoError(t, err, "Malformed resource in event", ev.Event.Resource)
		assert.Equal(t, id, ev.Event.ResourceId, "resource IDs in event do not match")
	case <-time.After(1 * time.Second):
		// Timeout to avoid waiting events indefinitely
		t.Fatalf("No events received within timeout")
	}
}

func assertSameResource(t *testing.T, expected, actual *inv_v1.Resource, fieldMask *fieldmaskpb.FieldMask) {
	t.Helper()
	eRes, err := util.GetSetResource(expected)
	require.NoError(t, err)
	expRes := proto.Clone(eRes)
	aRes, err := util.GetSetResource(actual)
	require.NoError(t, err)
	actRes := proto.Clone(aRes)

	if fieldMask != nil {
		// Apply the mask in order to make successful comparison
		fmutils.Filter(expRes, fieldMask.GetPaths())
		fmutils.Filter(actRes, fieldMask.GetPaths())
	}

	if eq, diff := inv_testing.ProtoEqualOrDiff(expRes, actRes); !eq {
		t.Errorf("Create/UpdateResource did not correctly create/update the resource: %v", diff)
	}
}
