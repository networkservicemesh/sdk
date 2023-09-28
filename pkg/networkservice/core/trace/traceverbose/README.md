# Trace chain element

The purpose of `trace.trace{Client, Server}` is to 
log requests and responses of wrapped `NetworkService{Client, Server}` to span.
It creates a new span for every Request() or Close().

Instead of logging full request, trace element saves it in context, so that the trace element 
that wrapped the next client or server, compares the request in the context with its own, and if they do not match,
calculates the diff and logs only it. The same with response.

## Benchmarks
If we assume that
* logRequest() - logs full `proto.Message`
* logRequestIfDiffers() - Compares previous and current `proto.Message`. Logs the current message only if they differ.
* logRequestDiff() - Compares previous and current `proto.Message`. If they differ calculates the diff of them and logs only it.

then we get the following results: 

| Type                 | Ops   | ns/op |
| -------------        |:-----:| -----:|
| logRequest()         | 32778 | 36565 |
| logRequestIfDiffers()| 33396 | 41140 |
| logRequestDiff()     | 34741 | 42793 |
