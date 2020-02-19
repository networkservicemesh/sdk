## Intro

There are lots of instances in which you want to insure that a series of executions have the properties of:

1.  Exclusivity: One at a time execution
2.  Order: Execution happens in first in, first out (FIFO) order

serial.Executor provides these guarantees.  Given a serialize.Executor ```executor```:

```go
executor.AsyncExec(func(){...})
```

will non-blockingly add ```func(){...}``` to an ordered queue (first in, first out (FIFO)) to be executed.

```executor.AsyncExec(func(){...}``` returns a channel that will be closed when func() has returned.  Therefore:

```go
<-executor.AsyncExec(func(){...})
```

will add ```func(){...}``` to an ordered queue (first in, first out (FIFO)) to be executed and return when 
```func(){...}``` has returned.

### Comparison to sync.Mutex

```sync.Mutex``` guarantees exclusivity, but not order.  For this reason it is often used synchronously to insure ordering.
```serialize.Executor``` guarantees both exclusivity and order, allowing asynchronous use.

## Uses

serial.Executor.Exec can be used in situations in which you need thread safe modification of
an object but don't need to or want to block waiting for it to happen:

```go
type myStruct struct {
  data string
  executor serialize.Executor
}

func (m *myStruct) Update(s string) {
  m.executor.AsyncExec(func(){
    m.data = s
  })
}
```

serialize.Executor.Exec can also be  in situations in which you need thread safe modification of
an object but need Synchronous update.

```go
type myStruct struct {
  data string
  executor serialize.Executor
}

func (m *myStruct) Update(s string) {
  <-m.executor.AsyncExec(func(){
    m.data = s
  })
}
```
