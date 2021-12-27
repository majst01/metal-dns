package api.v1.metalstack.io.authz

default allow = false

allow {
    # is_token_valid
    action_allowed
}

is_token_valid {
  token.valid
  now := time.now_ns() / 1000000000
  token.payload.nbf <= now
  # now < token.payload.exp
}

action_allowed {
  input.method == "/v1.DomainService/List"
  # input.request == null
}


token := {"valid": valid, "payload": payload} {
    [valid, _, payload] := io.jwt.decode_verify(input.token, {"secret": "secret"})
}