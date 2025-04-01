# Auditing Package for inframanager

This package implements a common auditing package for inframanager applications, based
on gRPC or REST interception.

The gRPC calls are intercepted both incoming and outgoing of the gRPC handler.
The operation, request, response, errors, status and the user are logged.

The REST calls are intercepted through a middleware. The path, operation request, response, errors, status
and the user are logged.

A key/value pair `.Str("event", "auditmessage")` is also appended to each log message for ease of grafana filtering.

## Examples of Auditing logs

### REST middleware Northbound

```json
{
  "level": "info",
  "component": "InfraInvAudit",
  "event": "auditmessage",
  "operation": "POST",
  "path": "test",
  "user": "username",
  "email": "test_email",
  "timestamp": "2024-06-19T15:01:00.395963Z",
  "message": "Northbound API Operation"
}
```

### gRPC interceptor

Operation request:

```json
{
  "level": "info",
  "component": "Audit",
  "event": "auditmessage",
  "operation": "/inventory.v1.InventoryService/CreateResource",
  "user": "username",
  "email": "test_email",
  "request": "client_uuid:\"50354142-dcf6-4d19-a5ff-84dfd4308ca6\"  resource:{host:{name:\"for unit testing purposes\"  desired_state:HOST_STATE_ONBOARDED  note:\"some note\"  hardware_kind:\"XDgen2\"  serial_number:\"12345678\"  uuid:\"8bea94d5-4829-4527-aeaa-c80177a7385e\"  memory_bytes:68719476736  cpu_model:\"12th Gen Intel(R) Core(TM) i9-12900\"  cpu_sockets:1  cpu_cores:14  cpu_architecture:\"x86_64\"  cpu_threads:10  mgmt_ip:\"192.168.10.10\"  bmc_kind:BAREMETAL_CONTROLLER_KIND_PDU  bmc_ip:\"10.0.0.10\"  bmc_username:\"user\"  bmc_password:\"pass\"  pxe_mac:\"90:49:fa:ff:ff:ff\"  hostname:\"testhost1\"  desired_power_state:POWER_STATE_ON}}",
  "timestamp": "2024-06-24T08:33:02.539896Z",
  "message": "Operation request"
}
```

Operation response:

```json
{
  "level": "info",
  "component": "Audit",
  "event": "auditmessage",
  "user": "username",
  "email": "test_email",
  "response": "host:{resource_id:\"host-697e00e0\"  name:\"for unit testing purposes\"  desired_state:HOST_STATE_ONBOARDED  note:\"some note\"  hardware_kind:\"XDgen2\"  serial_number:\"12345678\"  uuid:\"8bea94d5-4829-4527-aeaa-c80177a7385e\"  memory_bytes:68719476736  cpu_model:\"12th Gen Intel(R) Core(TM) i9-12900\"  cpu_sockets:1  cpu_cores:14  cpu_architecture:\"x86_64\"  cpu_threads:10  mgmt_ip:\"192.168.10.10\"  bmc_kind:BAREMETAL_CONTROLLER_KIND_PDU  bmc_ip:\"10.0.0.10\"  bmc_username:\"user\"  bmc_password:\"pass\"  pxe_mac:\"90:49:fa:ff:ff:ff\"  hostname:\"testhost1\"  desired_power_state:POWER_STATE_ON}",
  "timestamp": "2024-06-24T08:33:02.546452Z",
  "message": "Operation result"
}
```
