# Trace chain element

The purpose of `trace.trace{Client, Server}` is to 
log requests and responses of wrapped `NetworkService{Client, Server}` to span.
It creates a new span for every Request() or Close().

Instead of logging full request, trace element saves it in context, so that the trace element 
that wrapped the next client or server, compares the request in the context with its own, and if they do not match,
calculates the diff and logs only it. The same with response.
