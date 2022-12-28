package api.v1.metalstack.io.authz

jwt := io.jwt.encode_sign(
	{
		"typ": "JWT",
		"alg": "HS256",
	},
	{
		"sub": "1234567890",
		"name": "John Doe",
		"iat": time.now_ns() / 1000000000,
		"nbf": (time.now_ns() / 1000000000) - 100,
		"exp": (time.now_ns() / 1000000000) + 100,
		"domains": [
			"a.example.com",
			"b.example.com",
			"sample.com",
		],
		"permissions": [
			"/api.v1.TokenService/Create",
			"/api.v1.DomainService/Get",
			"/api.v1.DomainService/List",
			"/api.v1.DomainService/Create",
			"/api.v1.DomainService/Update",
			"/api.v1.DomainService/Delete",
			"/api.v1.RecordService/List",
			"/api.v1.RecordService/Create",
			"/api.v1.RecordService/Update",
			"/api.v1.RecordService/Delete",
		],
	},
	{
		"kty": "oct",
		"k": base64.encode("secret"),
	},
)

jwt_with_wrong_secret := io.jwt.encode_sign(
	{
		"typ": "JWT",
		"alg": "HS256",
	},
	{
		"sub": "1234567890",
		"name": "John Doe",
		"iat": time.now_ns() / 1000000000,
		"nbf": (time.now_ns() / 1000000000) - 100,
		"exp": (time.now_ns() / 1000000000) + 100,
		"domains": [
			"a.example.com",
			"b.example.com",
			"sample.com",
		],
	},
	{
		"kty": "oct",
		"k": base64.encode("wrong-secret"),
	},
)

jwt_with_wrong_domains := io.jwt.encode_sign(
	{
		"typ": "JWT",
		"alg": "HS256",
	},
	{
		"sub": "1234567890",
		"name": "John Doe",
		"iat": time.now_ns() / 1000000000,
		"nbf": (time.now_ns() / 1000000000) - 100,
		"domains": [
			"sample.com",
			"foo.bar",
		],
	},
	{
		"kty": "oct",
		"k": base64.encode("secret"),
	},
)
