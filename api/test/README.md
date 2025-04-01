# Integration Tests

TODO: UPDATE GUIDE
This file describes how to execute the integration tests placed in this folder.

> These tests do not delete created Hosts from the Inventory, which means that some tests may fail, if you run them two
times in a row.
> To avoid that, please either re-deploy API and Inventory, or use `hostcli` to delete duplicated/unnecessary hosts

## Requirements

It requires helm and kubectl installed, besides an operational Kubernetes cluster.

## Run

Clone and deploy the Edge Infrastructure Manager chart, with api, topology and inventory
components enabled, check the values.yaml file or use --set flags.

> `Authentication` functionality is disabled by default, don't forget to enable it in the charts.

```bash
  git clone https://github.com/open-edge-platform/infra-charts
  cd infra-charts
  make deps
  kubectl create ns orch-infra
  helm -n orch-infra install --set api.oidc.oidc_server_url="http://platform-keycloak.orch-platform:8080/realms/master" --set inventory.oidc.oidc_server_url="http://platform-keycloak.orch-platform:8080/realms/master" infra-core ./infra-core
```

Deploy a Keycloak instance with all dependencies:

```bash
  TODO: add instructions
```

Wait until all pods are in running state.

```bash
  kubectl -n orch-infra get pods
  kubectl -n orch-platform get pods
```

Then enable a port-forward to have interface with the API component via port 8080.

```bash
  kubectl -n orch-infra port-forward svc/api 8080
```

Enable port-forwarding for the `Keycloak`.

```bash
  kubectl -n orch-platform port-forward service/platform-keycloak 8090:8080
```

Credentials to access `Keycloak` can be found in `values.yaml` of a `Keycloak` deployment (they usually are `admin/ChangeMeOn1stLogin!`),

Now, you are almost good to go. In another terminal run the following command to obtain a valid JWT:

```bash
  curl --location --request POST 'http://localhost:8090/realms/master/protocol/openid-connect/token' \
  --header 'Content-Type: application/x-www-form-urlencoded' \
  --data-urlencode 'grant_type=password' \
  --data-urlencode 'client_id=system-client' \
  --data-urlencode 'username=admin' \
  --data-urlencode 'password=ChangeMeOn1stLogin!' \
--data-urlencode 'scope=openid profile email groups'
```

Extract from the response an `access_token` field and export it as an environmental variable `JWT_TOKEN` in your terminal:

```bash
export JWT_TOKEN=jwt-token-obtained-from-keycloak
```

In the same terminal run the tests:

```bash
go test -v ./test/client/  

# To run a specific test, define the -run flag
# go test -v ./test/client/ -run TestComputeHost 
```

## Trace Test Case

This test case is for demonstration of features only.
It must not be used as a routine test case for API.

The Edge Infrastructure Manager chart components must be deployed with the following values set:

```yaml
  traceURL: "observability-opentelemetry-collector.orch-platform.svc:4318"
  enableTracing: true
```

This test case presents how a client can be enabled to
add its tracing information inside the requests done to
the API component.

To make it work, it requires that at least Edge Infrastructure API and
inventory components are deployed, together with some
observability parts of Edge Infrastructure Manager deployment, listed below.

```yaml
- 02-mp-prometheus.yaml
- 11-mp-kube-prometheus-stack.yaml
- 12-mp-observability.yaml
```

After the deployment of those charts it requires that
port-forward is enabled for the following services.

```bash
kubectl -n orch-infra port-forward svc/api 8080
kubectl -n orch-platform port-forward svc/observability-opentelemetry-collector 4318
```

The test can be executed with the following command

```bash
go test -v -count=1 ./test/client -run TestTraceWithRegion
```

After the execution it is possible to visualizing the
traces in the grafana explore section, selecting the `Tempo` source.
Log into the grafana (e.g., `https://observability-admin.kind.internal/`) by using the credentials:

```bash
user: admin
password: kubectl -n orch-platform get secret kube-prometheus-stack-grafana-admin  -o jsonpath="{.data.admin-password}" | base64 --decode ; echo
```

In the logs presented in the output of the test case
it should contain a `trace_id` field, copy and paste
that value in the tempo search for a trace-id.
After running the query it should display all the
calls done from the client and its related spans in
api and inventory components.

## Run test with Edge Infrastructure Manager deployment

Download Edge Infrastructure Manager deployment repo.

