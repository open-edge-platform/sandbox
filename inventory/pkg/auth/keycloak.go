// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/Nerzal/gocloak/v13"
	"github.com/eko/gocache/lib/v4/cache"
	"github.com/eko/gocache/lib/v4/store"
	gocache_store "github.com/eko/gocache/store/go_cache/v4"
	gocache "github.com/patrickmn/go-cache"
	"google.golang.org/grpc/codes"

	inv_errors "github.com/open-edge-platform/infra-core/inventory/v2/pkg/errors"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/flags"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/logging"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/secretprovider"
)

var zlog = logging.GetLogger("OMKeycloakService")

const (
	DefaultKeycloakURL                    = "http://platform-keycloak.orch-platform:8080"
	DefaultKeycloakRealm                  = "master"
	DefaultOnboardingCredentialsSecretKey = "client_secret"
	DefaultENCredentialsPrefix            = "edgenode-"

	EnvNameKeycloakURL                     = "KEYCLOAK_URL"
	EnvNameKeycloakRealm                   = "KEYCLOAK_REALM"
	EnvNameOnboardingManagerClientName     = "ONBOARDING_MANAGER_CLIENT_NAME"
	EnvNameOnboardingCredentialsSecretName = "ONBOARDING_CREDENTIALS_SECRET_NAME"
	EnvNameOnboardingCredentialsSecretKey  = "ONBOARDING_CREDENTIALS_SECRET_KEY"
	EnvNameENCredentialsPrefix             = "EN_CREDENTIALS_PREFIX"
	EnvNameEnableGroupCache                = "ENABLE_GROUP_CACHE"
)

var enM2MSvcAccountTemplate = "%s_Edge-Node-M2M-Service-Account"

var (
	AuthServiceFactory = newKeycloakSecretService
	LoginMethod        = loginKeycloakClient

	keycloakRealm                   string
	onboardingManagerClientName     string
	onboardingCredentialsSecretName string
	onboardingCredentialsSecretKey  string
)

// KeycloakAPI wraps Keycloak under interface to enable mocking for unit testing.
//
//go:generate mockgen -package mocks -destination=../mocks/keycloak_mock.go . KeycloakAPI
type KeycloakAPI interface {
	CreateClient(ctx context.Context, accessToken, realm string, newClient gocloak.Client) (string, error)
	GetClientSecret(ctx context.Context, token, realm, idOfClient string) (*gocloak.CredentialRepresentation, error)
	GetClients(ctx context.Context, token, realm string, params gocloak.GetClientsParams) ([]*gocloak.Client, error)
	GetUsers(ctx context.Context, token, realm string, params gocloak.GetUsersParams) ([]*gocloak.User, error)
	GetRoleMappingByUserID(ctx context.Context, token, realm, userID string) (*gocloak.MappingsRepresentation, error)
	AddUserToGroup(ctx context.Context, token, realm, userID, groupID string) error
	DeleteClient(ctx context.Context, token, realm, idOfClient string) error
	Logout(ctx context.Context, clientID, clientSecret, realm, refreshToken string) error
	GetGroups(ctx context.Context, token, realm string, params gocloak.GetGroupsParams) ([]*gocloak.Group, error)
}

type keycloakAPI struct {
	keycloakClient *gocloak.GoCloak
}

func (k keycloakAPI) CreateClient(ctx context.Context, accessToken, realm string, newClient gocloak.Client) (string, error) {
	return k.keycloakClient.CreateClient(ctx, accessToken, realm, newClient)
}

func (k keycloakAPI) GetClientSecret(ctx context.Context, token, realm, idOfClient string,
) (*gocloak.CredentialRepresentation, error) {
	return k.keycloakClient.GetClientSecret(ctx, token, realm, idOfClient)
}

func (k keycloakAPI) GetClients(ctx context.Context, token, realm string, params gocloak.GetClientsParams,
) ([]*gocloak.Client, error) {
	return k.keycloakClient.GetClients(ctx, token, realm, params)
}

func (k keycloakAPI) GetUsers(ctx context.Context, token, realm string, params gocloak.GetUsersParams) ([]*gocloak.User, error) {
	return k.keycloakClient.GetUsers(ctx, token, realm, params)
}

