## Logging levels

- TRACE, DEBUG: these levels print diff, since diff is expensive operation (see diff between Trace and Info tests) it's recommended to use only for debugging.
- INFO: prints short information about requests and connections.
- FATAL, ERROR, WARN: Print only logs from chain elements that are using these levels directly in the source code e.g., `log.FromContext(ctx).Warn("todo")`

## Benchmarks

**Release v1.14.1**
```
Benchmark_ShortRequest_Info-8   	    3924	    344671 ns/op	   25641 B/op	     178 allocs/op
Benchmark_LongRequest_Info-8   	        340	        3765718 ns/op	  273254 B/op	     861 allocs/op

Benchmark_ShortRequest_Trace-8   	      79	  13600986 ns/op	  344445 B/op	    8475 allocs/op
Benchmark_LongRequest_Trace-8   	       2	 916385562 ns/op	20998324 B/op	  694678 allocs/op

Benchmark_LongRequest_Trace_NoDiff-8   	       2	 565520104 ns/op	12236116 B/op	  585667 allocs/op
Benchmark_LongRequest_Diff_Warn-8   	  340	        3765718 ns/op	  273254 B/op	     861 allocs/op

```


**Release v1.14.2**
```
Benchmark_ShortRequest_Info-8   	    4090	    350181 ns/op	   24246 B/op	     177 allocs/op
Benchmark_LongRequest_Info-8   	          373	   3064039 ns/op	  253359 B/op	     857 allocs/op

Benchmark_ShortRequest_Trace-8   	     543	   2599280 ns/op	  237262 B/op	    1825 allocs/op
Benchmark_LongRequest_Trace-8   	       9	 131145361 ns/op	22433480 B/op	   20749 allocs/op

Benchmark_LongRequest_Trace_NoDiff-8   	      18	  72167456 ns/op	10900859 B/op	   13685 allocs/op

Benchmark_LongRequest_Diff_Warn-8   	   31128	     36019 ns/op	    9600 B/op	     200 allocs/op
```
