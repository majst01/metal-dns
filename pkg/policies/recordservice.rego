package api.v1.metalstack.io.authz

e = {"permission": permissions["/v1.RecordService/List"], "public": false} {
	input.method == "/v1.RecordService/List"
	input.method == token.payload.permissions[_]
	endswith(input.request.domain, token.payload.domains[_])
}

e = {"permission": permissions["/v1.RecordService/Create"], "public": false} {
	input.method == "/v1.RecordService/Create"
	input.method == token.payload.permissions[_]
	endswith(input.request.name, token.payload.domains[_])
}

e = {"permission": permissions["/v1.RecordService/Update"], "public": false} {
	input.method == "/v1.RecordService/Update"
	input.method == token.payload.permissions[_]
	endswith(input.request.name, token.payload.domains[_])
}

e = {"permission": permissions["/v1.RecordService/Delete"], "public": false} {
	input.method == "/v1.RecordService/Delete"
	input.method == token.payload.permissions[_]
	input.request.name == token.payload.domains[_]
}

domain_name_allowed {
	some i
	domain := token.payload.domains[i]
	name := input.request.name
	endswith(domain, name)
}
