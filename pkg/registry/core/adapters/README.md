## Intro

This package provides API to convert client to server and vise versa.

## Benchmarks

| Benchmark                     | ns/op | allocs/op |
|-------------------------------|------:|----------:|
| `BenchmarkNSServer_Register`  |   223 |         5 |
| `BenchmarkNSServer_Find`      |   779 |         7 |
| `BenchmarkNSClient_Find`      |  1225 |        11 |
| `BenchmarkNSEServer_Register` |   222 |         5 |
| `BenchmarkNSEServer_Find`     |   786 |         7 |
| `BenchmarkNSEClient_Find`     |  1231 |        11 |
