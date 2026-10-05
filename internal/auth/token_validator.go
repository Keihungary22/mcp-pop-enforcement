package auth

// TokenValidator validates an access token and returns its claims.
// TokenValidatorはAccess Tokenを検証しClaimsを返す。
type TokenValidator interface {
	Validate(tokenString string) (*Claims, error)
}