To deploy minimal Edge Infrastructure environment:

```bash
  source features/min-infra.env
  mage -v deploy:kindAll
```

Retrieve a token from Platform Keycloak:

```bash
export JWT_TOKEN=`curl -s --location --request POST 'https://keycloak.kind.internal/realms/master/protocol/openid-connect/token' \
--header 'Content-Type: application/x-www-form-urlencoded' \
--data-urlencode 'grant_type=password' \
--data-urlencode 'client_id=system-client' \
--data-urlencode 'username=lp-admin' \
--data-urlencode 'password=ChangeMeOn1stLogin!' \
--data-urlencode 'scope=openid profile email groups' | jq .access_token`
```

Run integration tests:

```bash
TODO: update URL with MT GW 
go test -count=1 -failfast -v ./test/client/ -apiurl=https://[API_URL]/edge-infra.orchestrator.apis/v1/ -run TestHostInvalidate
```

or using Make target:

```bash
TODO: use new MT GW API.
make int-test-host API_URL="https://api.kind.internal/edge-infra.orchestrator.apis/v1/"
```

## Run test with Edge Infrastructure Manager deployment in Coder environment

1. Deploy Edge Infrastructure Manager in Coder environment. Follow the steps in
   [Orchestrator Deployment in Coder][coder-deployment] guide.

2. Set the following environment variables:

    ```bash
    ADMIN_USER=all-groups-example-user
    ADMIN_PASS=ChangeMeOn1stLogin!
    CLUSTER_FQDN=kind.internal
    ORG_NAME=example-org
    ORG_ADMIN_USER=${ORG_NAME}-admin
    ORG_ADMIN_PASS=ChangeMeOn1stLogin!
    PROJ_NAME=example-proj
    PROJ_ADMIN_USER=${PROJ_NAME}-admin
    PROJ_ADMIN_PASS=ChangeMeOn1stLogin!
    CUSTOMER_ID=1234567
    PRODUCT_KEY=1234567
    ```

    **Note:** Ensure that `$ADMIN_USER` is joined to the `Org-Admin-Group` Keycloak group - this can be done through
    Keycloak web-ui: `https://keycloak.${CLUSTER_FQDN}/`.

3. Retrieve a JWT token from Keycloak:

    ```bash
    JWT_TOKEN=$(curl -s --location --request POST https://keycloak.${CLUSTER_FQDN}/realms/master/protocol/openid-connect/token \
    --header 'Content-Type: application/x-www-form-urlencoded' \
    --data-urlencode 'grant_type=password' \
    --data-urlencode 'client_id=system-client' \
    --data-urlencode "username=${ADMIN_USER}" \
    --data-urlencode "password=${ADMIN_PASS}" \
    --data-urlencode 'scope=openid profile email groups' | jq -r .access_token)
    ```

4. Create a new org via MT APIs

    ```bash
    curl -X PUT "https://api.${CLUSTER_FQDN}/v1/orgs/${ORG_NAME}" -H "accept: application/json" -H "Authorization: Bearer ${JWT_TOKEN}" -H "Content-Type: application/json" -d "{\"description\":\"${ORG_NAME}\"}"

    ORG_UUID=$(curl --location https://api.${CLUSTER_FQDN}/v1/orgs/${ORG_NAME} -H "accept: application/json" -H "Content-Type: application/json" -H "Authorization: Bearer ${JWT_TOKEN}" | jq -r .orgStatus.uID)
    ```

5. Create a license for the new org

    ```bash
    curl -X PUT https://api.${CLUSTER_FQDN}/v1/orgs/${ORG_NAME}/licenses/enlicense -H "accept: application/json" -H "Authorization: Bearer ${JWT_TOKEN}" -H "Content-Type: application/json" -d "{\"customerID\":\"${CUSTOMER_ID}\",\"productKey\":\"${PRODUCT_KEY}\"}"
    ```

