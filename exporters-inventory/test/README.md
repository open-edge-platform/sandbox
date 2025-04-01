# Integration Tests

This file describes how to execute the integration tests placed in this folder.

## Requirements

It requires helm and kubectl installed, besides an operational Kubernetes cluster.

## Run

Then enable a port-forward to have interface with the API component via port 8080.

```bash
  kubectl -n orch-infra port-forward svc/api 8080
```

Then enable a port-forward to have interface with the Exporter component via port 9101.

```bash
  kubectl -n orch-infra orch-port-forward svc/exporter 9101
```

Now, you are almost good to go. In another terminal run the following command to obtain a valid JWT:

```bash
export FQDN=<TBD>
export USERNAME=<TBD>
export PASSWORD=<TBD>

JWT_TOKEN=$(curl -k --location --request POST "https://keycloak.${FQDN}/realms/master/protocol/openid-connect/token" --header 'Content-Type: application/x-www-form-urlencoded' --data-urlencode 'grant_type=password' --data-urlencode 'client_id=system-client' --data-urlencode "username=${USERNAME}" --data-urlencode "password=${PASSWORD}" --data-urlencode 'scope=openid profile email groups' | jq -r '.access_token')
```

In the same terminal run the tests:

```bash
  go test -v ./test/export
```