func (k keycloakAPI) GetRoleMappingByUserID(ctx context.Context, token, realm, userID string,
) (*gocloak.MappingsRepresentation, error) {
	return k.keycloakClient.GetRoleMappingByUserID(ctx, token, realm, userID)
}

func (k keycloakAPI) AddUserToGroup(ctx context.Context, token, realm, userID, groupID string) error {
	return k.keycloakClient.AddUserToGroup(ctx, token, realm, userID, groupID)
}

func (k keycloakAPI) DeleteClient(ctx context.Context, token, realm, idOfClient string) error {
	return k.keycloakClient.DeleteClient(ctx, token, realm, idOfClient)
}

func (k keycloakAPI) Logout(ctx context.Context, clientID, clientSecret, realm, refreshToken string) error {
	return k.keycloakClient.Logout(ctx, clientID, clientSecret, realm, refreshToken)
}

func (k keycloakAPI) GetGroups(ctx context.Context, token, realm string, params gocloak.GetGroupsParams) (
	[]*gocloak.Group, error,
) {
	return k.keycloakClient.GetGroups(ctx, token, realm, params)
}

type keycloakService struct {
	keycloakClient   KeycloakAPI
	jwtToken         *gocloak.JWT
	enableGroupCache bool
	groupIDCache     *cache.Cache[[]byte]
}

func newKeycloakSecretService(ctx context.Context) (AuthService, error) {
	kss := &keycloakService{}

	keycloakRealm = os.Getenv(EnvNameKeycloakRealm)
	if keycloakRealm == "" {
		zlog.InfraSec().Warn().Msgf("%s env variable is not set, using default value", EnvNameKeycloakRealm)
		keycloakRealm = DefaultKeycloakRealm
	}

	onboardingManagerClientName = os.Getenv(EnvNameOnboardingManagerClientName)
	if onboardingManagerClientName == "" {
		invErr := inv_errors.Errorf("%s env variable is not set", EnvNameOnboardingManagerClientName)
		zlog.InfraSec().Err(invErr).Msg("")
		return nil, invErr
	}

	onboardingCredentialsSecretName = os.Getenv(EnvNameOnboardingCredentialsSecretName)
	if onboardingCredentialsSecretName == "" {
		invErr := inv_errors.Errorf("%s env variable is not set", EnvNameOnboardingCredentialsSecretName)
		zlog.InfraSec().Err(invErr).Msg("")
		return nil, invErr
	}

	onboardingCredentialsSecretKey = os.Getenv(EnvNameOnboardingCredentialsSecretKey)
	if onboardingCredentialsSecretKey == "" {
		zlog.InfraSec().Warn().Msgf("%s env variable is not set, using default value", EnvNameOnboardingCredentialsSecretKey)
		onboardingCredentialsSecretKey = DefaultOnboardingCredentialsSecretKey
	}

	keycloakURL := os.Getenv(EnvNameKeycloakURL)
	if keycloakURL == "" {
		zlog.InfraSec().Warn().Msgf("%s env variable is not set, using default value", EnvNameKeycloakURL)
		keycloakURL = DefaultKeycloakURL
	}

	// Setup cache for Roles, to avoid reaching out to Keycloak each time.
	kss.groupIDCache = setupGroupCache()

	_, kss.enableGroupCache = os.LookupEnv(EnvNameEnableGroupCache)

	err := kss.login(ctx, keycloakURL)
	if err != nil {
		return nil, err
	}

	return kss, nil
}

func setupGroupCache() *cache.Cache[[]byte] {
	const cacheExpiryTime = 5 * time.Minute
	const cacheCleanupTime = 10 * time.Minute
	gocacheClient := gocache.New(cacheExpiryTime, cacheCleanupTime)
	gocacheStore := gocache_store.NewGoCache(gocacheClient)
	return cache.New[[]byte](gocacheStore)
}

func loginKeycloakClient(ctx context.Context, keycloakURL string) (*gocloak.GoCloak, *gocloak.JWT, error) {
	client := gocloak.NewClient(keycloakURL)

	jwtToken, err := client.LoginClient(ctx, onboardingManagerClientName,
		secretprovider.GetSecret(onboardingCredentialsSecretName, onboardingCredentialsSecretKey), keycloakRealm)

	return client, jwtToken, err
}

