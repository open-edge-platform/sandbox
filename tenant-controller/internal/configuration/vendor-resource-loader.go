// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package configuration

import (
	"context"
	"encoding/json"
	"reflect"

	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/proto"

	inv_v1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/inventory/v1"
	providerv1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/provider/v1"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/errors"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/flags"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/providerconfiguration"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/secrets"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/util/collections"
)

type LOCAConfig struct {
	Name        string `json:"name"`
	Username    string `json:"username"`
	Password    string `json:"password"`
	APIEndpoint string `json:"api_endpoint"`
	InstanceTpl string `json:"instance_tpl"`
	DNSDomain   string `json:"dns_domain"`
}

func NewLenovoInitResourcesDefinitionLoader(path string) (InitResourcesProvider, error) {
	var vaultS secrets.SecretsService
	var err error
	// init vault client
	if !*flags.FlagDisableCredentialsManagement {
		ctx := context.Background()
		vaultS, err = secrets.SecretServiceFactory(ctx)
		if err != nil {
			return nil, err
		}
		defer vaultS.Logout(ctx)
	}
	resources := make([]*inv_v1.Resource, 0)
	// Load the loca configurations
	locaConfigs, err := LoadLOCAConfigs(path)
	if err != nil {
		return nil, err
	}
	// iterate over the loca configurations
	//   convert loca config into actual provider
	//   check if the secret exists
	//   write the secret if needed
	//   append the provider into a list of resources
	for _, locaConfig := range locaConfigs {
		provider, err := locaConfigToProvider(locaConfig)
		if err != nil {
			log.Err(err).Msgf("Ignoring loca configuration: %+v", locaConfig)
			continue
		}
		if !*flags.FlagDisableCredentialsManagement {
			err = handleVaultSecret(vaultS, provider, locaConfig)
			if err != nil {
				log.InfraErr(err).Msgf("error while processing secret for %s provider", locaConfig.Name)
			}
		}
		resources = append(resources, provider)
	}
	// Might be expensive but doing it once
	log.Debug().Msg("Loaded Vendor configurations")
	for _, resource := range resources {
		log.Debug().Msgf("%v", resource)
	}
	return &lenovoInitResourcesDefinitionLoader{resources: resources}, nil
}

func handleVaultSecret(vaultS secrets.SecretsService, resource *inv_v1.Resource, locaConfig LOCAConfig) error {
	ctx := context.Background()
	provider := resource.GetProvider()
	if provider == nil {
		return errors.Errorf("Provider is invalid")
	}
	// We expect only credential - and it will contain both LOC-A user and password!
	for _, credential := range provider.ApiCredentials {
		data := map[string]interface{}{
			"username": locaConfig.Username,
			"password": locaConfig.Password,
		}
		existingSecret, err := vaultS.ReadSecret(ctx, credential)
		if err != nil && !errors.IsNotFound(err) {
			log.Warn().Msgf("Unable to read secret %s due to %v", credential, err)
			continue
		}
		if existingSecret != nil && reflect.DeepEqual(existingSecret, data) {
			log.Warn().Msgf("Secret %s already exist...skipping", credential)
			continue
		}
		newSecret := map[string]interface{}{
			"data": data,
		}
		_, err = vaultS.WriteSecret(ctx, credential, newSecret)
		if err != nil {
			log.Warn().Msgf("Unable to write secret %s...skipping", credential)
			continue
		}
	}
	return nil
}

type lenovoInitResourcesDefinitionLoader struct {
	resources []*inv_v1.Resource
}

func (i *lenovoInitResourcesDefinitionLoader) Get() []*inv_v1.Resource {
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

// LoadLOCAConfigs - reads inputFilePath and deserializes them into []LOCAConfig.
// inputFilePath contains array of inv resources specified in json format:
//
//	[{
//	    "apiEndpoint": "https://sc.loca1.lab/api/v1",
//	    "name": "LOCA1",
//	    "password": "somethingelse",
//	    "username": "something",
//	    "instance_tpl": "intel{{#}}",
//	    "dns_domain": "something",
//	 }]
func LoadLOCAConfigs(inputFilePath string) ([]LOCAConfig, error) {
	data, err := readFile(inputFilePath)
	if err != nil {
		log.Err(err).Msg("cannot read configuration file")
		return nil, err
	}
	locaConfigs, err := unmarshalLOCAConfigs(data)
	if err != nil {
		log.Err(err).Msg("cannot unmarshall configuration")
		return nil, err
	}
	return locaConfigs, nil
}

func unmarshalLOCAConfigs(data []byte) ([]LOCAConfig, error) {
	var locaConfigs []LOCAConfig
	if err := json.Unmarshal(data, &locaConfigs); err != nil {
		return nil, errors.Errorf("cannot unmarshal: %s", err)
	}
	return locaConfigs, nil
}

func locaConfigToProvider(locaConfig LOCAConfig) (*inv_v1.Resource, error) {
	providerResource := new(providerv1.ProviderResource)
	if locaConfig.Name == "" || locaConfig.APIEndpoint == "" ||
		locaConfig.Password == "" || locaConfig.Username == "" ||
		locaConfig.InstanceTpl == "" || locaConfig.DNSDomain == "" {
		return nil, errors.Errorfc(codes.InvalidArgument, "locaConfig %v is not fully initialized", locaConfig)
	}
	providerResource.ProviderKind = providerv1.ProviderKind_PROVIDER_KIND_BAREMETAL
	providerResource.ProviderVendor = providerv1.ProviderVendor_PROVIDER_VENDOR_LENOVO_LOCA
	providerResource.Name = locaConfig.Name
	providerResource.ApiEndpoint = locaConfig.APIEndpoint
	providerResource.ApiCredentials = []string{
		locaConfig.Name + "-secret",
	}
	// Serialize the config
	conf := providerconfiguration.LOCAProviderConfig{
		InstanceTpl: locaConfig.InstanceTpl,
		DNSDomain:   locaConfig.DNSDomain,
	}
	confData, err := json.Marshal(&conf)
	if err != nil {
		invErr := errors.Errorfc(codes.InvalidArgument, "Cannot generate ProviderConfig: %v", err)
		log.Error().Err(invErr).Send()
		return nil, invErr
	}
	providerResource.Config = string(confData)

	resource := inv_v1.Resource{
		Resource: &inv_v1.Resource_Provider{
			Provider: providerResource,
		},
	}
	return &resource, nil
}
