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
		"domains": [
			"a.example.com",
			"b.example.com",
			"sample.com",
		],
		"permissions": [
			"/v1.TokenService/Create",
			"/v1.DomainService/Get",
			"/v1.DomainService/List",
			"/v1.DomainService/Create",
			"/v1.DomainService/Update",
			"/v1.DomainService/Delete",
			"/v1.RecordService/List",
			"/v1.RecordService/Create",
			"/v1.RecordService/Update",
			"/v1.RecordService/Delete",
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
