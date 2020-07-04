package dnsresolve

type configurable interface {
	setResolver(r Resolver)
	setDomain(string)
}

type Option interface {
	apply(configurable)
}

type optionApplyFunc func(configurable)

func (f optionApplyFunc) apply(c configurable) {
	f(c)
}

func WithProxyDomain(domain string) Option {
	return optionApplyFunc(func(c configurable) {
		c.setDomain(domain)
	})
}

func WithResolver(r Resolver) Option {
	return optionApplyFunc(func(c configurable) {
		c.setResolver(r)
	})
}
