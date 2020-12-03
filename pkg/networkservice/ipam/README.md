# Functional requirements

1. For the different test scenarios we need an endpoint, providing IPAM service in at least 2 modes:
    * point 2 point - allocates pair of IP addresses in /31 subnet;
    * IP subnet - allocates IP address in some given subnet.
2. IPAM service is created on list of some IP subnets. Request can set some exclude IP prefixes for the allocated IP
addresses.
3. IPAM service should be idempotent, so if we have allocated some IP addresses for the request and request type (p2p,
subnet) hasn't changed, and allocated addresses are still not excluded by the excluded prefixes, we should return the
same addresses for the same connection.

# Implementation

## ipamServer

It is a server chain element implementing IPAM service. It works in 2 modes:
```go
conn, _ := ipam.NewServer("10.0.0.0/24").Request(ctx, request)
if request.GetConnection().GetContext().GetIPContext().GetDstIpRequired() {
    // p2p mode
    conn.GetConnection().GetContext().GetIPContext().GetDstIp() // <-- 10.0.0.(2x)/31
    conn.GetConnection().GetContext().GetIPContext().GetSrcIp() // <-- 10.0.0.(2x+1)/31
} else {
    // IP subnet mode
    conn.GetConnection().GetContext().GetIPContext().GetSrcIp() // <-- 10.0.0.x/24
}
```