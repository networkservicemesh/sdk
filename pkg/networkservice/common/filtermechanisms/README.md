## Intro

Package `filtermechanisms` filters out remote mechanisms if communicating by remote url 
filters out local mechanisms otherwise. Can be used with eNSMGR.

## Options

Supported option `WithExternalThreshold` that sets max path segments count for eNSMgrs.

## Examples
Note: **Is local** true when path segment index less than localThreshold and nsmgr knows NSE url

 1. Local:
 ```
     Scheme: nsc-->NSMgr-->cross-nse-->NSMgr-->nse
        NSMgr knows nse: true 
        filterMechanismsServer.localThreshold: 5 
        len(request.Connection.Path.PathSegments):  4
        Is local: true
```

 2. Local eNSM:
 ```
     Scheme: nsc-->eNSMgr-->nse
         eNSMgr knows nse: true
         filterMechanismsServer.localThreshold:  3
         len(request.Connection.Path.PathSegments): 2
         Is local: true
```

 3. Remote eNSM:
 ```
     Scheme: nsc-->eNSMgr1-->eNSMgr2-->nse
        eNSMgr1:
            knows nse: false
            filterMechanismsServer.localThreshold = 3
            len(request.Connection.Path.PathSegments) = 2
            Is local: false

        eNSMgr2:
            knows nse: true
            filterMechanismsServer.localThreshold = 3
            len(request.Connection.Path.PathSegments) = 3
            Is local: false
```

 4. Remote NSMgr + eNSMGR
 ```
     Scheme: nsc-->NSMgr1-->cross-nse1-->NSMgr1-->eNSMgr1-->nse
        NSMgr1:
            knows nse: false
            filterMechanismsServer.localThreshold = 5
            len(request.Connection.Path.PathSegments) = 4
            Is local: false

        eNSMgr1:
            knows nse: true
            filterMechanismsServer.localThreshold = 3
            len(request.Connection.Path.PathSegments) = 5
            Is local: false
```

 5. Remote eNSMGR + NSMgr
 ```
     Scheme: nsc-->eNSMgr1-->NSMgr1-->cross-nse1-->NSMgr1-->eNSMgr2-->nse
        eNSMgr1:
            knows nse: false
            filterMechanismsServer.localThreshold = 3
            len(request.Connection.Path.PathSegments) = 2
            Is local: false

        NSMgr1:
            knows nse: true
            filterMechanismsServer.localThreshold = 5
            len(request.Connection.Path.PathSegments) = 5
            Is local: false
```
 
6. Remote

```
     Scheme: nsc-->NSMgr1-->cross-nse1-->NSMgr1-->NSMgr2-->cross-nse2-->NSMgr2-->nse
        NSMgr2:
            knows nse: false
            filterMechanismsServer.localThreshold = 5
            len(request.Connection.Path.PathSegments) = 4
            Is local: false

        NSMgr1:
            knows nse: true
            filterMechanismsServer.localThreshold = 5
            len(request.Connection.Path.PathSegments) = 7
            Is local: false
```