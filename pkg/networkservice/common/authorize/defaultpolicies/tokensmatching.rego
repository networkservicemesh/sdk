# Token.Aud for token n in path matches Token.Sub for n+1 token in chain

package defaultpolicies

default not_match_tokens_exist = false
default valid_sub_aud_in_path = false

valid_sub_aud_in_path {
	not not_match_tokens_exist
}

not_match_tokens_exist {
    token := input.connection.path.path_segments[i].token
    next_token := input.connection.path.path_segments[i+1].token
    [_, payload, _] := io.jwt.decode(token)
    [_, next_payload, _] := io.jwt.decode(next_token)
    payload.aud != next_payload.sub
}