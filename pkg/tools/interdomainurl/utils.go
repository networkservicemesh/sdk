package interdomainurl

import "strings"

func Domain(s string) string {
	return strings.SplitN(s, "@", 2)[1]
}

func Name(s string) string {
	return strings.SplitN(s, "@", 2)[0]
}

func HasIdentifier(s string) bool {
	return strings.Contains(s, "@")
}

func Join(s ...string) string {
	return strings.Join(s, "@")
}