6. Create an admin user for the org

    ```bash
    curl -X POST "https://keycloak.${CLUSTER_FQDN}/admin/realms/master/users" \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer ${JWT_TOKEN}" \
        -d '{
              "username": "'${ORG_ADMIN_USER}'",
              "enabled": true,
              "emailVerified": true,
              "credentials": [{
                  "type": "password",
                  "value": "'${ORG_ADMIN_PASS}'",
                  "temporary": false
              }]
            }'

    ORG_ADMIN_USER_ID=$(curl -X GET "https://keycloak.${CLUSTER_FQDN}/admin/realms/master/users?username=${ORG_ADMIN_USER}" -H "Authorization: Bearer ${JWT_TOKEN}" | jq -r '.[0].id')

    ORG_PROJMGR_GROUP_ID=$(curl -X GET "https://keycloak.${CLUSTER_FQDN}/admin/realms/master/groups?search=${ORG_UUID}_Project-Manager-Group" -H "Authorization: Bearer ${JWT_TOKEN}" | jq -r '.[0].id')

    curl -X PUT "https://keycloak.${CLUSTER_FQDN}/admin/realms/master/users/${ORG_ADMIN_USER_ID}/groups/${ORG_PROJMGR_GROUP_ID}" \
        -H "Authorization: Bearer ${JWT_TOKEN}" \
        -H "Content-Type: application/json" \
        -d '{}'

    ORG_JWT_TOKEN=$(curl -s --location --request POST https://keycloak.${CLUSTER_FQDN}/realms/master/protocol/openid-connect/token \
        --header 'Content-Type: application/x-www-form-urlencoded' \
        --data-urlencode 'grant_type=password' \
        --data-urlencode 'client_id=system-client' \
        --data-urlencode "username=${ORG_ADMIN_USER}" \
        --data-urlencode "password=${ORG_ADMIN_PASS}" \
        --data-urlencode 'scope=openid profile email groups' | jq -r .access_token)
    ```

7. Create a project for the org

    ```bash
    curl -X PUT https://api.${CLUSTER_FQDN}/v1/projects/${PROJ_NAME} -H "accept: application/json" -H "Authorization: Bearer ${ORG_JWT_TOKEN}" -H "Content-Type: application/json" -d "{\"description\":\"${PROJ_NAME}\"}"

    PROJ_UUID=$(curl --location https://api.${CLUSTER_FQDN}/v1/projects/${PROJ_NAME} -H "accept: application/json" -H "Content-Type: application" -H "Authorization: Bearer ${ORG_JWT_TOKEN}" | jq -r .projectStatus.uID)
    ```

8. Wait for the project to be provisioned in the orchestrator

    ```bash
    while [ "$(curl -s --location https://api.${CLUSTER_FQDN}/v1/projects/${PROJ_NAME} -H "accept: application/json" -H "Content-Type: application" -H "Authorization: Bearer ${ORG_JWT_TOKEN}" | jq -r .projectStatus.statusIndicator)" != "STATUS_INDICATION_IDLE" ]; do
      echo "Waiting for ${PROJ_NAME} to be provisioned..."
      sleep 5
    done
    ```

