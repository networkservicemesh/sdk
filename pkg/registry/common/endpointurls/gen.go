package endpointurls

import (
	"sync"
)

//go:generate go-syncmap -output sync_set.gen.go -type Set<net/url.URL,struct{}>

// Set is like a Go map[url.URL]struct{} but is safe for concurrent use
// by multiple goroutines without additional locking or coordination
type Set sync.Map
