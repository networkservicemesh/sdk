# Last Token matches TLSInfo - checks to make sure that the Last Token (JWT) is signed by the same cert as found in TLSInfo.

package defaultpolicies

default last_token_matches_tls = false

last_token_matches_tls {
    token := input.connection.path.path_segments[input.connection.path.index].token
    cert := input.auth_info.certificate
    io.jwt.verify_es256(token, cert) # signature verification
}