9. Create admin user for the project

    ```bash
    curl -X POST "https://keycloak.${CLUSTER_FQDN}/admin/realms/master/users" \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer ${JWT_TOKEN}" \
        -d '{
              "username": "'${PROJ_ADMIN_USER}'",
              "enabled": true,
              "emailVerified": true,
              "credentials": [{
                  "type": "password",
                  "value": "'${PROJ_ADMIN_PASS}'",
                  "temporary": false
              }]
            }'

    PROJ_ADMIN_USER_ID=$(curl -X GET "https://keycloak.${CLUSTER_FQDN}/admin/realms/master/users?username=${PROJ_ADMIN_USER}" -H "Authorization: Bearer ${JWT_TOKEN}" | jq -r .[0].id)

    PROJ_EDGE_MANAGER_GROUP_ID=$(curl -X GET "https://keycloak.${CLUSTER_FQDN}/admin/realms/master/groups?search=${PROJ_UUID}_Edge-Manager-Group" -H "Authorization: Bearer ${JWT_TOKEN}" | jq -r .[0].id)

    PROJ_EDGE_ONB_GROUP_ID=$(curl -X GET "https://keycloak.${CLUSTER_FQDN}/admin/realms/master/groups?search=${PROJ_UUID}_Edge-Onboarding-Group" -H "Authorization: Bearer ${JWT_TOKEN}" | jq -r .[0].id)

    PROJ_EDGE_OP_GROUP_ID=$(curl -X GET "https://keycloak.${CLUSTER_FQDN}/admin/realms/master/groups?search=${PROJ_UUID}_Edge-Operator-Group" -H "Authorization: Bearer ${JWT_TOKEN}" | jq -r .[0].id)

    PROJ_HOST_MGR_GROUP_ID=$(curl -X GET "https://keycloak.${CLUSTER_FQDN}/admin/realms/master/groups?search=${PROJ_UUID}_Host-Manager-Group" -H "Authorization: Bearer ${JWT_TOKEN}" | jq -r .[0].id)

    PROJ_MGR_GROUP_ID=$(curl -X GET "https://keycloak.${CLUSTER_FQDN}/admin/realms/master/groups?search=${ORG_UUID}_Project-Manager-Group" -H "Authorization: Bearer ${JWT_TOKEN}" | jq -r .[0].id)

    curl -X PUT "https://keycloak.${CLUSTER_FQDN}/admin/realms/master/users/${PROJ_ADMIN_USER_ID}/groups/${PROJ_EDGE_MANAGER_GROUP_ID}" \
        -H "Authorization: Bearer ${JWT_TOKEN}" \
        -H "Content-Type: application/json" \
        -d '{}'

    curl -X PUT "https://keycloak.${CLUSTER_FQDN}/admin/realms/master/users/${PROJ_ADMIN_USER_ID}/groups/${PROJ_EDGE_ONB_GROUP_ID}" \
        -H "Authorization: Bearer ${JWT_TOKEN}" \
        -H "Content-Type: application/json" \
        -d '{}'

    curl -X PUT "https://keycloak.${CLUSTER_FQDN}/admin/realms/master/users/${PROJ_ADMIN_USER_ID}/groups/${PROJ_EDGE_OP_GROUP_ID}" \
        -H "Authorization: Bearer ${JWT_TOKEN}" \
        -H "Content-Type: application/json" \
        -d '{}'

    curl -X PUT "https://keycloak.${CLUSTER_FQDN}/admin/realms/master/users/${PROJ_ADMIN_USER_ID}/groups/${PROJ_HOST_MGR_GROUP_ID}" \
        -H "Authorization: Bearer ${JWT_TOKEN}" \
        -H "Content-Type: application/json" \
        -d '{}'

    curl -X PUT "https://keycloak.${CLUSTER_FQDN}/admin/realms/master/users/${PROJ_ADMIN_USER_ID}/groups/${PROJ_MGR_GROUP_ID}" \
        -H "Authorization: Bearer ${JWT_TOKEN}" \
        -H "Content-Type: application/json" \
        -d '{}'
    ```

10. Obtain JWT token of the project admin and set project-id to run the integration tests with MT-awareness:

    ```bash
    ORG_JWT_TOKEN=$(curl -s --location --request POST https://keycloak.${CLUSTER_FQDN}/realms/master/protocol/openid-connect/token \
        --header 'Content-Type: application/x-www-form-urlencoded' \
        --data-urlencode 'grant_type=password' \
        --data-urlencode 'client_id=system-client' \
        --data-urlencode "username=${ORG_ADMIN_USER}" \
        --data-urlencode "password=${ORG_ADMIN_PASS}" \
        --data-urlencode 'scope=openid profile email groups' | jq -r .access_token)
    export PROJECT_ID=$(curl --location https://api.${CLUSTER_FQDN}/v1/projects/${PROJ_NAME} -H "accept: application/json" -H "Content-Type: application" -H "Authorization: Bearer ${ORG_JWT_TOKEN}" | jq -r .projectStatus.uID)
    export JWT_TOKEN=$(curl -s --location --request POST https://keycloak.${CLUSTER_FQDN}/realms/master/protocol/openid-connect/token \
    --header 'Content-Type: application/x-www-form-urlencoded' \
    --data-urlencode 'grant_type=password' \
    --data-urlencode 'client_id=system-client' \
    --data-urlencode "username=${PROJ_ADMIN_USER}" \
    --data-urlencode "password=${PROJ_ADMIN_PASS}" \
    --data-urlencode 'scope=openid profile email groups' | jq -r .access_token)
    ```

11. Run integration tests:

    ```bash
    TODO: update URL with MT GW 
    go test -count=1 -failfast -v ./test/client/ -apiurl=https://[API_URL].kind.internal/edge-infra.orchestrator.apis/v1/ -run TestHostInvalidate
    ```

    or using Make target:

    ```bash
    TODO: update URL with MT GW 
    make int-test-host API_URL="https://[API_URL]/edge-infra.orchestrator.apis/v1/"
    ```

[coder-deployment]: TODO: Add link to guide for coder deployment
