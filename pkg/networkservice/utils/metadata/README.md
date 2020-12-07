# Functional requirements

1. Chain elements have some per-connection data, that should exist in some storage as long as connection lives and then
should be automatically cleaned up on connection close.
2. Some of this per-connection data should be 'public' and shareable between the chain elements.

# Implementation

## metadata.NewClient/Server()

A client/server chain element that insert a per-connection pair of `sync.Map` (client, server) into the request context.

## metadata.Map(ctx, isClient)

Retrieves per-connection client or server `sync.Map` from the request context.

# Helpers

Raw `metadata.Map(...)` returns untyped `sync.Map` shared between all chain elements. If chain element wants to use it
there are some things it needs to do:
1. create an unexported key type to guarantee that value wouldn't overwrite some other chain element data;
2. cast all data loaded from the `sync.Map` to some value type.

These actions are very common for every `metadata` user, so here are some [genny](https://github.com/cheekybits/genny)
templates with metadata helpers.

## helper/template.go

Provides single-value helper. Template parameters are:
* prefix - prefix for the metadata type and constructor;
* valueType - value type.

```go
type prefixMetaDataHelper struct {
    Store(value valueType)
    LoadOrStore(value valueType) (valueType, bool)
    Load() (valueType, bool)
    LoadAndDelete() (valueType, bool)
    Delete()
}

prefixMetaData(ctx context.Context, isClient bool) *prefixMetaDataHelper
```
