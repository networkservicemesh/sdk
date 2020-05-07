# No Tokens in path expired - none of the tokens in the chain are expired.

package defaultpolicies

default no_tokens_expired = false

no_tokens_expired {
    not tokens_expired
}

tokens_expired {
    token := input.connection.path.path_segments[_].token
    [_, payload, _] := io.jwt.decode(token) #get jwt claims
    now > payload.exp
}

now = s {
	 ns := time.now_ns()
     s := ns / 1e9
}