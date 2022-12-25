package api.v1.metalstack.io.authz

test_list_records_not_allowed {
	not decision.allow with input as {
		"method": "/api.v1.RecordService/List",
		"request": {"domain": "example.com"},
		"token": jwt,
		"secret": "secret",
	}
}

test_list_records_allowed {
	decision.allow with input as {
		"method": "/api.v1.RecordService/List",
		"request": {"domain": "a.example.com"},
		"token": jwt,
		"secret": "secret",
	}
}

test_list_records_not_allowed_with_wrong_domains_in_token {
	not decision.allow with input as {
		"method": "/api.v1.RecordService/List",
		"request": {"domain": "example.com"},
		"token": jwt_with_wrong_domains,
		"secret": "secret",
	}
}

test_create_records_allowed {
	decision.allow with input as {
		"method": "/api.v1.RecordService/Create",
		"request": {"name": "www.a.example.com"},
		"token": jwt,
		"secret": "secret",
	}
}
