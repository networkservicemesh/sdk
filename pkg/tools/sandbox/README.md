## Intro

The main goal of package `sandbox` is a simplification setup of NSMgr infrastructure for unit testing.
For example, if we need to check any chain element with remote NSM use-case then we need to configure and start the next GRPC servers:
```
NSMgr1
NSMgr2
Forwarder1
Forwarder2
Registry
```
And after that add needed test steps and asserts. So to reduce test lines was developed `sandbox` package.


## Uses

Let's consider the most common use-cases of package `sandbox`.

### Setup single node NSM 

Problem: setup NSMgr and Forwarder to checking new endpoint chain element.\
Solution:
```go
	...
	localDomain := sandbox.NewBuilder(ctx, t).
		SetNodesCount(1).
		SetNSMgrProxySupplier(nil).
		SetRegistryProxySupplier(nil).
		Build()
	localDomain.Nodes[0].NewEndpoint(ctx, &registry.Endpoint{...}, sandbox.GenerateTestToken, ...myNewEndpointChain)
	client := localDomain.Nodes[0].NewClient(ctx, sandbox.GenerateTestToken, ...clientChain)
	...
```

Problem: setup my external NSMgr and Forwarder to checking my external NSMgr.\
Solution:
```go
	...
	localDomain := sandbox.NewBuilder(ctx, t).
		SetNodesCount(1).
		SetNSMgrProxySupplier(nil).
		SetRegistryProxySupplier(nil).
		SetNSMgrSupplier(myExternalNSMgrFunc)
		Build()
	defer localDomain.Cleanup()
	...
```

Problem: setup my NSMgr and new Forwarder to checking my Forwarder chain.\
Solution:
```go
	...
	localDomain := sandbox.NewBuilder(ctx, t).
		SetNodesCount(1).
		SetNSMgrProxySupplier(nil).
		SetRegistryProxySupplier(nil).
		SetNodeSetup(func(ctx context.Context, node *sandbox.Node, _ int) {
			node.NewNSMgr(ctx, sandbox.Name("nsmgr"), nil, sandbox.GenerateTestToken, nsmgr.NewServer)
			node.NewForwarder(ctx, &registry.Endpoint{...}, sandbox.GenerateTestToken, ...myNewForwarderChain)
		}).
		Build()
	defer localDomain.Cleanup()
	...
```

### Setup remote NSM infrastructure for remote use-case 

Problem: setup NSMgr and Forwarder to checking remote use-case.\
Solution:
```go
	...
	localDomain := sandbox.NewBuilder(ctx, t).
		SetNodesCount(2).
		SetNSMgrProxySupplier(nil).
		SetRegistryProxySupplier(nil).
		Build()
	defer localDomain.Cleanup()
	localDomain.Nodes[1].NewEndpoint(ctx, &registry.Endpoint{...}, sandbox.GenerateTestToken, ...remoteEndpointChain)
	...
```

Problem: setup NSMgr, set of Forwarders and set of Endpoints on each node to checking complex scenarios.\
Solution:
```go
	...
	localDomain := sandbox.NewBuilder(ctx, t).
		SetNodesCount(nodesCount).
		SetNSMgrProxySupplier(nil).
		SetRegistryProxySupplier(nil).
		SetNodeSetup(func(ctx context.Context, node *sandbox.Node) {
			node.NewForwarder(ctx, &registry.Endpoint{...}, sandbox.GenerateTestToken, ...forwarderChain)
			node.NewEndpoint(ctx, &registry.Endpoint{...}, sandbox.GenerateTestToken, ...endpointChain)
		}).
		Build()
	defer localDomain.Cleanup()
	...
```

### Setup only local registry

Problem: setup registry and to check API.\
Solution: 
```go
	...
	localDomain := sandbox.NewBuilder(ctx, t).
		SetNodesCount(0).
		SetNSMgrProxySupplier(nil).
		SetRegistryProxySupplier(nil).
		Build()
	defer localDomain.Cleanup()
	registryURL := localDomain.Registry.URL
	// Create new registry client by registryURL
	...
```

### Setup remote NSM infrastructure for interdomain use-case 

Problem: setup NSMgrs, Forwarders, Registries to checking interdomain use-case via DNS.\
Solution:
```go
	...
	fakeServer := sandbox.NewFakeDNSResolver()
	domain1 := sandbox.NewBuilder(ctx, t).
		SetNodesCount(1).
		SetDNSDomainName("domain1").
		SetDNSResolver(fakeServer).
		Build()
	defer domain1.Cleanup()
	fakeServer.Register("domain1", domain1.Registry.URL)
	domain2 := sandbox.NewBuilder(ctx, t).
		SetNodesCount(1).
		SetDNSDomainName("domain2").
		SetDNSResolver(fakeServer).
		Build()
	defer domain1.Cleanup()
	fakeServer.Register("domain2", domain2.Registry.URL)
	...
```
