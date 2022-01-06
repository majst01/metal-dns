package api.v1.metalstack.io.authz

is_token_valid {
  token.valid
  now := time.now_ns() / 1000000000
  token.payload.nbf <= now
  # now < token.payload.exp
}

token := {"valid": valid, "payload": payload} {
    [valid, _, payload] := io.jwt.decode_verify(input.token, {"secret": input.secret})
}