// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package configuration

import (
	"encoding/json"
	"io"
	"os"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	inv_v1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/inventory/v1"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/errors"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/logging"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/util/collections"
)

var log = logging.GetLogger("resource-loader")

func NewInitResourcesProvider(path string) (InitResourcesProvider, error) {
	resources, err := Load(path)
	if err != nil {
		return nil, err
	}
	// Might be expensive but doing it once
	log.Debug().Msg("Loaded configurations")
	for _, resource := range resources {
		log.Debug().Msgf("Resource Loaded: %v", resource)
	}
	return &initResourcesDefinitionLoader{resources: resources}, nil
}

type InitResourcesProvider interface {
	Get() []*inv_v1.Resource
}

type initResourcesDefinitionLoader struct {
	resources []*inv_v1.Resource
}

func (i *initResourcesDefinitionLoader) Get() []*inv_v1.Resource {
	return collections.MapSlice[*inv_v1.Resource, *inv_v1.Resource](
		i.resources,
		func(resource *inv_v1.Resource) *inv_v1.Resource {
			res, ok := proto.Clone(resource).(*inv_v1.Resource)
			if !ok {
				log.Error().Msgf("unexpected type for Resource: %T", proto.Clone(resource))
				return nil
			}
			return res
		},
	)
}

// Load - reads inputFilePath and deserializes them into []*inv_v1.Resources.
// inputFilePath contains array of inv resources specified in json format:
//
//	[{
//	 "provider": {
//	   "providerKind": "PROVIDER_KIND_BAREMETAL",
//	   "apiEndpoint": "/endpoi"
//	  }
//	}]
func Load(inputFilePath string) ([]*inv_v1.Resource, error) {
	data, err := readFile(inputFilePath)
	if err != nil {
		log.Err(err).Msg("cannot read configuration file")
		return nil, err
	}
	resources, err := unmarshal(data)
	if err != nil {
		log.Err(err).Msg("cannot unmarshall configuration")
		return nil, err
	}
	return resources, nil
}

func unmarshal(data []byte) ([]*inv_v1.Resource, error) {
	var jsonMessages []json.RawMessage
	if err := json.Unmarshal(data, &jsonMessages); err != nil {
		return nil, errors.Errorf("cannot unmarshal: %s", err)
	}
	resources := make([]*inv_v1.Resource, 0)
	for _, msg := range jsonMessages {
		resource := new(inv_v1.Resource)
		if err := protojson.Unmarshal(msg, resource); err != nil {
			return nil, err
		}
		resources = append(resources, resource)
	}
	return resources, nil
}

func readFile(configFilePath string) ([]byte, error) {
	configFile, err := os.Open(configFilePath)
	if err != nil {
		return nil, err
	}
	bytes, err := io.ReadAll(configFile)
	if err != nil {
		return nil, err
	}
	return bytes, nil
}