func (k *keycloakService) login(ctx context.Context, keycloakURL string) error {
	client, jwtToken, err := LoginMethod(ctx, keycloakURL)
	if err != nil {
		errMsg := fmt.Sprintf("Failed to login to Keycloak %s", keycloakURL)
		zlog.InfraSec().Err(err).Msg(errMsg)
		return inv_errors.Errorf("%s", errMsg)
	}

	k.keycloakClient = &keycloakAPI{client}
	k.jwtToken = jwtToken

	zlog.InfraSec().Debug().Msgf("Keycloak client logged in successfully")
	return nil
}

func (k *keycloakService) getServiceAccountUserIDByClientName(ctx context.Context, clientName string) (string, error) {
	zlog.Debug().Msgf("Getting Keycloak service account user ID for client %s", clientName)

	serviceAccountName := fmt.Sprintf("service-account-%s", clientName)

	svcAccountUsers, err := k.keycloakClient.GetUsers(ctx, k.jwtToken.AccessToken, keycloakRealm, gocloak.GetUsersParams{
		Username: &serviceAccountName,
	})
	if err != nil {
		errMsg := fmt.Sprintf("Cannot retrieve Keycloak service account user %s", serviceAccountName)
		zlog.InfraSec().Err(err).Msg(errMsg)
		return "", inv_errors.Errorf("%s", errMsg)
	}

	if len(svcAccountUsers) == 0 {
		invErr := inv_errors.Errorfc(
			codes.NotFound, "No Keycloak service account user found with username %s", serviceAccountName)
		zlog.InfraSec().Err(invErr).Msg("")
		return "", invErr
	}

	// This should never happen but we could have more than one Keycloak user with the same username.
	// We print warning and get first.
	if len(svcAccountUsers) > 1 {
		zlog.Warn().Msgf(
			"More than one Keycloak service account user found for username %s, getting first one", serviceAccountName)
	}

	svcAccountUserID := *svcAccountUsers[0].ID
	return svcAccountUserID, nil
}

func (k *keycloakService) addGroupToEdgeNodeClient(ctx context.Context, tenantID, enClientID string) error {
	zlog.Debug().Msgf("Adding default group for edge node client [tenantID=%s, clientID=%s, groupName=%s]",
		tenantID, enClientID, enM2MSvcAccountTemplate)

	// service account should be automatically created when the client is created.
	enClientSvcAccountUserID, err := k.getServiceAccountUserIDByClientName(ctx, enClientID)
	if err != nil {
		errMsg := fmt.Sprintf("Failed to get Keycloak service account user ID for client %s", enClientID)
		zlog.InfraSec().Err(err).Msg(errMsg)
		return inv_errors.Errorf("%s", errMsg)
	}

	enM2MGroupName := fmt.Sprintf(enM2MSvcAccountTemplate, tenantID)
	enM2MGroupID, err := k.getGroupIDByName(ctx, enM2MGroupName)
	if err != nil {
		return err
	}

	err = k.keycloakClient.AddUserToGroup(ctx, k.jwtToken.AccessToken, keycloakRealm, enClientSvcAccountUserID, enM2MGroupID)
	if err != nil {
		errMsg := fmt.Sprintf("Cannot add group %s to service account user %s for client %s",
			enM2MGroupID, enClientSvcAccountUserID, enClientID)
		zlog.InfraSec().Err(err).Msg(errMsg)
		return inv_errors.Errorf("%s", errMsg)
	}

	return nil
}

