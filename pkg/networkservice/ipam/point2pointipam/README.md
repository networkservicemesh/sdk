# Functional requirements

1. For the different test scenarios we need an endpoint, providing IPAM service in point 2 point mode - allocates pair 
of IP addresses in /32 subnets and provides static routes for them.
2. IPAM service is created on list of some IP subnets. Request can set some exclude IP prefixes for the allocated IP
addresses.
3. IPAM service should be idempotent, so if we have allocated some IP addresses for the request and request type (p2p,
subnet) hasn't changed, and allocated addresses are still not excluded by the excluded prefixes, we should return the
same addresses for the same connection.

# Implementation

## ipamServer

It is a server chain element implementing point 2 point IPAM service.

```go
conn, _ := ipam.NewServer("10.0.0.0/24").Request(ctx, &networkservice.NetworkServiceRequest{
    Connection: &networkservice.Connection{
        Context: &networkservice.Context{
            IpContext: &networkservice.IpContext{
                ExcludedPrefixes: []string{"10.0.0.1/32"}
            },
        },
    },
})
conn.GetConnection().GetContext().GetIpContext().GetDstIp()                    // <-- 10.0.0.0/32
conn.GetConnection().GetContext().GetIpContext().GetDstRoutes()[0].GetPrefix() // <-- 10.0.0.2/32
conn.GetConnection().GetContext().GetIpContext().GetSrcIp()                    // <-- 10.0.0.2/32
conn.GetConnection().GetContext().GetIpContext().GetSrcRoutes()[0].GetPrefix() // <-- 10.0.0.0/32
```
