package api.v1.metalstack.io.authz

jwt := io.jwt.encode_sign({
    "typ": "JWT",
    "alg": "HS256"
}, {
  "sub": "1234567890",
  "name": "John Doe",
  "iat": 1516239022,
  "nbf": 1516239022,
  "domains": [
    "a.example.com",
    "b.example.com",
    "sample.com"
  ]
}, {
    "kty": "oct",
    # base64 encoded "secret"
    "k": "c2VjcmV0"
})



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
