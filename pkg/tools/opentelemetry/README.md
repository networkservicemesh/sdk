# OpenTelemetry Collector Example

This file contains a description of how to launch OpenTelemetry container with Jaeger and Zipkin for spans and Prometheus for metrics using Docker.


## Collector
First, you need to clone repository and switch to the folder with docker-compose file:
```shell
git clone git@github.com:open-telemetry/opentelemetry-collector-contrib.git
cd opentelemetry-collector-contrib/examples/demo
```
Before running docker-compose you need to expose port `4317` for `otel-collector` container inside `docker-compose.yaml`

Run docker-compose file:
```shell
docker-compose up -d
```

The example exposes the following backends:

- Jaeger at http://0.0.0.0:16686
- Zipkin at http://0.0.0.0:9411
- Prometheus at http://0.0.0.0:9090

## Using spans and metrics in tests
After running docker-compose you can enable spans and metrics inside any test using the following code:
```Go
log.EnableTracing(true)
os.Setenv("TELEMETRY", "opentelemetry")
spanExporter := opentelemetry.InitSpanExporter(ctx, "0.0.0.0:4317")
metricExporter := opentelemetry.InitMetricExporter(ctx, "0.0.0.0:4317")
o := opentelemetry.Init(ctx, spanExporter, metricExporter, "NSM")
defer o.Close()
```

Metrics are disabled in tests by default. You can create simple metrics chain element to test them:
```Go
type metricsServer struct {
}
func NewServer() networkservice.NetworkServiceServer {
	return &metricsServer{}
}
func (c *metricsServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	request.Connection.GetCurrentPathSegment().Metrics = make(map[string]string)
	request.Connection.GetCurrentPathSegment().Metrics["my_nsm_metric"] = "10000"
	return next.Server(ctx).Request(ctx, request)
}
func (c *metricsServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	return next.Server(ctx).Close(ctx, conn)
}
```

## Clean up
To clean up docker containers run `docker-compose down` in `opentelemetry-collector-contrib/examples/demo` folder