func (k *keycloakService) getGroupIDByName(ctx context.Context, groupName string) (string, error) {
	if k.enableGroupCache {
		groupID, err := k.groupIDCache.Get(ctx, groupName)
		if err == nil {
			groupIDString := string(groupID)
			// found in cache. Assumption is that groups won't change in Keycloak
			zlog.InfraSec().Debug().Msgf("GroupID found in cache: groupName=%s, groupID=%s", groupName, groupIDString)
			return groupIDString, nil
		}
		// check if error is store.NotFound
		var notFound *store.NotFound
		if errors.As(err, &notFound) {
			zlog.InfraSec().Debug().Msgf("GroupID not found in cache: groupName=%s", groupName)
		} else {
			// Not found or other errors in the cache
			zlog.InfraSec().Warn().Err(err).
				Msgf("Error when getting the groupID from cache, fallback to keycloak: groupName=%s", groupName)
		}
	}
	groups, err := k.keycloakClient.GetGroups(ctx, k.jwtToken.AccessToken, keycloakRealm, gocloak.GetGroupsParams{
		Search: &groupName,
	})
	if err != nil {
		errMsg := fmt.Sprintf("Failed to get Keycloak group %s", groupName)
		zlog.InfraSec().Err(err).Msg(errMsg)
		return "", inv_errors.Errorf("%s", errMsg)
	}

	if len(groups) == 0 {
		errMsg := fmt.Sprintf("No Keycloak group found for %s", groupName)
		zlog.InfraSec().Err(err).Msg(errMsg)
		return "", inv_errors.Errorfc(codes.NotFound, "%s", errMsg)
	}

	// This should never happen but we could have more than one group with the same name.
	// We print warning and get first.
	if len(groups) > 1 {
		zlog.Warn().Msgf("More than one Keycloak group found for %s, getting first one", groupName)
	}
	groupID := *groups[0].ID

	if k.enableGroupCache {
		err = k.groupIDCache.Set(ctx, groupName, []byte(groupID))
		if err != nil {
			zlog.InfraSec().Err(err).Msgf("Error when storing the groupID in cache, continuing: groupName=%s", groupName)
		}
	}

	return groupID, nil
}

func (k *keycloakService) CreateCredentialsWithUUID(ctx context.Context, tenantID, uuid string) (string, string, error) {
	edgeNodeClient := getEdgeNodeClientFromTemplate(tenantID, uuid)

	zlog.Info().Msgf("Creating Keycloak credentials for host [tenantID=%s, UUID=%s]", tenantID, uuid)

	id, err := k.keycloakClient.CreateClient(ctx, k.jwtToken.AccessToken, keycloakRealm, edgeNodeClient)
	if err != nil {
		errMsg := fmt.Sprintf("Failed to create Keycloak client with [tenantID=%s, UUID=%s]", tenantID, uuid)
		zlog.InfraSec().Err(err).Msg(errMsg)
		return "", "", inv_errors.Errorf("%s", errMsg)
	}

	zlog.InfraSec().Debug().Msgf("Keycloak credentials for host [tenantID=%s, UUID=%s] created successfully, ID: %s",
		tenantID, uuid, id)

	err = k.addGroupToEdgeNodeClient(ctx, tenantID, *edgeNodeClient.ClientID)
	if err != nil {
		errMsg := fmt.Sprintf("Failed to add default client roles for host [tenantID=%s, UUID=%s]", tenantID, uuid)
		zlog.InfraSec().Err(err).Msg(errMsg)
		return "", "", inv_errors.Errorf("%s", errMsg)
	}

	creds, err := k.keycloakClient.GetClientSecret(ctx, k.jwtToken.AccessToken, keycloakRealm, id)
	if err != nil {
		errMsg := fmt.Sprintf("Failed to get Keycloak client secret for client ID %s (host [tenantID=%s, UUID=%s])",
			id, tenantID, uuid)
		zlog.InfraSec().Err(err).Msg(errMsg)
		return "", "", inv_errors.Errorf("%s", errMsg)
	}

	if creds.Value == nil {
		err = inv_errors.Errorf("Received empty client secret for client ID %s (host [tenantID=%s, UUID=%s])",
			id, tenantID, uuid)
		zlog.InfraSec().Err(err).Msg("")
		return "", "", err
	}

	zlog.InfraSec().Debug().Msgf("Keycloak client secret for host [tenantID=%s, UUID=%s] obtained successfully, ID: %s",
		tenantID, uuid, id)

	return *edgeNodeClient.ClientID, *creds.Value, nil
}

