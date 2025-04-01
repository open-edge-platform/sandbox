# Tracing

Tracing allows developers to quickly follow requests across services and find
relevant information faster.

## Enabling trace exports from applications

### gRPC server setup

```go
import "github.com/open-edge-platform/infra-core/inventory/v2/pkg/tracing"

// CLI flags.
enableTracing = flag.Bool(tracing.EnableTracing, false, tracing.EnableTracingDescription)
traceURL      = flag.String(tracing.TraceURL, "", tracing.TraceURLDescription)

// Setup and run HTTP trace exporter (pusher).
if *enableTracing {
    cleanup, err := tracing.NewTraceExporterHTTP(*traceURL, "some-service", nil)
    // ...
}

// Setup automatic trace exports for gRPC server.
if *enableTracing {
    srvOpts = append(srvOpts, grpc.StatsHandler(otelgrpc.NewServerHandler()))
}
    grpc.NewServer(srvOpts...)
```

### RPC handler code

```go
func (srv *FoogRPCServer) Foo(
	ctx context.Context,
	in *pb.FooRequest,
) (*pb.FooResponse, error) {
    zlog := zlog.TraceCtx(ctx) // Starts or continues a trace from incoming context.
    zlog.Info().Msg("Start of Foo gRPC call") // Any log message will also be sent in the trace.
}
```

### REST server

```go
if *enableTracing {
    tracing.EnableEchoAutoTracing(e, apiTraceName)
}
```

### Manual

```go
func (fr *FooReconciler) Foo(ctx context.Context,
	request rec_v2.Request[ResourceID],
) rec_v2.Directive[ResourceID] {
	if fr.enableTracing {
		ctx = tracing.StartTrace(ctx, "BarService", "FooReconciler")
		defer tracing.StopTrace(ctx)
	}
}
```

## Local setup

Tracing can also be done within local setups.

First, we need a place to push metrics to. [Jaeger](https://www.jaegertracing.io/)
serves as collector, storage and [UI](http://localhost:16686).
Start its docker container:

```bash
make jaeger-start
```

The UI is accessible at [http://localhost:16686](http://localhost:16686).

All Infra services (should) support trace exports. Make sure this feature is enabled in the
flags: `-enableTracing -traceURL="127.0.0.1:4318"`
