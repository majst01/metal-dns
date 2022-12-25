package api.v1.metalstack.io.authz

e = {"permission": permissions["/api.v1.DomainService/Get"], "public": false} {
	input.method == "/api.v1.DomainService/Get"
	input.method == token.payload.permissions[_]
	input.request.name == token.payload.domains[_]
}

e = {"permission": permissions["/api.v1.DomainService/List"], "public": false} {
	input.method == "/api.v1.DomainService/List"
	input.method == token.payload.permissions[_]
}

e = {"permission": permissions["/api.v1.DomainService/Create"], "public": false} {
	input.method == "/api.v1.DomainService/Create"
	input.method == token.payload.permissions[_]
	input.request.name == token.payload.domains[_]
}

e = {"permission": permissions["/api.v1.DomainService/Update"], "public": false} {
	input.method == "/api.v1.DomainService/Update"
	input.method == token.payload.permissions[_]
	input.request.name == token.payload.domains[_]
}

e = {"permission": permissions["/api.v1.DomainService/Delete"], "public": false} {
	input.method == "/api.v1.DomainService/Delete"
	input.method == token.payload.permissions[_]
	input.request.name == token.payload.domains[_]
}

domain_name_allowed {
	some i
	domain := token.payload.domains[i]
	name := input.request.name
	endswith(domain, name)
}