func (k *keycloakService) GetCredentialsByUUID(ctx context.Context, tenantID, uuid string) (string, string, error) {
	edgeNodeClientID := getEdgenodeClientName(uuid)

	zlog.Info().Msgf("Getting Keycloak credentials for host [tenantID=%s, UUID=%s]", tenantID, uuid)

	clients, err := k.keycloakClient.GetClients(ctx, k.jwtToken.AccessToken, keycloakRealm, gocloak.GetClientsParams{
		ClientID: &edgeNodeClientID,
	})
	if err != nil {
		errMsg := fmt.Sprintf("Keycloak client for edge node by [tenantID=%s, UUID=%s] does not exist",
			tenantID, uuid)
		zlog.InfraSec().Err(err).Msg(errMsg)
		return "", "", inv_errors.Errorf("%s", errMsg)
	}

	if len(clients) == 0 {
		errMsg := fmt.Sprintf("No Keycloak clients found for [tenantID=%s, UUID=%s]", tenantID, uuid)
		zlog.InfraSec().Err(err).Msg(errMsg)
		return "", "", inv_errors.Errorfc(codes.NotFound, "%s", errMsg)
	}

	// This should never happen but we could have more than one Keycloak client for a UUID.
	// We print warning and get first.
	if len(clients) > 1 {
		zlog.Warn().Msgf("More than one Keycloak client found for [tenantID=%s, UUID=%s], getting first one",
			tenantID, uuid)
	}

	secret := clients[0].Secret
	// if we received secret as part of GetClients(), return it. Otherwise, use GetClientSecret().
	if secret != nil {
		return edgeNodeClientID, *secret, nil
	}

	id := *clients[0].ID
	creds, err := k.keycloakClient.GetClientSecret(ctx, k.jwtToken.AccessToken, keycloakRealm, id)
	if err != nil {
		errMsg := fmt.Sprintf("Failed to get Keycloak client secret for client ID %s (host [tenantID=%s, UUID=%s])",
			id, tenantID, uuid)
		zlog.InfraSec().Err(err).Msg(errMsg)
		return "", "", inv_errors.Errorf("%s", errMsg)
	}

	if creds.Value == nil {
		err = inv_errors.Errorf("Received empty client secret for client ID %s (host [tenantID=%s, UUID=%s])",
			id, tenantID, uuid)
		zlog.InfraSec().Err(err).Msg("")
		return "", "", err
	}

	return edgeNodeClientID, *creds.Value, nil
}

func (k *keycloakService) RevokeCredentialsByUUID(ctx context.Context, tenantID, uuid string) error {
	edgeNodeClientID := getEdgenodeClientName(uuid)

	clients, err := k.keycloakClient.GetClients(ctx, k.jwtToken.AccessToken, keycloakRealm, gocloak.GetClientsParams{
		ClientID: &edgeNodeClientID,
	})
	if err != nil {
		errMsg := fmt.Sprintf("Keycloak client for edge node by [tenantID=%s, UUID=%s] does not exist",
			tenantID, uuid)
		zlog.InfraSec().Err(err).Msg(errMsg)
		return inv_errors.Errorf("%s", errMsg)
	}

	if len(clients) == 0 {
		errMsg := fmt.Sprintf("No Keycloak clients found for [tenantID=%s, UUID=%s]", tenantID, uuid)
		zlog.InfraSec().Err(err).Msg(errMsg)
		return inv_errors.Errorfc(codes.NotFound, "%s", errMsg)
	}

	// This should never happen but we could have more than one Keycloak client for a UUID.
	// We print warning and remove all clients.
	if len(clients) > 1 {
		zlog.Warn().Msgf("More than one Keycloak client found for [tenantID=%s, UUID=%s], deleting all",
			tenantID, uuid)
	}

	for _, edgeNodeClient := range clients {
		if edgeNodeClient.ID == nil {
			zlog.Debug().Msgf("Found Keycloak client for [tenantID=%s, UUID=%s] with empty ID, skipping deletion",
				tenantID, uuid)
			continue
		}

		err = k.keycloakClient.DeleteClient(ctx, k.jwtToken.AccessToken, keycloakRealm, *edgeNodeClient.ID)
		if err != nil {
			errMsg := fmt.Sprintf("Failed to delete Keycloak client for edge node with [tenantID=%s, UUID=%s]",
				tenantID, uuid)
			zlog.InfraSec().Err(err).Msg(errMsg)
			return inv_errors.Errorf("%s", errMsg)
		}

		zlog.InfraSec().Debug().Msgf("Keycloak credentials for host [tenantID=%s, UUID=%s] revoked successfully, ID: %s",
			tenantID, uuid, *edgeNodeClient.ID)
	}

	return nil
}

