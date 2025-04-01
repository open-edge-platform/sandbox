# PPROF HTTP Server

The inventory perf package enables pprof instrumentation in a component.
The perf package can be imported with the statement below.
It needs to be imported in the main file of the component, as it adds an input flag to the code.
The flag `pprofServerAddress` must be specified in the execution of the compiled code.
When enabled, it starts a web server in the specified address, such as `0.0.0.0:6060`.

```golang
_ "github.com/open-edge-platform/infra-core/inventory/v2/pkg/perf"
```

After the code above is specified in a component, and its docker image is build,
the parameters below enable the pprof instrumentation in the component.

Example to set pprof server address in Inventory:

```yaml
  miinv:
    pprofServerAddress: "0.0.0.0:6060"
```

Example to set pprof server address in API:

```yaml
  serviceArgs:
    pprofServerAddress: "0.0.0.0:6060"
```

There are different ways to read the pprof instrumentation.
For instance, a user can perform a port-forward in the component,
and use the pprof tool to read the pprof outputs.
The commands above provide examples.
Notice the port-forward is associated with the pod (not the service).

```bash
 kubectl -n orch-infra port-forward deploy/inventory 6060
```

```bash
go tool pprof http://localhost:6060/debug/pprof/profile
```
