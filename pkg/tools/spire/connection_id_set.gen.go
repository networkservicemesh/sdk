// Code generated by "-output connection_id_set.gen.go -type ConnectionIDSet<string,bool> -output connection_id_set.gen.go -type ConnectionIDSet<string,bool>"; DO NOT EDIT.
// Install -output connection_id_set.gen.go -type ConnectionIDSet<string,bool> by "go get -u github.com/searKing/golang/tools/-output connection_id_set.gen.go -type ConnectionIDSet<string,bool>"

package spire

import (
	"sync" // Used by sync.Map.
)

// Generate code that will fail if the constants change value.
func _() {
	// An "cannot convert ConnectionIDSet literal (type ConnectionIDSet) to type sync.Map" compiler error signifies that the base type have changed.
	// Re-run the go-syncmap command to generate them again.
	_ = (sync.Map)(ConnectionIDSet{})
}

var _nil_ConnectionIDSet_bool_value = func() (val bool) { return }()

// Load returns the value stored in the map for a key, or nil if no
// value is present.
// The ok result indicates whether value was found in the map.
func (m *ConnectionIDSet) Load(key string) (bool, bool) {
	value, ok := (*sync.Map)(m).Load(key)
	if value == nil {
		return _nil_ConnectionIDSet_bool_value, ok
	}
	return value.(bool), ok
}

// Store sets the value for a key.
func (m *ConnectionIDSet) Store(key string, value bool) {
	(*sync.Map)(m).Store(key, value)
}

// LoadOrStore returns the existing value for the key if present.
// Otherwise, it stores and returns the given value.
// The loaded result is true if the value was loaded, false if stored.
func (m *ConnectionIDSet) LoadOrStore(key string, value bool) (bool, bool) {
	actual, loaded := (*sync.Map)(m).LoadOrStore(key, value)
	if actual == nil {
		return _nil_ConnectionIDSet_bool_value, loaded
	}
	return actual.(bool), loaded
}

// LoadAndDelete deletes the value for a key, returning the previous value if any.
// The loaded result reports whether the key was present.
func (m *ConnectionIDSet) LoadAndDelete(key string) (value bool, loaded bool) {
	actual, loaded := (*sync.Map)(m).LoadAndDelete(key)
	if actual == nil {
		return _nil_ConnectionIDSet_bool_value, loaded
	}
	return actual.(bool), loaded
}

// Delete deletes the value for a key.
func (m *ConnectionIDSet) Delete(key string) {
	(*sync.Map)(m).Delete(key)
}

// Range calls f sequentially for each key and value present in the map.
// If f returns false, range stops the iteration.
//
// Range does not necessarily correspond to any consistent snapshot of the Map's
// contents: no key will be visited more than once, but if the value for any key
// is stored or deleted concurrently, Range may reflect any mapping for that key
// from any point during the Range call.
//
// Range may be O(N) with the number of elements in the map even if f returns
// false after a constant number of calls.
func (m *ConnectionIDSet) Range(f func(key string, value bool) bool) {
	(*sync.Map)(m).Range(func(key, value interface{}) bool {
		return f(key.(string), value.(bool))
	})
}