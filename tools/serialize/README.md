## Intro

There are lots of instances in which you want to insure that a series of executions happen:

1.  One at a time
2.  In order

That is what serial.Executor does.  Given a serial executor ```serialExcecutor```:

```go
serialExecutor.AsyncExec(func(){...})
```

will non-blockingly add ```func(){...}``` to an ordered queue (first come, first serve) to be executed.

```go
serialExecutor.SyncExec(func(){...})
```

will add ```func(){...}``` to an ordered queue (first come, first serve) to be executed and return when its done.
## Uses

serial.Executor.AsyncExec can be used instead of a mutex in situations in which you need thread safe modification of
an object but don't need to or want to block waiting for it to happen:

```go
type myStruct struct {
  data string
  executor serialExecutor
}

func (m *myStruct) Update(s string) {
  m.executor.AsynExec(func(){
    m.data = s
  })
}
```

serial.Executor.SyncExec can be used instead of a mutex in situations in which you need thread safe modification of
an object but need Synchronous update.

```go
type myStruct struct {
  data string
  executor serialExecutor
}

func (m *myStruct) Update(s string) {
  m.executor.AsynExec(func(){
    m.data = s
  })
}
```
