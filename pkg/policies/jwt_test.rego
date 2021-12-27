package api.v1.metalstack.io.authz


test_list_domains_allowed {
    allow with input as {
        "method":"/v1.DomainService/List", 
        "request": null,
        "token" :"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyLCJuYmYiOjE1MTYyMzkwMjIsImRvbWFpbnMiOlsiYS5leGFtcGxlLmNvbSIsImIuZXhhbXBsZS5jb20iXX0.jPEP4TKNpmAcDz_y6AK3wtDr6UOpE69dAylp_qwUNGU",
        }
}
