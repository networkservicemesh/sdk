# Tokens valid - checks the validity of some or all of the tokens in the path.

package defaultpolicies

default tokens_valid = false

tokens_valid {
    token := input.connection.path.path_segments[input.connection.path.index].token
    cert := input.auth_info.certificate
    spiffe_id := input.auth_info.spiffe_id # spiffe_id from tlsInfo(SVIDx509Cert)
    io.jwt.verify_es256(token, cert) # signature verification
    [_, payload, _] := io.jwt.decode(token)
    payload.sub == spiffe_id
}
