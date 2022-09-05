package service

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

type JWTManager struct {
	// secret key to sign and verify the access token
	secretKey string
	// valid duration of the token.
	tokenDuration time.Duration
}

func NewJWTManager(secretKey string, tokenDuration time.Duration) *JWTManager {
	return &JWTManager{secretKey, tokenDuration}
}

/*
 The JSON web token should contain a claims object, which has some useful
 information about the user who owns it.
*/
// UserClaims contains the JWT StandardClaims as a composite field.
type UserClaims struct {
	jwt.RegisteredClaims
	Username string `json:"username"`
	Role     string `json:"role"`
}

// Generate generate and sign a new access token for a specific user.
func (manager *JWTManager) Generate(user *User) (string, error) {
	claims := UserClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: &jwt.NumericDate{
				Time: time.Now().Add(manager.tokenDuration),
			},
		},
		Username: user.Username,
		Role:     user.Role,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	return token.SignedString([]byte(manager.secretKey))
}

// Verify parses and verifies access token
func (manager *JWTManager) Verify(accessToken string) (*UserClaims, error) {
	// call jwt.ParseWithClaims(), pass in the access token,
	// an empty user claims, and a custom key function.
	// ParseWithClaims() function will return a token object and an error.
	token, err := jwt.ParseWithClaims(
		accessToken,
		&UserClaims{},
		func(token *jwt.Token) (interface{}, error) {
			// check the signing method of the token to make sure that it
			// matches with the algorithm our server uses, which in our case is HMAC
			_, ok := token.Method.(*jwt.SigningMethodHMAC)
			if !ok {
				return nil, fmt.Errorf("unexpected token signing method")
			}

			// return the secret key that is used to sign the token.
			return []byte(manager.secretKey), nil
		},
	)

	// If the error is not nil, we return invalid token.
	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	// Else we get the claims from the token and convert it to a UserClaims object.
	claims, ok := token.Claims.(*UserClaims)
	// f the conversion fails, we return invalid token claims error.
	if !ok {
		return nil, fmt.Errorf("invalid token claims")
	}

	// Otherwise, just return the user claims.
	return claims, nil
}
