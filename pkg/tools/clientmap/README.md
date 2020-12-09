# clientmap.Map

It is a `sync.Map` typed for the `string` keys and `networkservice.NetworkServiceClient` values.

# clientmap.RefcountMap

It is a `clientmap.Map` wrapped with refcounting:
```
Store, LoadOrStore (store)  -->  count = 1
Load, LoadOrStore (load)    -->  count += 1
LoadAndDelete, Delete       -->  count -= 1
```
if count becomes 0, value deletes.

## Performance

`clientmap.RefcountMap` is a very thin wrapper, it doesn't hardly affect the `clientmap.Map` performance:
```
BenchmarkMap
BenchmarkMap-4           	20850049	        54.7 ns/op
BenchmarkRefcountMap
BenchmarkRefcountMap-4   	14408266	        76.3 ns/op
```