# metal-dns

## Authorization

We can use JWT Tokens which include the allowed domains where the owner of this token is allowed to:

- get/list/create domains if not already present
- get/add/edit/delete records

Example JWT Payload:

```json
{
  "sub": "1234567890",
  "name": "John Doe",
  "iat": 1516239022,
  "domains": [
     "a.example.com",
     "b.example.com"
  ]
}
```
