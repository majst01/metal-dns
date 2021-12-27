package api.v1.metalstack.io.authz

default allow = false

allow {
  # FIXME add some sort of admin auth
  input.method == "/v1.TokenService/Create"
}

allow {
    is_token_valid
    action_allowed
}

action_allowed {
  input.method == "/v1.DomainService/List"
}

action_allowed {
  input.method == "/v1.DomainService/Create"
  input.request.name == token.payload.domains[_]
}

action_allowed {
  input.method == "/v1.RecordService/List"
  input.request.domain == token.payload.domains[_]
}

action_allowed {
  input.method == "/v1.RecordService/Create"
  domain_name_allowed
}

domain_name_allowed {
    some i
    domain := token.payload.domains[i]
    endswith(input.request.name, domain)
}

is_token_valid {
  token.valid
  now := time.now_ns() / 1000000000
  token.payload.nbf <= now
  # now < token.payload.exp
}

token := {"valid": valid, "payload": payload} {
    [valid, _, payload] := io.jwt.decode_verify(input.token, {"secret": input.secret})
}