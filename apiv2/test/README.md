# Integration Tests

This file describes how to execute the integration tests placed in this folder.

> These tests do not delete created Hosts from the Inventory, which means that some tests may fail,
> if you run them two times in a row.
> To avoid that, please either re-deploy API and Inventory,
> or use `hostcli` to delete duplicated/unnecessary hosts

## Requirements

It requires curl, jq, golang and make installed.
And it requires an infra orchestrator deployed and operational, with a project and user already configured.

## Run

Retrieve a token from Keycloak, make sure to set the correct username and password.

```bash
API_TOKEN=$(curl -k --location --request POST 'https://keycloak.kind.internal/realms/master/protocol/openid-connect/token' --header 'Content-Type: application/x-www-form-urlencoded' --data-urlencode 'grant_type=password' --data-urlencode 'client_id=system-client' --data-urlencode 'username=sample-project-api-user' --data-urlencode 'password=ChangeMeOn1stLogin!' --data-urlencode 'scope=openid profile email groups' | jq -r '.access_token')
```

Get the Project ID associated with the user credentials used to get the token above, and run integration tests as below:

```bash
PROJECT_ID=8f13df5b-5853-4ce3-8b83-db23108daa54  JWT_TOKEN=$API_TOKEN go test -timeout=30m -count=1 -failfast -v ./test/client/ -apiurl=https://iaasv2.kind.internal -caPath=ca.crt -run TestHost
```

Or using a Make target:

```bash
make int-test-host API_URL="https://iaasv2.kind.internal" PROJECT_ID=8f13df5b-5853-4ce3-8b83-db23108daa54  JWT_TOKEN=$API_TOKEN
```
