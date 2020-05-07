# No Tokens in path expired - none of the tokens in the chain are expired.

package defaultpolicies

default no_any_expired_tokens = false

no_any_expired_tokens {
    not any_expired_tokens
}

any_expired_tokens {
    token := input.connection.path.path_segments[_].token
    [_, payload, _] := io.jwt.decode(token) #get jwt claims
    now > payload.exp
}

now = s {
	 ns := time.now_ns()
     s := ns / 1e9
}