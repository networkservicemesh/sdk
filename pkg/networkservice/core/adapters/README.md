## Intro

This package provides adapters to translate between `NetworkServiceServer` and `NetworkServiceClient` interfaces.

## Benchmarks

| Benchmark                           | ns/op | allocs/op |
|-------------------------------------|------:|----------:|
| `BenchmarkServerToClient_Request`   |   288 |         6 |
| `BenchmarkClientToServer_Request`   |   291 |         6 |
| `BenchmarkNested20Adapters_Request` |  6159 |       120 |
