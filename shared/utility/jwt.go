package utility

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type JWTTokenizer interface {
	// CreateToken create a token with a content
	CreateToken(content []byte, now time.Time, expired time.Duration) (string, error)

	// VerifyToken verify and return the content
	VerifyToken(tokenString string) ([]byte, error)
}

const fieldContent = "content"

type jwtToken struct {
	secretKey string
}

func NewJWTTokenizer(secretKey string) (JWTTokenizer, error) {

	if strings.TrimSpace(secretKey) == "" {
		return nil, fmt.Errorf("SecretKey must not empty")
	}

	return &jwtToken{
		secretKey: secretKey,
	}, nil
}

func (j jwtToken) CreateToken(content []byte, now time.Time, expired time.Duration) (string, error) {

	contentBase64 := base64.StdEncoding.EncodeToString(content)

	// Create a new token object, specifying signing method and the claims
	// you would like it to contain.
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"exp":        now.Add(expired).Unix(),
		fieldContent: contentBase64,
	})

	// Sign and get the complete encoded token as a string using the secret
	tokenString, err := token.SignedString([]byte(j.secretKey))
	if err != nil {
		return "", err
	}

	return tokenString, nil

}

func (j jwtToken) VerifyToken(tokenString string) ([]byte, error) {

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Don't forget to validate the alg is what you expect:
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		// hmacSampleSecret is a []byte containing your secret, e.g. []byte("my_secret_key")
		return []byte(j.secretKey), nil
	})

	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, fmt.Errorf("token is not valid")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("claims is can not asserted")
	}

	decodeStringInBytes, err := base64.StdEncoding.DecodeString(claims[fieldContent].(string))
	if err != nil {
		return nil, err
	}

	return decodeStringInBytes, nil
}

func IsTokenValid(token string, now time.Time) error {

	if token == "" {
		return fmt.Errorf("empty token")
	}

	type TokenPayload struct {
		Exp int64 `json:"exp"`
		Iat int64 `json:"iat"`
		Nbf int64 `json:"nbf"`
	}

	// Split token by dot separator
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return errors.New("invalid token format")
	}

	// Decode payload (second part)
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return errors.New("failed to decode payload")
	}

	// Parse payload
	var claims TokenPayload
	if err := json.Unmarshal(payload, &claims); err != nil {
		return errors.New("failed to parse payload")
	}

	// Convert now to Unix timestamp
	nowUnix := now.Unix()

	// Check if token is expired
	if nowUnix > claims.Exp {
		return errors.New("token has expired")
	}

	// Check if token is not yet valid
	if nowUnix < claims.Nbf {
		return errors.New("token is not yet valid")
	}

	// Check if token was issued in the future
	if nowUnix < claims.Iat {
		return errors.New("token was issued in the future")
	}

	return nil
}
