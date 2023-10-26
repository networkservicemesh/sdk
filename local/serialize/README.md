## Ordered, Exclusive, Easy

Ensure that a series of executions have the properties of:

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

```sync.Mutex``` guarantees exclusivity, but not order.  For this reason it is often used synchronously to ensure ordering.
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


## Warning about Deadlocks

serialize.Executor is pretty resistant to most classes of deadlock mistakes.  You can nest calls to AsyncExec as far down
as you'd like:

```go
func nested(executor *serialize.Executor) {
    executor.AsyncExec(func(){
        executor.AsyncExec(func(){
            executor.AsyncExec(func(){
            }
        }
    })
}
```

will not deadlock no matter how far down you nest.

However, care must be taken to not nest calls to AsyncExec if you block on the returned done channel.

```go
func deadlockNested(executor *serialize.Executor) {
    executor.AsyncExec(func(){
        <-executor.AsyncExec(func(){
        
        }
    }
}
```

will deadlock 100% of the time.  Do not block on done channels in nested calls to AsyncExec
