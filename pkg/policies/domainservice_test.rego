package api.v1.metalstack.io.authz

secret = "secret"

test_get_domain_allowed {
	decision.allow with input as {
		"method": "/api.v1.DomainService/Get",
		"request": {"name": "a.example.com"},
		"token": jwt,
	}
		with data.secret as secret
}

test_list_domains_not_allowed_with_wrong_jwt {
	not decision.allow with input as {
		"method": "/api.v1.DomainService/List",
		"request": null,
		"token": jwt_with_wrong_secret,
	}
		with data.secret as secret
}

test_list_domains_allowed_with_empty_domain {
	decision.allow with input as {
		"method": "/api.v1.DomainService/List",
		"request": null,
		"token": jwt,
	}
		with data.secret as secret
}

test_list_domains_allowed {
	decision.allow with input as {
		"method": "/api.v1.DomainService/List",
		"request": {"name": "example.com"},
		"token": jwt,
	}
		with data.secret as secret
}

test_create_domains_allowed {
	decision.allow with input as {
		"method": "/api.v1.DomainService/Create",
		"request": {"name": "a.example.com"},
		"token": jwt,
	}
		with data.secret as secret
}

test_create_domains_not_allowed {
	not decision.allow with input as {
		"method": "/api.v1.DomainService/Create",
		"request": {"name": "example.com"},
		"token": jwt,
	}
		with data.secret as secret
}
