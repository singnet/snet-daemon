package token

// PayLoad is the value bound to a token, normally a prepaid channel ID. The JWT
// implementation compares its fmt string representation rather than encoding a
// structured value.
type PayLoad any

// CustomToken holds an implementation-specific token; the JWT manager returns a string.
type CustomToken any

// Manager creates and verifies prepaid authorization tokens. It does not validate
// an MPE claim signature, channel balance, nonce, or prepaid spending limit.
type Manager interface {
	// CreateToken binds a payload to the supplied user address and configured scope.
	CreateToken(key PayLoad, signer string) (token CustomToken, err error)
	// VerifyToken authenticates the token and payload and returns its user address.
	VerifyToken(token CustomToken, key PayLoad) (userAddress string, err error)
}
