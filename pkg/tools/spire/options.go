package spire

import (
	"context"
)

type entry struct {
	spiffeID string
	selector string
}

type option struct {
	ctx     context.Context
	agentID string
	entries []*entry
}

// Option for spire
type Option func(*option)

// WithContext - use ctx as context for starting spire
func WithContext(ctx context.Context) Option {
	return func(o *option) {
		o.ctx = ctx
	}
}

// WithAgentID - agentID for starting spire
func WithAgentID(agentID string) Option {
	return func(o *option) {
		o.agentID = agentID
	}
}

// WithEntry - Option to add Entry to spire-server.  May be used multiple times.
func WithEntry(spiffeID, selector string) Option {
	return func(o *option) {
		o.entries = append(o.entries, &entry{
			spiffeID: spiffeID,
			selector: selector,
		})
	}
}