func (k *keycloakService) Logout(ctx context.Context) {
	// refresh_token is required to logout but it's not provided for all Keycloak clients.
	// Skip logging out if refresh_token is not provided.
	if k.jwtToken.RefreshToken == "" {
		return
	}
	if err := k.keycloakClient.Logout(ctx, onboardingManagerClientName,
		secretprovider.GetSecret(onboardingCredentialsSecretName, onboardingCredentialsSecretKey),
		keycloakRealm, k.jwtToken.RefreshToken); err != nil {
		zlog.InfraSec().Err(err).Msgf("Failed to logout from Keycloak")
		return
	}
}

func getEdgenodeClientName(uuid string) string {
	enCredentialsPrefix := os.Getenv(EnvNameENCredentialsPrefix)
	if enCredentialsPrefix == "" {
		zlog.InfraSec().Warn().Msgf("%s env variable is not set, using default value", EnvNameENCredentialsPrefix)
		enCredentialsPrefix = DefaultENCredentialsPrefix
	}

	return fmt.Sprintf("%s%s", enCredentialsPrefix, uuid)
}

func getEdgeNodeClientFromTemplate(tenantID, uuid string) gocloak.Client {
	description := fmt.Sprintf("Client to use by Edge Node [tenantID=%s, UUID=%s], created by Onboarding Manager at %s",
		tenantID, uuid, time.Now().UTC().String())
	clientID := getEdgenodeClientName(uuid)
	name := fmt.Sprintf("Edge Node [tenantID=%s, UUID=%s]", tenantID, uuid)
	authTypeClientSecret := "client-secret"
	protocolOpenidConnect := "openid-connect"
	boolTrue := true
	boolFalse := false
	zero := int32(0)
	attributes := map[string]string{
		"oidc.ciba.grant.enabled":                   "false",
		"oauth2.device.authorization.grant.enabled": "false",
		"backchannel.logout.revoke.offline.tokens":  "false",
	}
	defaultClientScopes := []string{
		"web-origins",
		"acr",
		"profile",
		"roles",
		"email",
	}
	optionalClientScopes := []string{
		"address",
		"phone",
		"offline_access",
		"microprofile-jwt",
	}
	return gocloak.Client{
		ClientID:                  &clientID,
		Name:                      &name,
		Description:               &description,
		SurrogateAuthRequired:     &boolFalse,
		Enabled:                   &boolTrue,
		ClientAuthenticatorType:   &authTypeClientSecret,
		NotBefore:                 &zero,
		BearerOnly:                &boolFalse,
		ConsentRequired:           &boolFalse,
		StandardFlowEnabled:       &boolFalse,
		ImplicitFlowEnabled:       &boolFalse,
		DirectAccessGrantsEnabled: &boolFalse,
		ServiceAccountsEnabled:    &boolTrue,
		PublicClient:              &boolFalse,
		Protocol:                  &protocolOpenidConnect,
		Attributes:                &attributes,
		FullScopeAllowed:          &boolTrue,
		DefaultClientScopes:       &defaultClientScopes,
		OptionalClientScopes:      &optionalClientScopes,
	}
}

func RevokeHostCredentials(ctx context.Context, tenantID, uuID string) error {
	if *flags.FlagDisableCredentialsManagement {
		zlog.Warn().Msgf("disableCredentialsManagement flag is set to true, " +
			"skip credentials revocation")
		return nil
	}
	authService, err := AuthServiceFactory(ctx)
	if err != nil {
		return err
	}
	defer authService.Logout(ctx)

	revokeErr := authService.RevokeCredentialsByUUID(ctx, tenantID, uuID)
	if revokeErr != nil && !inv_errors.IsNotFound(revokeErr) {
		zlog.InfraSec().InfraError("Failed to revoke credentials for Host [tenantID=%s, UUID=%s]].", tenantID, uuID).
			Msg("RevokeHostCredentials")
		return inv_errors.Wrap(revokeErr)
	}

	return nil
}
