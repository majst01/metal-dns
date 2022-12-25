package api.v1.metalstack.io.authz

e = {"permission": permissions["/v1.TokenService/Create"], "public": true} {
	# FIXME add some sort of admin auth
	input.method == "/api.v1.TokenService/Create"
}

# First token must not contain any permission
# input.method == token.payload.permissions[_]
