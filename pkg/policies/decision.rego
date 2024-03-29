package api.v1.metalstack.io.authz

default decision = {"allow": false, "isAdmin": false}

admin(e) {
	is_token_valid
	ae := sprintf("%s.admin", [e])
	permissions[ae]
}

user(e) {
	is_token_valid
	permissions[e]
}

decision = {"allow": true, "isAdmin": false} {
	user(e.permission)
	not admin(e.permission)
}

decision = {"allow": true, "isAdmin": true} {
	admin(e.permission)
}

decision = {"allow": false, "isAdmin": false, "reason": reason} {
	not user(e.permission)
	not admin(e.permission)
	not e.public
	reason := sprintf("missing permission on %s", [e.permission])
}

decision = {"allow": true, "isAdmin": false} {
	e.public
}
