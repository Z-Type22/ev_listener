package jwt

import (
	"crypto/rsa"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt"
)

var ErrRefreshTokenExpired = errors.New("token is expired")

func GetKey(KeyPath string) ([]byte, error) {
	key, err := os.ReadFile(KeyPath)
	if err != nil {
		return nil, err
	}

	return key, nil
}

func GetSigningMethod(publicKey *rsa.PublicKey) jwt.Keyfunc {
	return func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		return publicKey, nil
	}
}

func DecodeJWT(tokenString string, publicKeyPath string) (jwt.MapClaims, error) {
	secret, err := GetKey(publicKeyPath)
	if err != nil {
		return nil, err
	}

	publicKey, err := jwt.ParseRSAPublicKeyFromPEM(secret)
	if err != nil {
		return nil, err
	}

	token, err := jwt.Parse(tokenString, GetSigningMethod(publicKey))
	if err != nil {
		var ve *jwt.ValidationError

		if errors.As(err, &ve) {
			if ve.Errors&jwt.ValidationErrorExpired != 0 {
				return nil, ErrRefreshTokenExpired
			}
		}

		return nil, err
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	exp, ok := claims["exp"].(float64)
	if !ok || time.Now().Unix() > int64(exp) {
		return nil, ErrRefreshTokenExpired
	}

	return claims, nil
}
