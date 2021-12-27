package api.v1.metalstack.io.authz

jwt := io.jwt.encode_sign({
    "typ": "JWT",
    "alg": "HS256"
}, {
  "sub": "1234567890",
  "name": "John Doe",
  "iat": (time.now_ns() / 1000000000),
  "nbf": (time.now_ns() / 1000000000) - 100,
  "domains": [
    "a.example.com",
    "b.example.com",
    "sample.com"
  ]
}, {
    "kty": "oct",
    "k": base64.encode("secret")
})

jwt_with_wrong_secret := io.jwt.encode_sign({
    "typ": "JWT",
    "alg": "HS256"
}, {
  "sub": "1234567890",
  "name": "John Doe",
  "iat": (time.now_ns() / 1000000000),
  "nbf": (time.now_ns() / 1000000000) - 100,
  "domains": [
    "a.example.com",
    "b.example.com",
    "sample.com"
  ]
}, {
    "kty": "oct",
    "k": base64.encode("wrong-secret")
})

jwt_with_wrong_domains := io.jwt.encode_sign({
    "typ": "JWT",
    "alg": "HS256"
}, {
  "sub": "1234567890",
  "name": "John Doe",
  "iat": (time.now_ns() / 1000000000),
  "nbf": (time.now_ns() / 1000000000) - 100,
  "domains": [
    "sample.com",
    "foo.bar"
  ]
}, {
    "kty": "oct",
    "k": base64.encode("secret")
})

test_list_domains_not_allowed_with_wrong_jwt {
    not allow with input as {
        "method":"/v1.DomainService/List",
        "request": null,
        "token" : jwt_with_wrong_secret,
        "secret": "secret",
        }
}

test_list_domains_allowed {
    allow with input as {
        "method":"/v1.DomainService/List",
        "request": null,
        "token" : jwt,
        "secret": "secret",
        }
}

test_create_domains_allowed {
    allow with input as {
        "method":"/v1.DomainService/Create",
        "request": { "name":"a.example.com"},
        "token" : jwt,
        "secret": "secret",
        }
}
test_create_domains_not_allowed {
    not allow with input as {
        "method":"/v1.DomainService/Create",
        "request": { "name":"example.com"},
        "token" : jwt,
        "secret": "secret",
        }
}

test_list_records_not_allowed {
    not allow with input as {
        "method":"/v1.RecordService/List",
        "request": { "domain":"example.com"},
        "token" : jwt,
        "secret": "secret",
        }
}

test_list_records_not_allowed_with_wrong_domains_in_token {
    not allow with input as {
        "method":"/v1.RecordService/List",
        "request": { "domain":"example.com"},
        "token" : jwt_with_wrong_domains,
        "secret": "secret",
        }
}

test_list_records_allowed {
    allow with input as {
        "method":"/v1.RecordService/List",
        "request": { "domain":"a.example.com"},
        "token" : jwt,
        "secret": "secret",
        }
}

test_create_records_allowed {
    allow with input as {
        "method":"/v1.RecordService/Create",
        "request": { "name":"www.a.example.com"},
        "token" : jwt,
        "secret": "secret",
        }
}
